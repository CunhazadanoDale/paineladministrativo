package postgres

import (
	"context"
	"fmt"
	"strconv"
	"time"

	domaindre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/dre"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/dre"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var _ portsout.RelatorioSemanalRepository = (*DRERelatorioSemanalRepository)(nil)

type DRERelatorioSemanalRepository struct {
	db *sqlx.DB
}

func NewDRERelatorioSemanalRepository(db *sqlx.DB) *DRERelatorioSemanalRepository {
	return &DRERelatorioSemanalRepository{db: db}
}

type linhaNotasEmitidas struct {
	EmpresaID  uuid.UUID `db:"empresa_id"`
	Natureza   string    `db:"natureza"`
	Quantidade int       `db:"quantidade"`
	Valor      int64     `db:"valor"`
}

func (r *DRERelatorioSemanalRepository) NotasEmitidas(ctx context.Context, filtro portsout.FiltroSemanal) ([]domaindre.NotasEmitidas, error) {
	argumentos := []any{filtro.Intervalo.InicioInstante(), filtro.Intervalo.FimInstante()}
	empresas, argumentos := filtroDREPorEmpresa("empresa_id", filtro.EmpresaIDs, argumentos)

	query := `
		SELECT empresa_id, natureza, count(*) AS quantidade, sum(valor_total_centavos) AS valor
		FROM nota_fiscal
		WHERE status = 'autorizada'
		  AND emitida_em >= $1 AND emitida_em < $2` + empresas + `
		GROUP BY 1, 2
		ORDER BY 1, 2
	`

	var linhas []linhaNotasEmitidas
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, fmt.Errorf("consultar notas emitidas na semana: %w", err)
	}

	notas := make([]domaindre.NotasEmitidas, 0, len(linhas))
	for _, linha := range linhas {
		notas = append(notas, domaindre.NotasEmitidas{
			EmpresaID: linha.EmpresaID,
			Natureza:  linha.Natureza,
			Total:     domaindre.TotalSemanal{Quantidade: linha.Quantidade, ValorCentavos: linha.Valor},
		})
	}

	return notas, nil
}

type linhaPagamentosFeitos struct {
	EmpresaID  uuid.UUID `db:"empresa_id"`
	LinhaDRE   *string   `db:"linha_dre"`
	Quantidade int       `db:"quantidade"`
	Valor      int64     `db:"valor"`
}

func (r *DRERelatorioSemanalRepository) PagamentosFeitos(ctx context.Context, filtro portsout.FiltroSemanal) ([]domaindre.PagamentosFeitos, error) {
	argumentos := []any{filtro.Intervalo.InicioInstante(), filtro.Intervalo.FimInstante()}
	empresas, argumentos := filtroDREPorEmpresa("s.empresa_id", filtro.EmpresaIDs, argumentos)

	query := `
		SELECT s.empresa_id, c.linha_dre, count(*) AS quantidade, sum(p.valor) AS valor
		FROM pagamento p
		JOIN solicitacao s ON s.id = p.solicitacao_id
		LEFT JOIN categoria_financeira c ON c.id = s.categoria_financeira_id
		WHERE NOT p.estornado
		  AND p.pago_em >= $1 AND p.pago_em < $2` + empresas + `
		GROUP BY 1, 2
		ORDER BY 1, 2 NULLS LAST
	`

	var linhas []linhaPagamentosFeitos
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, fmt.Errorf("consultar pagamentos da semana: %w", err)
	}

	pagamentos := make([]domaindre.PagamentosFeitos, 0, len(linhas))
	for _, linha := range linhas {
		pagamentos = append(pagamentos, domaindre.PagamentosFeitos{
			EmpresaID: linha.EmpresaID,
			LinhaDRE:  linha.LinhaDRE,
			Total:     domaindre.TotalSemanal{Quantidade: linha.Quantidade, ValorCentavos: linha.Valor},
		})
	}

	return pagamentos, nil
}

type linhaRecebimentosFeitos struct {
	EmpresaID  uuid.UUID `db:"empresa_id"`
	Tipo       string    `db:"tipo"`
	Quantidade int       `db:"quantidade"`
	Valor      int64     `db:"valor"`
	Juros      int64     `db:"juros"`
}

