package postgres

import (
	"context"
	"strconv"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/importacao"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type importacaoAlocacaoLinha struct {
	ImportacaoItemID uuid.UUID  `db:"importacao_item_id"`
	PedidoItemID     uuid.UUID  `db:"pedido_item_id"`
	Quantidade       int        `db:"quantidade"`
	Reservada        *int       `db:"reservada"`
	ReservaID        *uuid.UUID `db:"reserva_id"`
	UsuarioID        uuid.UUID  `db:"usuario_id"`
	CriadoEm         time.Time  `db:"criado_em"`
	AtualizadoEm     time.Time  `db:"atualizado_em"`
}

type pedidoItemAlocavelLinha struct {
	PedidoItemID       uuid.UUID  `db:"pedido_item_id"`
	PedidoID           uuid.UUID  `db:"pedido_id"`
	PedidoNumero       int        `db:"pedido_numero"`
	EmpresaID          uuid.UUID  `db:"empresa_id"`
	Modelo             string     `db:"modelo"`
	StatusPedido       string     `db:"status_pedido"`
	PrazoLimite        *time.Time `db:"prazo_limite"`
	ProdutoID          uuid.UUID  `db:"produto_id"`
	Quantidade         int        `db:"quantidade"`
	QuantidadeEntregue int        `db:"quantidade_entregue"`
	Reservado          int        `db:"reservado"`
	AlocadoEmOutras    int        `db:"alocado_em_outras"`
}

const alocadoEmImportacoesAbertas = `
	SELECT COALESCE(SUM(a.quantidade), 0)
	FROM importacao_alocacao a
	JOIN importacao_item ii ON ii.id = a.importacao_item_id
	JOIN importacao i ON i.id = ii.importacao_id
	WHERE a.pedido_item_id = pi.id
	  AND i.status NOT IN ('cancelada', 'recebida')
`

func (r *ImportacaoRepository) ItensDePedido(
	ctx context.Context, pedidoItemIDs []uuid.UUID, excetoImportacaoID uuid.UUID,
) ([]*domainimportacao.PedidoItemAlocavel, error) {
	if len(pedidoItemIDs) == 0 {
		return []*domainimportacao.PedidoItemAlocavel{}, nil
	}

	var linhas []*pedidoItemAlocavelLinha
	if err := r.db.SelectContext(ctx, &linhas, `
		SELECT pi.id AS pedido_item_id, p.id AS pedido_id, p.numero AS pedido_numero, p.empresa_id, p.modelo,
		       p.status AS status_pedido, p.prazo_limite, pi.produto_id, pi.quantidade, pi.quantidade_entregue,
		       COALESCE((
		           SELECT SUM(r.quantidade) FROM produto_reserva r
		           WHERE r.origem = 'pedido_item' AND r.origem_id = pi.id AND r.status = 'ativa'
		       ), 0) AS reservado,
		       (`+alocadoEmImportacoesAbertas+` AND i.id <> $2) AS alocado_em_outras
		FROM pedido_item pi
		JOIN pedido p ON p.id = pi.pedido_id
		WHERE pi.id = ANY($1::uuid[])
		ORDER BY pi.id
	`, pq.Array(textosDeIDs(pedidoItemIDs)), excetoImportacaoID); err != nil {
		return nil, err
	}

	itens := make([]*domainimportacao.PedidoItemAlocavel, 0, len(linhas))
	for _, linha := range linhas {
		itens = append(itens, &domainimportacao.PedidoItemAlocavel{
			PedidoItemID:       linha.PedidoItemID,
			PedidoID:           linha.PedidoID,
			PedidoNumero:       linha.PedidoNumero,
			EmpresaID:          linha.EmpresaID,
			Modelo:             domainpedido.ModeloDe(linha.Modelo),
			StatusPedido:       domainpedido.StatusDe(linha.StatusPedido),
			PrazoLimite:        dataDaImportacao(linha.PrazoLimite),
			ProdutoID:          linha.ProdutoID,
			Quantidade:         linha.Quantidade,
			QuantidadeEntregue: linha.QuantidadeEntregue,
			Reservado:          linha.Reservado,
			AlocadoEmOutras:    linha.AlocadoEmOutras,
		})
	}

	return itens, nil
}

