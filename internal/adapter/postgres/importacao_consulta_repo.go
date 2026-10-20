package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/importacao"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ portsout.ConsultaRepository = (*ImportacaoConsultaRepository)(nil)

type ImportacaoConsultaRepository struct {
	db *sqlx.DB
}

func NewImportacaoConsultaRepository(db *sqlx.DB) *ImportacaoConsultaRepository {
	return &ImportacaoConsultaRepository{db: db}
}

type alocacaoAbertaLinha struct {
	ImportacaoID     uuid.UUID  `db:"importacao_id"`
	ImportacaoNumero int        `db:"importacao_numero"`
	ImportacaoStatus string     `db:"importacao_status"`
	EmpresaID        uuid.UUID  `db:"empresa_id"`
	ChegadaPrevista  *time.Time `db:"chegada_prevista"`
	PedidoID         uuid.UUID  `db:"pedido_id"`
	PedidoNumero     int        `db:"pedido_numero"`
	PedidoItemID     uuid.UUID  `db:"pedido_item_id"`
	ProdutoID        uuid.UUID  `db:"produto_id"`
	Quantidade       int        `db:"quantidade"`
	PrazoLimite      *time.Time `db:"prazo_limite"`
}

type pedidoComFaltaLinha struct {
	PedidoID     uuid.UUID  `db:"pedido_id"`
	PedidoNumero int        `db:"pedido_numero"`
	EmpresaID    uuid.UUID  `db:"empresa_id"`
	PedidoItemID uuid.UUID  `db:"pedido_item_id"`
	ProdutoID    uuid.UUID  `db:"produto_id"`
	PrazoLimite  *time.Time `db:"prazo_limite"`
	Restante     int        `db:"restante"`
	Reservado    int        `db:"reservado"`
	Alocado      int        `db:"alocado"`
}

type atendimentoLinha struct {
	ImportacaoID     uuid.UUID  `db:"importacao_id"`
	ImportacaoNumero int        `db:"importacao_numero"`
	ImportacaoStatus string     `db:"importacao_status"`
	ChegadaPrevista  *time.Time `db:"chegada_prevista"`
	RecebidaEm       *time.Time `db:"recebida_em"`
	PedidoItemID     uuid.UUID  `db:"pedido_item_id"`
	ProdutoID        uuid.UUID  `db:"produto_id"`
	Quantidade       int        `db:"quantidade"`
	Reservada        *int       `db:"reservada"`
}

func filtroDeEmpresaDaImportacao(coluna string, filtro portsout.AlertasFiltro, argumentos []any) (string, []any) {
	condicoes := ""
	if filtro.Empresa != nil {
		argumentos = append(argumentos, *filtro.Empresa)
		condicoes += " AND " + coluna + " = $" + strconv.Itoa(len(argumentos))
	}
	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		condicoes += " AND " + coluna + " = ANY($" + strconv.Itoa(len(argumentos)) + "::uuid[])"
	}

	return condicoes, argumentos
}

func (r *ImportacaoConsultaRepository) AlocacoesAbertas(ctx context.Context, filtro portsout.AlertasFiltro) ([]domainimportacao.AlocacaoAberta, error) {
	condicoes, argumentos := filtroDeEmpresaDaImportacao("i.empresa_id", filtro, nil)

	var linhas []*alocacaoAbertaLinha
	if err := r.db.SelectContext(ctx, &linhas, `
		SELECT i.id AS importacao_id, i.numero AS importacao_numero, i.status AS importacao_status, i.empresa_id,
		       i.chegada_prevista, p.id AS pedido_id, p.numero AS pedido_numero, pi.id AS pedido_item_id, pi.produto_id,
		       a.quantidade, p.prazo_limite
		FROM importacao_alocacao a
		JOIN importacao_item ii ON ii.id = a.importacao_item_id
		JOIN importacao i ON i.id = ii.importacao_id
		JOIN pedido_item pi ON pi.id = a.pedido_item_id
		JOIN pedido p ON p.id = pi.pedido_id
		WHERE i.status IN ('aberta', 'embarcada', 'desembaracada')
		  AND p.status IN ('confirmado', 'parcialmente_entregue')`+condicoes+`
		ORDER BY p.prazo_limite ASC NULLS LAST, i.numero ASC, p.numero ASC, pi.id ASC
	`, argumentos...); err != nil {
		return nil, err
	}

	alocacoes := make([]domainimportacao.AlocacaoAberta, 0, len(linhas))
	for _, linha := range linhas {
		alocacoes = append(alocacoes, domainimportacao.AlocacaoAberta{
			ImportacaoID:     linha.ImportacaoID,
			ImportacaoNumero: linha.ImportacaoNumero,
			ImportacaoStatus: domainimportacao.StatusDe(linha.ImportacaoStatus),
			EmpresaID:        linha.EmpresaID,
			ChegadaPrevista:  dataDaImportacao(linha.ChegadaPrevista),
			PedidoID:         linha.PedidoID,
			PedidoNumero:     linha.PedidoNumero,
			PedidoItemID:     linha.PedidoItemID,
			ProdutoID:        linha.ProdutoID,
			Quantidade:       linha.Quantidade,
			PrazoLimite:      dataDaImportacao(linha.PrazoLimite),
		})
	}

	return alocacoes, nil
}