func (r *DRERelatorioSemanalRepository) RecebimentosFeitos(ctx context.Context, filtro portsout.FiltroSemanal) ([]domaindre.RecebimentosFeitos, error) {
	argumentos := []any{filtro.Intervalo.Inicio, filtro.Intervalo.Fim}
	empresas, argumentos := filtroDREPorEmpresa("t.empresa_id", filtro.EmpresaIDs, argumentos)

	query := `
		SELECT t.empresa_id, r.tipo, count(*) AS quantidade, sum(r.valor_centavos) AS valor, sum(r.juros_centavos) AS juros
		FROM titulo_recebimento r
		JOIN titulo_receber t ON t.id = r.titulo_id
		WHERE NOT r.estornado
		  AND r.recebido_em >= $1 AND r.recebido_em < $2` + empresas + `
		GROUP BY 1, 2
		ORDER BY 1, 2
	`

	var linhas []linhaRecebimentosFeitos
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, fmt.Errorf("consultar recebimentos da semana: %w", err)
	}

	recebimentos := make([]domaindre.RecebimentosFeitos, 0, len(linhas))
	for _, linha := range linhas {
		recebimentos = append(recebimentos, domaindre.RecebimentosFeitos{
			EmpresaID:     linha.EmpresaID,
			Tipo:          linha.Tipo,
			Total:         domaindre.TotalSemanal{Quantidade: linha.Quantidade, ValorCentavos: linha.Valor},
			JurosCentavos: linha.Juros,
		})
	}

	return recebimentos, nil
}

type linhaVencidosPorEmpresa struct {
	EmpresaID  uuid.UUID `db:"empresa_id"`
	Quantidade int       `db:"quantidade"`
	Valor      int64     `db:"valor"`
}

func paraVencidosPorEmpresa(linhas []linhaVencidosPorEmpresa) []domaindre.VencidosPorEmpresa {
	vencidos := make([]domaindre.VencidosPorEmpresa, 0, len(linhas))
	for _, linha := range linhas {
		vencidos = append(vencidos, domaindre.VencidosPorEmpresa{
			EmpresaID: linha.EmpresaID,
			Total:     domaindre.TotalSemanal{Quantidade: linha.Quantidade, ValorCentavos: linha.Valor},
		})
	}

	return vencidos
}

func limiteDaLista(argumentos []any, limite int) (string, []any) {
	argumentos = append(argumentos, limite)

	return " LIMIT $" + strconv.Itoa(len(argumentos)), argumentos
}

const condicaoTituloVencido = `
	t.status IN ('carteira', 'em_banco', 'antecipado')
	AND t.vencimento < $1
	AND t.valor_centavos > t.recebido_centavos`

type linhaTituloVencido struct {
	ID          uuid.UUID `db:"id"`
	EmpresaID   uuid.UUID `db:"empresa_id"`
	ClienteNome string    `db:"cliente_nome"`
	Numero      string    `db:"numero"`
	Parcela     int       `db:"parcela"`
	Status      string    `db:"status"`
	Vencimento  time.Time `db:"vencimento"`
	Saldo       int64     `db:"saldo"`
}

func (r *DRERelatorioSemanalRepository) TitulosVencidos(ctx context.Context, filtro portsout.FiltroSemanal) (domaindre.Vencidos[domaindre.TituloVencido], error) {
	empresas, argumentos := filtroDREPorEmpresa("t.empresa_id", filtro.EmpresaIDs, []any{filtro.Hoje})

	var porEmpresa []linhaVencidosPorEmpresa
	if err := r.db.SelectContext(ctx, &porEmpresa, `
		SELECT t.empresa_id, count(*) AS quantidade, sum(t.valor_centavos - t.recebido_centavos) AS valor
		FROM titulo_receber t
		WHERE `+condicaoTituloVencido+empresas+`
		GROUP BY 1
		ORDER BY 1
	`, argumentos...); err != nil {
		return domaindre.Vencidos[domaindre.TituloVencido]{}, fmt.Errorf("consultar títulos vencidos por empresa: %w", err)
	}

	limite, argumentos := limiteDaLista(argumentos, filtro.Limite)
	var linhas []linhaTituloVencido
	if err := r.db.SelectContext(ctx, &linhas, `
		SELECT t.id, t.empresa_id, c.nome AS cliente_nome, t.numero, t.parcela, t.status, t.vencimento,
		       t.valor_centavos - t.recebido_centavos AS saldo
		FROM titulo_receber t
		JOIN cliente c ON c.id = t.cliente_id
		WHERE `+condicaoTituloVencido+empresas+`
		ORDER BY t.vencimento, t.id`+limite, argumentos...); err != nil {
		return domaindre.Vencidos[domaindre.TituloVencido]{}, fmt.Errorf("consultar títulos vencidos: %w", err)
	}

	itens := make([]domaindre.TituloVencido, 0, len(linhas))
	for _, linha := range linhas {
		itens = append(itens, domaindre.TituloVencido{
			ID:            linha.ID,
			EmpresaID:     linha.EmpresaID,
			ClienteNome:   linha.ClienteNome,
			Numero:        linha.Numero,
			Parcela:       linha.Parcela,
			Status:        linha.Status,
			Vencimento:    linha.Vencimento,
			SaldoCentavos: linha.Saldo,
		})
	}

	return domaindre.NovosVencidos(paraVencidosPorEmpresa(porEmpresa), itens), nil
}

