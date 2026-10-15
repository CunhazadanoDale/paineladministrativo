package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/pedido"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ portsout.PedidoRepository = (*PedidoRepository)(nil)

type PedidoRepository struct {
	db *sqlx.DB
}

func NewPedidoRepository(db *sqlx.DB) *PedidoRepository {
	return &PedidoRepository{db: db}
}

var guardaDoPedido = guardaDeVersao{
	tabela:        "pedido",
	naoEncontrado: func() error { return domain.ErroNaoEncontrado("pedido não encontrado") },
	alterado:      domainpedido.ErroVersaoDivergente,
}

var guardaDaEntrega = guardaDeVersao{
	tabela:        "entrega",
	naoEncontrado: func() error { return domain.ErroNaoEncontrado("entrega não encontrada") },
	alterado:      domainpedido.ErroVersaoEntregaDivergente,
}

type pedidoLinha struct {
	ID                     uuid.UUID  `db:"id"`
	EmpresaID              uuid.UUID  `db:"empresa_id"`
	Numero                 int        `db:"numero"`
	ClienteID              uuid.UUID  `db:"cliente_id"`
	ObraID                 *uuid.UUID `db:"obra_id"`
	RepresentanteID        *uuid.UUID `db:"representante_id"`
	Modelo                 string     `db:"modelo"`
	EntradaBP              int        `db:"entrada_bp"`
	PrazoBoletoDias        *int       `db:"prazo_boleto_dias"`
	PrazoEntregaDias       *int       `db:"prazo_entrega_dias"`
	PrazoLimite            *time.Time `db:"prazo_limite"`
	NumeroPedidoCliente    *string    `db:"numero_pedido_cliente"`
	Observacoes            string     `db:"observacoes"`
	Status                 string     `db:"status"`
	ValorTotalCentavos     int64      `db:"valor_total_centavos"`
	EntradaPagaEm          *time.Time `db:"entrada_paga_em"`
	EntradaPagaCentavos    *int64     `db:"entrada_paga_centavos"`
	CanceladoMotivo        *string    `db:"cancelado_motivo"`
	ValorADevolverCentavos *int64     `db:"valor_a_devolver_centavos"`
	UsuarioID              uuid.UUID  `db:"usuario_id"`
	Versao                 int        `db:"versao"`
	CriadoEm               time.Time  `db:"criado_em"`
	AtualizadoEm           time.Time  `db:"atualizado_em"`
}

type pedidoItemLinha struct {
	ID                    uuid.UUID `db:"id"`
	PedidoID              uuid.UUID `db:"pedido_id"`
	ProdutoID             uuid.UUID `db:"produto_id"`
	Quantidade            int       `db:"quantidade"`
	PrecoUnitarioCentavos int64     `db:"preco_unitario_centavos"`
	QuantidadeEntregue    int       `db:"quantidade_entregue"`
	CriadoEm              time.Time `db:"criado_em"`
}

type pedidoHistoricoLinha struct {
	ID             uuid.UUID `db:"id"`
	PedidoID       uuid.UUID `db:"pedido_id"`
	UsuarioID      uuid.UUID `db:"usuario_id"`
	Acao           string    `db:"acao"`
	StatusAnterior *string   `db:"status_anterior"`
	StatusNovo     string    `db:"status_novo"`
	Detalhe        string    `db:"detalhe"`
	CriadoEm       time.Time `db:"criado_em"`
}

const colunasPedido = `
	id, empresa_id, numero, cliente_id, obra_id, representante_id, modelo, entrada_bp,
	prazo_boleto_dias, prazo_entrega_dias, prazo_limite, numero_pedido_cliente, observacoes,
	status, valor_total_centavos, entrada_paga_em, entrada_paga_centavos, cancelado_motivo,
	valor_a_devolver_centavos, usuario_id, versao, criado_em, atualizado_em
`

const colunasPedidoItem = `
	id, pedido_id, produto_id, quantidade, preco_unitario_centavos, quantidade_entregue, criado_em
`