func (r *ImportacaoConsultaRepository) PedidosComFalta(ctx context.Context, filtro portsout.AlertasFiltro) ([]domainimportacao.PedidoSemImportacao, error) {
	condicoes, argumentos := filtroDeEmpresaDaImportacao("p.empresa_id", filtro, nil)

	var linhas []*pedidoComFaltaLinha
	if err := r.db.SelectContext(ctx, &linhas, `
		SELECT pedido_id, pedido_numero, empresa_id, pedido_item_id, produto_id, prazo_limite, restante, reservado, alocado
		FROM (
			SELECT p.id AS pedido_id, p.numero AS pedido_numero, p.empresa_id, pi.id AS pedido_item_id, pi.produto_id,
			       p.prazo_limite, pi.quantidade - pi.quantidade_entregue AS restante,
			       COALESCE((
			           SELECT SUM(r.quantidade)::int FROM produto_reserva r
			           WHERE r.origem = 'pedido_item' AND r.origem_id = pi.id AND r.status = 'ativa'
			       ), 0) AS reservado,
			       (`+alocadoEmImportacoesAbertas+`)::int AS alocado
			FROM pedido_item pi
			JOIN pedido p ON p.id = pi.pedido_id
			WHERE p.modelo = 'importacao'
			  AND p.status IN ('confirmado', 'parcialmente_entregue')
			  AND pi.quantidade > pi.quantidade_entregue`+condicoes+`
		) itens
		WHERE restante - reservado - alocado > 0
		ORDER BY prazo_limite ASC NULLS LAST, pedido_numero ASC, pedido_item_id ASC
	`, argumentos...); err != nil {
		return nil, err
	}

	pedidos := make([]domainimportacao.PedidoSemImportacao, 0, len(linhas))
	for _, linha := range linhas {
		pedidos = append(pedidos, domainimportacao.PedidoSemImportacao{
			PedidoID:     linha.PedidoID,
			PedidoNumero: linha.PedidoNumero,
			EmpresaID:    linha.EmpresaID,
			PedidoItemID: linha.PedidoItemID,
			ProdutoID:    linha.ProdutoID,
			PrazoLimite:  dataDaImportacao(linha.PrazoLimite),
			Restante:     linha.Restante,
			Reservado:    linha.Reservado,
			Alocado:      linha.Alocado,
		})
	}

	return pedidos, nil
}

func (r *ImportacaoConsultaRepository) EmpresaDoPedido(ctx context.Context, pedidoID uuid.UUID) (*uuid.UUID, error) {
	var empresaID uuid.UUID
	if err := r.db.GetContext(ctx, &empresaID, `SELECT empresa_id FROM pedido WHERE id = $1`, pedidoID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &empresaID, nil
}

func (r *ImportacaoConsultaRepository) AtendimentosDoPedido(ctx context.Context, pedidoID uuid.UUID) ([]domainimportacao.AtendimentoDoPedido, error) {
	var linhas []*atendimentoLinha
	if err := r.db.SelectContext(ctx, &linhas, `
		SELECT i.id AS importacao_id, i.numero AS importacao_numero, i.status AS importacao_status, i.chegada_prevista,
		       i.recebida_em, pi.id AS pedido_item_id, pi.produto_id, a.quantidade, a.reservada
		FROM importacao_alocacao a
		JOIN importacao_item ii ON ii.id = a.importacao_item_id
		JOIN importacao i ON i.id = ii.importacao_id
		JOIN pedido_item pi ON pi.id = a.pedido_item_id
		WHERE pi.pedido_id = $1
		  AND i.status <> 'cancelada'
		ORDER BY i.numero ASC, pi.id ASC
	`, pedidoID); err != nil {
		return nil, err
	}

	atendimentos := make([]domainimportacao.AtendimentoDoPedido, 0, len(linhas))
	for _, linha := range linhas {
		atendimentos = append(atendimentos, domainimportacao.AtendimentoDoPedido{
			ImportacaoID:     linha.ImportacaoID,
			ImportacaoNumero: linha.ImportacaoNumero,
			ImportacaoStatus: domainimportacao.StatusDe(linha.ImportacaoStatus),
			ChegadaPrevista:  dataDaImportacao(linha.ChegadaPrevista),
			RecebidaEm:       dataDaImportacao(linha.RecebidaEm),
			PedidoItemID:     linha.PedidoItemID,
			ProdutoID:        linha.ProdutoID,
			Quantidade:       linha.Quantidade,
			Reservada:        linha.Reservada,
		})
	}

	return atendimentos, nil
}