func carregarAlocacoes(ctx context.Context, tx *sqlx.Tx, importacao *domainimportacao.Importacao) error {
	var linhas []*importacaoAlocacaoLinha
	if err := tx.SelectContext(ctx, &linhas, `
		SELECT a.importacao_item_id, a.pedido_item_id, a.quantidade, a.reservada, a.reserva_id, a.usuario_id,
		       a.criado_em, a.atualizado_em
		FROM importacao_alocacao a
		JOIN importacao_item ii ON ii.id = a.importacao_item_id
		WHERE ii.importacao_id = $1
		ORDER BY a.criado_em ASC, a.importacao_item_id ASC, a.pedido_item_id ASC
	`, importacao.ID); err != nil {
		return err
	}

	importacao.Alocacoes = make([]*domainimportacao.Alocacao, 0, len(linhas))
	for _, linha := range linhas {
		importacao.Alocacoes = append(importacao.Alocacoes, &domainimportacao.Alocacao{
			ImportacaoItemID: linha.ImportacaoItemID,
			PedidoItemID:     linha.PedidoItemID,
			Quantidade:       linha.Quantidade,
			Reservada:        linha.Reservada,
			ReservaID:        linha.ReservaID,
			UsuarioID:        linha.UsuarioID,
			CriadoEm:         linha.CriadoEm,
			AtualizadoEm:     linha.AtualizadoEm,
		})
	}

	return nil
}

func gravarAlocacoesERecebimento(ctx context.Context, tx *sqlx.Tx, gravacao *portsout.Gravacao) error {
	if gravacao.SubstituirAlocacoes {
		if err := substituirAlocacoes(ctx, tx, gravacao.Importacao); err != nil {
			return err
		}
	}
	if !gravacao.Recebimento {
		return nil
	}

	statusDosPedidos, err := exigirPedidosEmAtendimento(ctx, tx, gravacao.PedidosAtendidos)
	if err != nil {
		return err
	}
	if err := travarProdutosEmOrdem(ctx, tx, gravacao.OperacoesEstoque); err != nil {
		return err
	}
	for _, operacao := range gravacao.OperacoesEstoque {
		if err := gravarOperacaoEstoque(ctx, tx, operacao); err != nil {
			return err
		}
	}
	for _, item := range gravacao.Importacao.Itens {
		if _, err := tx.ExecContext(ctx, `
			UPDATE importacao_item
			SET quantidade_recebida = $1, custo_unitario_centavos = $2, movimento_id = $3
			WHERE id = $4 AND importacao_id = $5
		`, item.QuantidadeRecebida, custoGravado(item), item.MovimentoID, item.ID, gravacao.Importacao.ID); err != nil {
			return tratarErroDeGravacao(err)
		}
	}
	for _, alocacao := range gravacao.Importacao.Alocacoes {
		if _, err := tx.ExecContext(ctx, `
			UPDATE importacao_alocacao
			SET reservada = $1, reserva_id = $2, atualizado_em = $3
			WHERE importacao_item_id = $4 AND pedido_item_id = $5
		`, alocacao.Reservada, alocacao.ReservaID, gravacao.Importacao.AtualizadoEm, alocacao.ImportacaoItemID, alocacao.PedidoItemID); err != nil {
			return tratarErroDeGravacao(err)
		}
	}
	if err := exigirReservasNoRestante(ctx, tx, gravacao.Importacao.Alocacoes); err != nil {
		return err
	}
	for _, historico := range gravacao.HistoricosDePedido {
		if status, ok := statusDosPedidos[historico.PedidoID]; ok {
			historico.StatusNovo = domainpedido.StatusDe(status)
		}
		if err := inserirHistoricoDoPedido(ctx, tx, historico); err != nil {
			return err
		}
	}

	return nil
}

