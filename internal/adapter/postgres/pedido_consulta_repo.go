package postgres

import (
	"context"
	"strconv"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/pedido"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ portsout.ConsultaRepository = (*PedidoConsultaRepository)(nil)

type PedidoConsultaRepository struct {
	db *sqlx.DB
}

func NewPedidoConsultaRepository(db *sqlx.DB) *PedidoConsultaRepository {
	return &PedidoConsultaRepository{db: db}
}

type pendenciaLinha struct {
	PedidoID            uuid.UUID  `db:"pedido_id"`
	Numero              int        `db:"numero"`
	EmpresaID           uuid.UUID  `db:"empresa_id"`
	ClienteID           uuid.UUID  `db:"cliente_id"`
	RepresentanteID     *uuid.UUID `db:"representante_id"`
	Status              string     `db:"status"`
	PrazoLimite         *time.Time `db:"prazo_limite"`
	ItemID              uuid.UUID  `db:"item_id"`
	ProdutoID           uuid.UUID  `db:"produto_id"`
	ProdutoNome         string     `db:"produto_nome"`
	Vendido             int        `db:"vendido"`
	Entregue            int        `db:"entregue"`
	Programado          int        `db:"programado"`
	Reservado           int        `db:"reservado"`
	DisponivelNaEmpresa int        `db:"disponivel_na_empresa"`
}

type consultaDePedido struct {
	condicoes  []string
	argumentos []any
}

func (c *consultaDePedido) igual(coluna string, valor any) {
	c.argumentos = append(c.argumentos, valor)
	c.condicoes = append(c.condicoes, condicao(coluna, len(c.argumentos)))
}

func (c *consultaDePedido) comparar(coluna, operador string, valor any) {
	c.argumentos = append(c.argumentos, valor)
	c.condicoes = append(c.condicoes, coluna+" "+operador+" $"+strconv.Itoa(len(c.argumentos)))
}

func (c *consultaDePedido) escopo(escopo portsout.EscopoFiltro) {
	if escopo.Empresa != nil {
		c.igual("p.empresa_id", *escopo.Empresa)
	}
	if escopo.EmpresaIDs != nil {
		c.argumentos = append(c.argumentos, pq.Array(textosDeIDs(escopo.EmpresaIDs)))
		c.condicoes = append(c.condicoes, "p.empresa_id = ANY($"+strconv.Itoa(len(c.argumentos))+"::uuid[])")
	}
	if escopo.RepresentanteID != nil {
		c.igual("p.representante_id", *escopo.RepresentanteID)
	}
}

func (c *consultaDePedido) paginar(filtro domain.PaginacaoFiltro) string {
	pagina := filtro.Normalizada()
	c.argumentos = append(c.argumentos, pagina.Size, (pagina.Page-1)*pagina.Size)

	return limiteOffset(c.argumentos)
}

func (r *PedidoConsultaRepository) Pendencias(ctx context.Context, filtro portsout.PendenciaFiltro) ([]*domainpedido.Pendencia, error) {
	consulta := &consultaDePedido{}
	consulta.condicoes = append(consulta.condicoes,
		"p.status IN ('confirmado', 'parcialmente_entregue')",
		"i.quantidade > i.quantidade_entregue",
	)
	consulta.escopo(filtro.EscopoFiltro)
	if filtro.ClienteID != nil {
		consulta.igual("p.cliente_id", *filtro.ClienteID)
	}
	if filtro.ProdutoID != nil {
		consulta.igual("i.produto_id", *filtro.ProdutoID)
	}

	query := `
		SELECT p.id AS pedido_id, p.numero, p.empresa_id, p.cliente_id, p.representante_id, p.status, p.prazo_limite,
		       i.id AS item_id, i.produto_id, pr.nome AS produto_nome,
		       i.quantidade AS vendido, i.quantidade_entregue AS entregue,
		       COALESCE((
		           SELECT SUM(ei.quantidade)
		           FROM entrega_item ei
		           JOIN entrega e ON e.id = ei.entrega_id
		           WHERE ei.pedido_item_id = i.id AND e.status = 'pendente'
		       ), 0) AS programado,
		       COALESCE((
		           SELECT SUM(r.quantidade)
		           FROM produto_reserva r
		           WHERE r.origem_id = i.id AND r.status = 'ativa'
		       ), 0) AS reservado,
		       COALESCE((
		           SELECT SUM(s.quantidade - s.reservado)
		           FROM produto_saldo_local s
		           JOIN estoque_local l ON l.id = s.local_id
		           WHERE s.produto_id = i.produto_id AND l.empresa_id = p.empresa_id AND l.ativo
		       ), 0) AS disponivel_na_empresa
		FROM pedido_item i
		JOIN pedido p ON p.id = i.pedido_id
		JOIN produto pr ON pr.id = i.produto_id
		WHERE ` + unirCondicoes(consulta.condicoes) + `
		ORDER BY p.prazo_limite ASC NULLS LAST, p.criado_em ASC, p.id ASC, i.id ASC` +
		consulta.paginar(filtro.PaginacaoFiltro)

	var linhas []*pendenciaLinha
	if err := r.db.SelectContext(ctx, &linhas, query, consulta.argumentos...); err != nil {
		return nil, err
	}

	pendencias := make([]*domainpedido.Pendencia, 0, len(linhas))
	for _, linha := range linhas {
		pendencias = append(pendencias, &domainpedido.Pendencia{
			PedidoID:            linha.PedidoID,
			Numero:              linha.Numero,
			EmpresaID:           linha.EmpresaID,
			ClienteID:           linha.ClienteID,
			RepresentanteID:     linha.RepresentanteID,
			Status:              domainpedido.StatusDe(linha.Status),
			PrazoLimite:         dataDoBanco(linha.PrazoLimite),
			ItemID:              linha.ItemID,
			ProdutoID:           linha.ProdutoID,
			ProdutoNome:         linha.ProdutoNome,
			Vendido:             linha.Vendido,
			Entregue:            linha.Entregue,
			Programado:          linha.Programado,
			Reservado:           linha.Reservado,
			DisponivelNaEmpresa: linha.DisponivelNaEmpresa,
		})
	}

	return pendencias, nil
}

func (r *PedidoConsultaRepository) PorPrazo(ctx context.Context, filtro portsout.PrazoFiltro) ([]*domainpedido.Pedido, error) {
	consulta := &consultaDePedido{}
	consulta.condicoes = append(consulta.condicoes,
		"p.status IN ('confirmado', 'parcialmente_entregue')",
		"p.prazo_limite IS NOT NULL",
	)
	consulta.escopo(filtro.EscopoFiltro)
	if filtro.Janela.De != nil {
		consulta.comparar("p.prazo_limite", ">=", domainpedido.DataDe(*filtro.Janela.De))
	}
	consulta.comparar("p.prazo_limite", "<=", domainpedido.DataDe(filtro.Janela.Ate))

	return r.listar(ctx, consulta, `p.prazo_limite ASC, p.id ASC`, filtro.PaginacaoFiltro)
}

func (r *PedidoConsultaRepository) Cancelamentos(ctx context.Context, filtro portsout.CancelamentoFiltro) ([]*domainpedido.Pedido, error) {
	consulta := &consultaDePedido{}
	consulta.condicoes = append(consulta.condicoes, "p.status = 'cancelado'")
	consulta.escopo(filtro.EscopoFiltro)
	if filtro.ComDevolucao {
		consulta.condicoes = append(consulta.condicoes, "p.valor_a_devolver_centavos > 0")
	}
	if filtro.De != nil {
		consulta.comparar("p.atualizado_em", ">=", domainpedido.InicioDoDia(*filtro.De))
	}
	if filtro.Ate != nil {
		consulta.comparar("p.atualizado_em", "<", domainpedido.InicioDoDia(*filtro.Ate).AddDate(0, 0, 1))
	}

	return r.listar(ctx, consulta, `p.atualizado_em DESC, p.id DESC`, filtro.PaginacaoFiltro)
}

func (r *PedidoConsultaRepository) listar(
	ctx context.Context, consulta *consultaDePedido, ordem string, paginacao domain.PaginacaoFiltro,
) ([]*domainpedido.Pedido, error) {
	query := `SELECT ` + colunasPedidoComPrefixo + ` FROM pedido p WHERE ` + unirCondicoes(consulta.condicoes) +
		` ORDER BY ` + ordem + consulta.paginar(paginacao)

	var linhas []*pedidoLinha
	if err := r.db.SelectContext(ctx, &linhas, query, consulta.argumentos...); err != nil {
		return nil, err
	}

	pedidos := make([]*domainpedido.Pedido, 0, len(linhas))
	for _, linha := range linhas {
		pedidos = append(pedidos, paraDominioPedido(linha))
	}

	return pedidos, nil
}

const colunasPedidoComPrefixo = `
	p.id, p.empresa_id, p.numero, p.cliente_id, p.obra_id, p.representante_id, p.modelo, p.entrada_bp,
	p.prazo_boleto_dias, p.prazo_entrega_dias, p.prazo_limite, p.numero_pedido_cliente, p.observacoes,
	p.status, p.valor_total_centavos, p.entrada_paga_em, p.entrada_paga_centavos, p.cancelado_motivo,
	p.valor_a_devolver_centavos, p.usuario_id, p.versao, p.criado_em, p.atualizado_em
`