const condicaoContaVencida = `
	s.status IN ('pendente_aprovacao', 'aprovado')
	AND s.prazo_pagamento < $1`

type linhaContaVencida struct {
	ID             uuid.UUID `db:"id"`
	EmpresaID      uuid.UUID `db:"empresa_id"`
	FornecedorNome *string   `db:"fornecedor_nome"`
	Observacao     string    `db:"observacao"`
	Status         string    `db:"status"`
	Vencimento     time.Time `db:"vencimento"`
	Valor          int64     `db:"valor"`
}

func (r *DRERelatorioSemanalRepository) ContasVencidas(ctx context.Context, filtro portsout.FiltroSemanal) (domaindre.Vencidos[domaindre.ContaVencida], error) {
	empresas, argumentos := filtroDREPorEmpresa("s.empresa_id", filtro.EmpresaIDs, []any{filtro.Hoje})

	var porEmpresa []linhaVencidosPorEmpresa
	if err := r.db.SelectContext(ctx, &porEmpresa, `
		SELECT s.empresa_id, count(*) AS quantidade, sum(s.valor_estimado) AS valor
		FROM solicitacao s
		WHERE `+condicaoContaVencida+empresas+`
		GROUP BY 1
		ORDER BY 1
	`, argumentos...); err != nil {
		return domaindre.Vencidos[domaindre.ContaVencida]{}, fmt.Errorf("consultar contas vencidas por empresa: %w", err)
	}

	limite, argumentos := limiteDaLista(argumentos, filtro.Limite)
	var linhas []linhaContaVencida
	if err := r.db.SelectContext(ctx, &linhas, `
		SELECT s.id, s.empresa_id, f.nome AS fornecedor_nome, s.observacao, s.status, s.prazo_pagamento AS vencimento,
		       s.valor_estimado AS valor
		FROM solicitacao s
		LEFT JOIN fornecedor f ON f.id = s.fornecedor_id
		WHERE `+condicaoContaVencida+empresas+`
		ORDER BY s.prazo_pagamento, s.id`+limite, argumentos...); err != nil {
		return domaindre.Vencidos[domaindre.ContaVencida]{}, fmt.Errorf("consultar contas vencidas: %w", err)
	}

	itens := make([]domaindre.ContaVencida, 0, len(linhas))
	for _, linha := range linhas {
		itens = append(itens, domaindre.ContaVencida{
			ID:             linha.ID,
			EmpresaID:      linha.EmpresaID,
			FornecedorNome: linha.FornecedorNome,
			Observacao:     linha.Observacao,
			Status:         linha.Status,
			Vencimento:     linha.Vencimento,
			ValorCentavos:  linha.Valor,
		})
	}

	return domaindre.NovosVencidos(paraVencidosPorEmpresa(porEmpresa), itens), nil
}

const condicaoEstoqueBaixo = `
	ativo
	AND estoque_minimo IS NOT NULL
	AND estoque_minimo > 0
	AND quantidade_atual <= estoque_minimo`

type linhaProdutoEstoqueBaixo struct {
	ID              uuid.UUID `db:"id"`
	Nome            string    `db:"nome"`
	Codigo          *string   `db:"codigo"`
	QuantidadeAtual int       `db:"quantidade_atual"`
	EstoqueMinimo   int       `db:"estoque_minimo"`
}

func (r *DRERelatorioSemanalRepository) EstoqueBaixo(ctx context.Context, limite int) (domaindre.EstoqueBaixo, error) {
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT count(*) FROM produto WHERE `+condicaoEstoqueBaixo); err != nil {
		return domaindre.EstoqueBaixo{}, fmt.Errorf("contar produtos com estoque baixo: %w", err)
	}

	var linhas []linhaProdutoEstoqueBaixo
	if err := r.db.SelectContext(ctx, &linhas, `
		SELECT id, nome, codigo, quantidade_atual, estoque_minimo
		FROM produto
		WHERE `+condicaoEstoqueBaixo+`
		ORDER BY nome, id
		LIMIT $1
	`, limite); err != nil {
		return domaindre.EstoqueBaixo{}, fmt.Errorf("listar produtos com estoque baixo: %w", err)
	}

	itens := make([]domaindre.ProdutoEstoqueBaixo, 0, len(linhas))
	for _, linha := range linhas {
		itens = append(itens, domaindre.ProdutoEstoqueBaixo{
			ID:              linha.ID,
			Nome:            linha.Nome,
			Codigo:          linha.Codigo,
			QuantidadeAtual: linha.QuantidadeAtual,
			EstoqueMinimo:   linha.EstoqueMinimo,
		})
	}

	return domaindre.EstoqueBaixo{Total: total, Itens: itens}, nil
}