func travarProdutosEmOrdem(ctx context.Context, tx *sqlx.Tx, operacoes []*domainestoque.OperacaoEstoque) error {
	if len(operacoes) == 0 {
		return nil
	}

	ids := make([]uuid.UUID, 0, len(operacoes))
	for _, operacao := range operacoes {
		ids = append(ids, operacao.ProdutoID)
	}
	if _, err := tx.ExecContext(ctx,
		`SELECT id FROM produto WHERE id = ANY($1::uuid[]) ORDER BY id FOR UPDATE`, pq.Array(textosDeIDs(ids)),
	); err != nil {
		return tratarErroDeGravacao(err)
	}

	return nil
}

func exigirReservasNoRestante(ctx context.Context, tx *sqlx.Tx, alocacoes []*domainimportacao.Alocacao) error {
	ids := make([]uuid.UUID, 0, len(alocacoes))
	for _, alocacao := range alocacoes {
		if alocacao.ReservaID != nil {
			ids = append(ids, alocacao.PedidoItemID)
		}
	}
	if len(ids) == 0 {
		return nil
	}

	var excedidos []int
	if err := tx.SelectContext(ctx, &excedidos, `
		SELECT p.numero
		FROM pedido_item pi
		JOIN pedido p ON p.id = pi.pedido_id
		WHERE pi.id = ANY($1::uuid[])
		  AND (
		      SELECT COALESCE(SUM(r.quantidade), 0) FROM produto_reserva r
		      WHERE r.origem = 'pedido_item' AND r.origem_id = pi.id AND r.status = 'ativa'
		  ) > pi.quantidade - pi.quantidade_entregue
		ORDER BY p.numero
	`, pq.Array(textosDeIDs(ids))); err != nil {
		return tratarErroDeGravacao(err)
	}
	if len(excedidos) > 0 {
		return domain.ErroConflito("o pedido " + strconv.Itoa(excedidos[0]) +
			" mudou durante o recebimento e ficaria com reserva acima do que falta entregar; recarregue e tente de novo")
	}

	return nil
}

func custoGravado(item *domainimportacao.Item) *int64 {
	if item.MovimentoID == nil {
		return nil
	}

	return item.CustoUnitarioCentavos
}