func (r *PedidoRepository) Gravar(ctx context.Context, gravacao *portsout.Gravacao) error {
	if gravacao == nil || gravacao.Pedido == nil {
		return domain.ErroValidacao("gravação de pedido sem pedido")
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	pedido := gravacao.Pedido
	numero := pedido.Numero
	versao := pedido.Versao
	if gravacao.Novo {
		if numero, err = inserirPedido(ctx, tx, pedido); err != nil {
			return err
		}
	} else if versao, err = atualizarPedido(ctx, tx, gravacao); err != nil {
		return err
	}

	if err := gravarItens(ctx, tx, gravacao); err != nil {
		return err
	}
	for _, entrega := range gravacao.EntregasNovas {
		if err := inserirEntrega(ctx, tx, entrega); err != nil {
			return err
		}
	}
	versoesDasEntregas := make([]int, len(gravacao.EntregasAlteradas))
	for indice, alterada := range gravacao.EntregasAlteradas {
		if versoesDasEntregas[indice], err = atualizarEntrega(ctx, tx, alterada); err != nil {
			return err
		}
	}
	for _, operacao := range gravacao.OperacoesEstoque {
		if err := gravarOperacaoEstoque(ctx, tx, operacao); err != nil {
			return err
		}
	}
	for _, alterada := range gravacao.EntregasAlteradas {
		if err := vincularMovimentos(ctx, tx, alterada.Entrega); err != nil {
			return err
		}
	}
	for _, historico := range gravacao.Historicos {
		if err := inserirHistoricoDoPedido(ctx, tx, historico); err != nil {
			return err
		}
	}

	if err := tratarErroDeGravacao(tx.Commit()); err != nil {
		return err
	}

	pedido.Numero = numero
	pedido.Versao = versao
	for indice, alterada := range gravacao.EntregasAlteradas {
		alterada.Entrega.Versao = versoesDasEntregas[indice]
	}

	return nil
}

func (r *PedidoRepository) Obter(ctx context.Context, id uuid.UUID) (*domainpedido.Pedido, error) {
	var linha pedidoLinha
	if err := r.db.GetContext(ctx, &linha, `SELECT `+colunasPedido+` FROM pedido WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	var itens []*pedidoItemLinha
	if err := r.db.SelectContext(ctx, &itens,
		`SELECT `+colunasPedidoItem+` FROM pedido_item WHERE pedido_id = $1 ORDER BY id ASC`, id,
	); err != nil {
		return nil, err
	}

	pedido := paraDominioPedido(&linha)
	pedido.Itens = make([]*domainpedido.Item, 0, len(itens))
	for _, item := range itens {
		pedido.Itens = append(pedido.Itens, paraDominioPedidoItem(item))
	}

	return pedido, nil
}

func (r *PedidoRepository) Listar(ctx context.Context, filtro portsout.PedidoFiltro) ([]*domainpedido.Pedido, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	condicoes := make([]string, 0, 7)
	argumentos := make([]any, 0, 9)
	adicionar := func(coluna string, valor any) {
		argumentos = append(argumentos, valor)
		condicoes = append(condicoes, condicao(coluna, len(argumentos)))
	}

	if filtro.Empresa != nil {
		adicionar("empresa_id", *filtro.Empresa)
	}
	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		condicoes = append(condicoes, "empresa_id = ANY($"+strconv.Itoa(len(argumentos))+"::uuid[])")
	}
	if filtro.ClienteID != nil {
		adicionar("cliente_id", *filtro.ClienteID)
	}
	if filtro.RepresentanteID != nil {
		adicionar("representante_id", *filtro.RepresentanteID)
	}
	if filtro.Status != "" {
		adicionar("status", filtro.Status.String())
	}
	if filtro.Modelo != "" {
		adicionar("modelo", filtro.Modelo.String())
	}
	if filtro.Numero != nil {
		adicionar("numero", *filtro.Numero)
	}

	query := `SELECT ` + colunasPedido + ` FROM pedido`
	if len(condicoes) > 0 {
		query += ` WHERE ` + unirCondicoes(condicoes)
	}
	argumentos = append(argumentos, filtro.Size, offset)
	query += ` ORDER BY criado_em DESC, id DESC` + limiteOffset(argumentos)

	var linhas []*pedidoLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	pedidos := make([]*domainpedido.Pedido, 0, len(linhas))
	for _, linha := range linhas {
		pedidos = append(pedidos, paraDominioPedido(linha))
	}

	return pedidos, nil
}

func (r *PedidoRepository) ListarHistorico(ctx context.Context, pedidoID uuid.UUID) ([]*domainpedido.Historico, error) {
	var linhas []*pedidoHistoricoLinha
	if err := r.db.SelectContext(ctx, &linhas, `
		SELECT id, pedido_id, usuario_id, acao, status_anterior, status_novo, detalhe, criado_em
		FROM pedido_historico
		WHERE pedido_id = $1
		ORDER BY criado_em ASC, id ASC
	`, pedidoID); err != nil {
		return nil, err
	}

	historicos := make([]*domainpedido.Historico, 0, len(linhas))
	for _, linha := range linhas {
		historico := &domainpedido.Historico{
			ID:         linha.ID,
			PedidoID:   linha.PedidoID,
			UsuarioID:  linha.UsuarioID,
			Acao:       domainpedido.AcaoDe(linha.Acao),
			StatusNovo: domainpedido.StatusDe(linha.StatusNovo),
			Detalhe:    linha.Detalhe,
			CriadoEm:   linha.CriadoEm,
		}
		if linha.StatusAnterior != nil {
			anterior := domainpedido.StatusDe(*linha.StatusAnterior)
			historico.StatusAnterior = &anterior
		}
		historicos = append(historicos, historico)
	}

	return historicos, nil
}

func inserirPedido(ctx context.Context, tx *sqlx.Tx, pedido *domainpedido.Pedido) (int, error) {
	var numero int
	if err := tx.GetContext(ctx, &numero, `
		INSERT INTO pedido_sequencia (empresa_id, ultimo_numero)
		VALUES ($1, 1)
		ON CONFLICT (empresa_id) DO UPDATE
		SET ultimo_numero = pedido_sequencia.ultimo_numero + 1
		RETURNING ultimo_numero
	`, pedido.EmpresaID); err != nil {
		return 0, tratarErroDeGravacao(err)
	}

	linha := paraLinhaPedido(pedido)
	linha.Numero = numero
	if _, err := tx.NamedExecContext(ctx, `
		INSERT INTO pedido (`+colunasPedido+`)
		VALUES (
			:id, :empresa_id, :numero, :cliente_id, :obra_id, :representante_id, :modelo, :entrada_bp,
			:prazo_boleto_dias, :prazo_entrega_dias, :prazo_limite, :numero_pedido_cliente, :observacoes,
			:status, :valor_total_centavos, :entrada_paga_em, :entrada_paga_centavos, :cancelado_motivo,
			:valor_a_devolver_centavos, :usuario_id, :versao, :criado_em, :atualizado_em
		)
	`, linha); err != nil {
		return 0, tratarErroDeGravacao(err)
	}

	return numero, nil
}

func atualizarPedido(ctx context.Context, tx *sqlx.Tx, gravacao *portsout.Gravacao) (int, error) {
	linha := paraLinhaPedido(gravacao.Pedido)

	return guardaDoPedido.atualizar(ctx, tx, linha.ID, `
		UPDATE pedido
		SET cliente_id = $1,
		    obra_id = $2,
		    representante_id = $3,
		    modelo = $4,
		    entrada_bp = $5,
		    prazo_boleto_dias = $6,
		    prazo_entrega_dias = $7,
		    prazo_limite = $8,
		    numero_pedido_cliente = $9,
		    observacoes = $10,
		    status = $11,
		    valor_total_centavos = $12,
		    entrada_paga_em = $13,
		    entrada_paga_centavos = $14,
		    cancelado_motivo = $15,
		    valor_a_devolver_centavos = $16,
		    atualizado_em = $17,
		    versao = versao + 1
		WHERE id = $18
		  AND versao = $19
		  AND status = $20
		RETURNING versao
	`,
		linha.ClienteID, linha.ObraID, linha.RepresentanteID, linha.Modelo, linha.EntradaBP,
		linha.PrazoBoletoDias, linha.PrazoEntregaDias, linha.PrazoLimite, linha.NumeroPedidoCliente,
		linha.Observacoes, linha.Status, linha.ValorTotalCentavos, linha.EntradaPagaEm,
		linha.EntradaPagaCentavos, linha.CanceladoMotivo, linha.ValorADevolverCentavos, linha.AtualizadoEm,
		linha.ID, gravacao.VersaoLida, gravacao.StatusAnterior.String(),
	)
}

func gravarItens(ctx context.Context, tx *sqlx.Tx, gravacao *portsout.Gravacao) error {
	pedido := gravacao.Pedido

	if !gravacao.Novo && !gravacao.SubstituirItens {
		for _, item := range pedido.Itens {
			if _, err := tx.ExecContext(ctx, `
				UPDATE pedido_item
				SET quantidade_entregue = $1
				WHERE id = $2
				  AND pedido_id = $3
			`, item.QuantidadeEntregue, item.ID, pedido.ID); err != nil {
				return tratarErroDeGravacao(err)
			}
		}

		return nil
	}

	if gravacao.SubstituirItens && !gravacao.Novo {
		if _, err := tx.ExecContext(ctx, `DELETE FROM pedido_item WHERE pedido_id = $1`, pedido.ID); err != nil {
			return tratarErroDeGravacao(err)
		}
	}

	for _, item := range pedido.Itens {
		if _, err := tx.NamedExecContext(ctx, `
			INSERT INTO pedido_item (`+colunasPedidoItem+`)
			VALUES (:id, :pedido_id, :produto_id, :quantidade, :preco_unitario_centavos, :quantidade_entregue, :criado_em)
		`, paraLinhaPedidoItem(pedido.ID, item)); err != nil {
			return tratarErroDeGravacao(err)
		}
	}

	return nil
}

func inserirHistoricoDoPedido(ctx context.Context, tx *sqlx.Tx, historico *domainpedido.Historico) error {
	var anterior *string
	if historico.StatusAnterior != nil {
		valor := historico.StatusAnterior.String()
		anterior = &valor
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO pedido_historico (id, pedido_id, usuario_id, acao, status_anterior, status_novo, detalhe, criado_em)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		historico.ID, historico.PedidoID, historico.UsuarioID, historico.Acao.String(),
		anterior, historico.StatusNovo.String(), historico.Detalhe, historico.CriadoEm,
	); err != nil {
		return tratarErroDeGravacao(err)
	}

	return nil
}

func paraLinhaPedido(pedido *domainpedido.Pedido) pedidoLinha {
	return pedidoLinha{
		ID:                     pedido.ID,
		EmpresaID:              pedido.EmpresaID,
		Numero:                 pedido.Numero,
		ClienteID:              pedido.ClienteID,
		ObraID:                 pedido.ObraID,
		RepresentanteID:        pedido.RepresentanteID,
		Modelo:                 pedido.Modelo.String(),
		EntradaBP:              pedido.Condicao.Entrada().PontosBase(),
		PrazoBoletoDias:        pedido.Condicao.PrazoBoletoDias(),
		PrazoEntregaDias:       pedido.Condicao.PrazoEntregaDias(),
		PrazoLimite:            pedido.PrazoLimite,
		NumeroPedidoCliente:    pedido.NumeroPedidoCliente,
		Observacoes:            pedido.Observacoes,
		Status:                 pedido.Status.String(),
		ValorTotalCentavos:     pedido.ValorTotalCentavos,
		EntradaPagaEm:          pedido.EntradaPagaEm,
		EntradaPagaCentavos:    pedido.EntradaPagaCentavos,
		CanceladoMotivo:        pedido.CanceladoMotivo,
		ValorADevolverCentavos: pedido.ValorADevolverCentavos,
		UsuarioID:              pedido.UsuarioID,
		Versao:                 pedido.Versao,
		CriadoEm:               pedido.CriadoEm,
		AtualizadoEm:           pedido.AtualizadoEm,
	}
}

func paraLinhaPedidoItem(pedidoID uuid.UUID, item *domainpedido.Item) pedidoItemLinha {
	return pedidoItemLinha{
		ID:                    item.ID,
		PedidoID:              pedidoID,
		ProdutoID:             item.ProdutoID,
		Quantidade:            item.Quantidade,
		PrecoUnitarioCentavos: item.PrecoUnitario.Centavos(),
		QuantidadeEntregue:    item.QuantidadeEntregue,
		CriadoEm:              item.CriadoEm,
	}
}

func paraDominioPedido(linha *pedidoLinha) *domainpedido.Pedido {
	return &domainpedido.Pedido{
		ID:                     linha.ID,
		EmpresaID:              linha.EmpresaID,
		Numero:                 linha.Numero,
		ClienteID:              linha.ClienteID,
		ObraID:                 linha.ObraID,
		RepresentanteID:        linha.RepresentanteID,
		Modelo:                 domainpedido.ModeloDe(linha.Modelo),
		Condicao:               domainpedido.CondicaoDe(linha.EntradaBP, linha.PrazoBoletoDias, linha.PrazoEntregaDias),
		PrazoLimite:            dataDoBanco(linha.PrazoLimite),
		NumeroPedidoCliente:    linha.NumeroPedidoCliente,
		Observacoes:            linha.Observacoes,
		Status:                 domainpedido.StatusDe(linha.Status),
		ValorTotalCentavos:     linha.ValorTotalCentavos,
		EntradaPagaEm:          dataDoBanco(linha.EntradaPagaEm),
		EntradaPagaCentavos:    linha.EntradaPagaCentavos,
		CanceladoMotivo:        linha.CanceladoMotivo,
		ValorADevolverCentavos: linha.ValorADevolverCentavos,
		UsuarioID:              linha.UsuarioID,
		Versao:                 linha.Versao,
		CriadoEm:               linha.CriadoEm,
		AtualizadoEm:           linha.AtualizadoEm,
	}
}

func paraDominioPedidoItem(linha *pedidoItemLinha) *domainpedido.Item {
	return &domainpedido.Item{
		ID:                 linha.ID,
		PedidoID:           linha.PedidoID,
		ProdutoID:          linha.ProdutoID,
		Quantidade:         linha.Quantidade,
		PrecoUnitario:      domainpedido.PrecoDe(linha.PrecoUnitarioCentavos),
		QuantidadeEntregue: linha.QuantidadeEntregue,
		CriadoEm:           linha.CriadoEm,
	}
}

func dataDoBanco(valor *time.Time) *time.Time {
	if valor == nil {
		return nil
	}

	data := domainpedido.DataDe(*valor)

	return &data
}