func substituirAlocacoes(ctx context.Context, tx *sqlx.Tx, importacao *domainimportacao.Importacao) error {
	var anteriores []uuid.UUID
	if err := tx.SelectContext(ctx, &anteriores, `
		SELECT a.pedido_item_id
		FROM importacao_alocacao a
		JOIN importacao_item ii ON ii.id = a.importacao_item_id
		WHERE ii.importacao_id = $1
	`, importacao.ID); err != nil {
		return tratarErroDeGravacao(err)
	}

	ids := append([]uuid.UUID(nil), anteriores...)
	for _, alocacao := range importacao.Alocacoes {
		ids = append(ids, alocacao.PedidoItemID)
	}
	if len(ids) > 0 {
		var pedidoIDs []uuid.UUID
		if err := tx.SelectContext(ctx, &pedidoIDs,
			`SELECT DISTINCT pedido_id FROM pedido_item WHERE id = ANY($1::uuid[])`, pq.Array(textosDeIDs(ids)),
		); err != nil {
			return tratarErroDeGravacao(err)
		}
		if _, err := travarPedidos(ctx, tx, pedidoIDs); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`SELECT id FROM pedido_item WHERE id = ANY($1::uuid[]) ORDER BY id FOR UPDATE`, pq.Array(textosDeIDs(ids)),
		); err != nil {
			return tratarErroDeGravacao(err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM importacao_alocacao a
		USING importacao_item ii
		WHERE ii.id = a.importacao_item_id AND ii.importacao_id = $1
	`, importacao.ID); err != nil {
		return tratarErroDeGravacao(err)
	}
	for _, alocacao := range importacao.Alocacoes {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO importacao_alocacao (importacao_item_id, pedido_item_id, quantidade, usuario_id, criado_em, atualizado_em)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, alocacao.ImportacaoItemID, alocacao.PedidoItemID, alocacao.Quantidade, alocacao.UsuarioID,
			alocacao.CriadoEm, alocacao.AtualizadoEm); err != nil {
			return tratarErroDeGravacao(err)
		}
	}

	if len(importacao.Alocacoes) == 0 {
		return nil
	}

	novos := make([]uuid.UUID, 0, len(importacao.Alocacoes))
	for _, alocacao := range importacao.Alocacoes {
		novos = append(novos, alocacao.PedidoItemID)
	}
	var foraDeAtendimento []int
	if err := tx.SelectContext(ctx, &foraDeAtendimento, `
		SELECT DISTINCT p.numero
		FROM pedido_item pi
		JOIN pedido p ON p.id = pi.pedido_id
		WHERE pi.id = ANY($1::uuid[])
		  AND p.status NOT IN ('confirmado', 'parcialmente_entregue')
		ORDER BY p.numero
	`, pq.Array(textosDeIDs(novos))); err != nil {
		return tratarErroDeGravacao(err)
	}
	if len(foraDeAtendimento) > 0 {
		return domain.ErroConflito("o pedido " + strconv.Itoa(foraDeAtendimento[0]) +
			" deixou de estar em atendimento; recarregue e tente de novo")
	}

	var excedidos []int
	if err := tx.SelectContext(ctx, &excedidos, `
		SELECT p.numero
		FROM pedido_item pi
		JOIN pedido p ON p.id = pi.pedido_id
		WHERE pi.id = ANY($1::uuid[])
		  AND (`+alocadoEmImportacoesAbertas+`) > pi.quantidade - pi.quantidade_entregue
		ORDER BY p.numero
	`, pq.Array(textosDeIDs(ids))); err != nil {
		return tratarErroDeGravacao(err)
	}
	if len(excedidos) > 0 {
		return domain.ErroConflito("as alocações do pedido " + strconv.Itoa(excedidos[0]) +
			" passam do que falta entregar; outra importação foi alocada ao mesmo tempo, recarregue e tente de novo")
	}

	return nil
}

type pedidoTravadoLinha struct {
	ID     uuid.UUID `db:"id"`
	Numero int       `db:"numero"`
	Status string    `db:"status"`
}

func travarPedidos(ctx context.Context, tx *sqlx.Tx, pedidoIDs []uuid.UUID) ([]*pedidoTravadoLinha, error) {
	var travados []*pedidoTravadoLinha
	if err := tx.SelectContext(ctx, &travados,
		`SELECT id, numero, status FROM pedido WHERE id = ANY($1::uuid[]) ORDER BY id FOR SHARE`, pq.Array(textosDeIDs(pedidoIDs)),
	); err != nil {
		return nil, tratarErroDeGravacao(err)
	}

	return travados, nil
}

func exigirPedidosEmAtendimento(ctx context.Context, tx *sqlx.Tx, pedidoIDs []uuid.UUID) (map[uuid.UUID]string, error) {
	statusDosPedidos := make(map[uuid.UUID]string, len(pedidoIDs))
	if len(pedidoIDs) == 0 {
		return statusDosPedidos, nil
	}

	travados, err := travarPedidos(ctx, tx, pedidoIDs)
	if err != nil {
		return nil, err
	}
	for _, pedido := range travados {
		if !domainpedido.StatusDe(pedido.Status).EmAtendimento() {
			return nil, domain.ErroConflito("o pedido " + strconv.Itoa(pedido.Numero) +
				" deixou de estar em atendimento durante o recebimento; recarregue e tente de novo")
		}
		statusDosPedidos[pedido.ID] = pedido.Status
	}

	return statusDosPedidos, nil
}
