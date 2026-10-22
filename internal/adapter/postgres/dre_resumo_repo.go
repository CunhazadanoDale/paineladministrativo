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
	"github.com/lib/pq"
)

var _ portsout.ResumoRepository = (*DREResumoRepository)(nil)

type DREResumoRepository struct {
	db *sqlx.DB
}

func NewDREResumoRepository(db *sqlx.DB) *DREResumoRepository {
	return &DREResumoRepository{db: db}
}

func mesNoFusoDoGrupo(coluna string) string {
	return fmt.Sprintf("date_trunc('month', (%s AT TIME ZONE 'UTC') - interval '3 hours')::date", coluna)
}

func filtroDREPorEmpresa(coluna string, empresaIDs []uuid.UUID, argumentos []any) (string, []any) {
	if empresaIDs == nil {
		return "", argumentos
	}

	argumentos = append(argumentos, pq.Array(textosDeIDs(empresaIDs)))

	return " AND " + coluna + " = ANY($" + strconv.Itoa(len(argumentos)) + "::uuid[])", argumentos
}

type linhaDREFixa struct {
	Mes        time.Time `db:"mes"`
	EmpresaID  uuid.UUID `db:"empresa_id"`
	Chave      string    `db:"chave"`
	Valor      int64     `db:"valor"`
	Quantidade int       `db:"quantidade"`
}

type linhaDRECategoria struct {
	Mes           time.Time  `db:"mes"`
	EmpresaID     uuid.UUID  `db:"empresa_id"`
	LinhaDRE      *string    `db:"linha_dre"`
	CategoriaID   *uuid.UUID `db:"categoria_id"`
	CategoriaNome *string    `db:"categoria_nome"`
	Valor         int64      `db:"valor"`
	Quantidade    int        `db:"quantidade"`
}

var linhasDosItensFixos = map[string]domaindre.Linha{
	domaindre.ItemNFVenda:             domaindre.LinhaReceitaBruta,
	domaindre.ItemNFDevolucao:         domaindre.LinhaDeducoes,
	domaindre.ItemCMV:                 domaindre.LinhaCMV,
	domaindre.ItemPerdasEstoque:       domaindre.LinhaPerdasEstoque,
	domaindre.ItemJurosRecebidos:      domaindre.LinhaResultadoFinanceiro,
	domaindre.ItemDescontosConcedidos: domaindre.LinhaResultadoFinanceiro,
	domaindre.ItemEncargosAntecipacao: domaindre.LinhaResultadoFinanceiro,
}

func (r *DREResumoRepository) Lancamentos(ctx context.Context, filtro portsout.FiltroResumo) ([]domaindre.Lancamento, error) {
	lancamentos := make([]domaindre.Lancamento, 0)

	for _, consulta := range []func(context.Context, portsout.FiltroResumo) ([]linhaDREFixa, error){
		r.notasFiscais, r.custosDeEstoque, r.recebimentos, r.encargosDeAntecipacao,
	} {
		linhas, err := consulta(ctx, filtro)
		if err != nil {
			return nil, err
		}
		for _, linha := range linhas {
			lancamentos = append(lancamentos, domaindre.Lancamento{
				Mes:           domaindre.MesDe(linha.Mes),
				EmpresaID:     linha.EmpresaID,
				Linha:         linhasDosItensFixos[linha.Chave],
				Item:          domaindre.ItemFixo(linha.Chave),
				ValorCentavos: linha.Valor,
				Quantidade:    linha.Quantidade,
			})
		}
	}

	for _, consulta := range []func(context.Context, portsout.FiltroResumo) ([]linhaDRECategoria, error){
		r.contasAPagar, r.titulosManuais, r.lancamentosDeCaixa,
	} {
		linhas, err := consulta(ctx, filtro)
		if err != nil {
			return nil, err
		}
		for _, linha := range linhas {
			lancamentos = append(lancamentos, lancamentoDaCategoria(linha))
		}
	}

	return lancamentos, nil
}

func lancamentoDaCategoria(linha linhaDRECategoria) domaindre.Lancamento {
	lancamento := domaindre.Lancamento{
		Mes:           domaindre.MesDe(linha.Mes),
		EmpresaID:     linha.EmpresaID,
		Linha:         domaindre.LinhaNaoClassificado,
		Item:          domaindre.ItemFixo(domaindre.ItemSemCategoria),
		ValorCentavos: linha.Valor,
		Quantidade:    linha.Quantidade,
	}
	if linha.LinhaDRE == nil || linha.CategoriaID == nil || linha.CategoriaNome == nil {
		return lancamento
	}

	lancamento.Linha, _ = domaindre.LinhaDaCategoria(*linha.LinhaDRE)
	lancamento.Item = domaindre.ItemDaCategoria(*linha.CategoriaID, *linha.CategoriaNome)

	return lancamento
}

func (r *DREResumoRepository) notasFiscais(ctx context.Context, filtro portsout.FiltroResumo) ([]linhaDREFixa, error) {
	argumentos := []any{filtro.Intervalo.InicioInstante(), filtro.Intervalo.FimInstante()}
	empresas, argumentos := filtroDREPorEmpresa("empresa_id", filtro.EmpresaIDs, argumentos)

	query := `
		SELECT ` + mesNoFusoDoGrupo("emitida_em") + ` AS mes,
		       empresa_id,
		       CASE WHEN natureza = 'venda' THEN '` + domaindre.ItemNFVenda + `' ELSE '` + domaindre.ItemNFDevolucao + `' END AS chave,
		       sum(CASE WHEN natureza = 'venda' THEN valor_total_centavos ELSE -valor_total_centavos END) AS valor,
		       count(*) FILTER (WHERE finalidade NOT IN ('complementar', 'ajuste')) AS quantidade
		FROM nota_fiscal
		WHERE status = 'autorizada'
		  AND NOT entre_empresas
		  AND (natureza = 'venda' OR (natureza = 'devolucao' AND tipo_operacao = 'entrada'))
		  AND emitida_em >= $1 AND emitida_em < $2` + empresas + `
		GROUP BY 1, 2, 3
		ORDER BY 1, 2, 3
	`

	return r.selecionarFixas(ctx, query, argumentos)
}

func (r *DREResumoRepository) custosDeEstoque(ctx context.Context, filtro portsout.FiltroResumo) ([]linhaDREFixa, error) {
	argumentos := []any{filtro.Intervalo.InicioInstante(), filtro.Intervalo.FimInstante()}
	empresas, argumentos := filtroDREPorEmpresa("l.empresa_id", filtro.EmpresaIDs, argumentos)

	query := `
		SELECT ` + mesNoFusoDoGrupo("m.criado_em") + ` AS mes,
		       l.empresa_id,
		       CASE WHEN m.tipo = 'saida' THEN '` + domaindre.ItemCMV + `' ELSE '` + domaindre.ItemPerdasEstoque + `' END AS chave,
		       -coalesce(sum(ml.quantidade::bigint * ml.custo_unitario_centavos), 0) AS valor,
		       count(DISTINCT m.id) AS quantidade
		FROM produto_movimento m
		JOIN produto_movimento_lote ml ON ml.movimento_id = m.id
		JOIN estoque_local l ON l.id = m.local_id
		WHERE m.tipo IN ('saida', 'perda', 'ajuste')
		  AND m.criado_em >= $1 AND m.criado_em < $2` + empresas + `
		GROUP BY 1, 2, 3
		ORDER BY 1, 2, 3
	`

	return r.selecionarFixas(ctx, query, argumentos)
}

func (r *DREResumoRepository) recebimentos(ctx context.Context, filtro portsout.FiltroResumo) ([]linhaDREFixa, error) {
	argumentos := []any{filtro.Intervalo.Inicio, filtro.Intervalo.Fim}
	empresas, argumentos := filtroDREPorEmpresa("t.empresa_id", filtro.EmpresaIDs, argumentos)

	query := `
		SELECT mes, empresa_id, chave, valor, quantidade
		FROM (
			SELECT date_trunc('month', r.recebido_em)::date AS mes,
			       t.empresa_id,
			       sum(r.juros_centavos) AS juros,
			       count(*) FILTER (WHERE r.juros_centavos > 0) AS com_juros,
			       sum(r.desconto_centavos) AS descontos,
			       count(*) FILTER (WHERE r.desconto_centavos > 0) AS com_desconto
			FROM titulo_recebimento r
			JOIN titulo_receber t ON t.id = r.titulo_id
			WHERE NOT r.estornado
			  AND r.tipo = 'conta'
			  AND (r.juros_centavos > 0 OR r.desconto_centavos > 0)
			  AND r.recebido_em >= $1 AND r.recebido_em < $2` + empresas + `
			GROUP BY 1, 2
		) agregado
		CROSS JOIN LATERAL (VALUES
			('` + domaindre.ItemJurosRecebidos + `', juros, com_juros),
			('` + domaindre.ItemDescontosConcedidos + `', -descontos, com_desconto)
		) AS item (chave, valor, quantidade)
		WHERE item.quantidade > 0
		ORDER BY 1, 2, 3
	`

	return r.selecionarFixas(ctx, query, argumentos)
}

func (r *DREResumoRepository) encargosDeAntecipacao(ctx context.Context, filtro portsout.FiltroResumo) ([]linhaDREFixa, error) {
	argumentos := []any{filtro.Intervalo.Inicio, filtro.Intervalo.Fim}
	empresas, argumentos := filtroDREPorEmpresa("empresa_id", filtro.EmpresaIDs, argumentos)

	query := `
		SELECT date_trunc('month', operado_em)::date AS mes,
		       empresa_id,
		       '` + domaindre.ItemEncargosAntecipacao + `' AS chave,
		       -sum(valor_bruto_centavos - valor_liquido_centavos) AS valor,
		       count(*) AS quantidade
		FROM antecipacao
		WHERE status = 'ativa'
		  AND valor_bruto_centavos > valor_liquido_centavos
		  AND operado_em >= $1 AND operado_em < $2` + empresas + `
		GROUP BY 1, 2
		ORDER BY 1, 2
	`

	return r.selecionarFixas(ctx, query, argumentos)
}

func (r *DREResumoRepository) contasAPagar(ctx context.Context, filtro portsout.FiltroResumo) ([]linhaDRECategoria, error) {
	argumentos := []any{filtro.Intervalo.Inicio, filtro.Intervalo.Fim, pq.Array(domaindre.LinhasDoPlanoDeContas())}
	empresas, argumentos := filtroDREPorEmpresa("s.empresa_id", filtro.EmpresaIDs, argumentos)

	query := `
		SELECT s.competencia AS mes,
		       s.empresa_id,
		       c.linha_dre,
		       raiz.id AS categoria_id,
		       raiz.nome AS categoria_nome,
		       -sum(CASE WHEN s.status = 'pago' THEN coalesce(p.valor, s.valor_estimado) ELSE s.valor_estimado END) AS valor,
		       count(*) AS quantidade
		FROM solicitacao s
		LEFT JOIN categoria_financeira c ON c.id = s.categoria_financeira_id
		LEFT JOIN categoria_financeira raiz ON raiz.id = coalesce(c.categoria_pai_id, c.id)
		LEFT JOIN pagamento p ON p.solicitacao_id = s.id AND NOT p.estornado
		WHERE s.status NOT IN ('cancelado', 'rejeitado')
		  AND s.competencia >= $1 AND s.competencia < $2
		  AND (c.id IS NULL OR c.linha_dre = ANY($3::text[]))
		  AND NOT EXISTS (SELECT 1 FROM importacao_parcela ip WHERE ip.solicitacao_id = s.id)
		  AND NOT EXISTS (SELECT 1 FROM importacao_custo ic WHERE ic.solicitacao_id = s.id AND ic.compoe_custo)` + empresas + `
		GROUP BY 1, 2, 3, 4, 5
		ORDER BY 1, 2, 3, 4
	`

	return r.selecionarPorCategoria(ctx, query, argumentos)
}

func (r *DREResumoRepository) titulosManuais(ctx context.Context, filtro portsout.FiltroResumo) ([]linhaDRECategoria, error) {
	argumentos := []any{filtro.Intervalo.Inicio, filtro.Intervalo.Fim, pq.Array(domaindre.LinhasDoPlanoDeContas())}
	empresas, argumentos := filtroDREPorEmpresa("t.empresa_id", filtro.EmpresaIDs, argumentos)

	query := `
		SELECT t.competencia AS mes,
		       t.empresa_id,
		       c.linha_dre,
		       raiz.id AS categoria_id,
		       raiz.nome AS categoria_nome,
		       sum(t.valor_centavos) AS valor,
		       count(*) AS quantidade
		FROM titulo_receber t
		JOIN categoria_financeira c ON c.id = t.categoria_financeira_id
		JOIN categoria_financeira raiz ON raiz.id = coalesce(c.categoria_pai_id, c.id)
		WHERE t.origem = 'manual'
		  AND t.status <> 'cancelado'
		  AND t.competencia >= $1 AND t.competencia < $2
		  AND c.linha_dre = ANY($3::text[])` + empresas + `
		GROUP BY 1, 2, 3, 4, 5
		ORDER BY 1, 2, 3, 4
	`

	return r.selecionarPorCategoria(ctx, query, argumentos)
}

func (r *DREResumoRepository) lancamentosDeCaixa(ctx context.Context, filtro portsout.FiltroResumo) ([]linhaDRECategoria, error) {
	argumentos := []any{filtro.Intervalo.Inicio, filtro.Intervalo.Fim, pq.Array(domaindre.LinhasDoPlanoDeContas())}
	empresas, argumentos := filtroDREPorEmpresa("lc.empresa_id", filtro.EmpresaIDs, argumentos)

	query := `
		SELECT lc.competencia AS mes,
		       lc.empresa_id,
		       c.linha_dre,
		       raiz.id AS categoria_id,
		       raiz.nome AS categoria_nome,
		       sum(lc.valor_centavos) AS valor,
		       count(*) AS quantidade
		FROM lancamento_caixa lc
		JOIN categoria_financeira c ON c.id = lc.categoria_financeira_id
		JOIN categoria_financeira raiz ON raiz.id = coalesce(c.categoria_pai_id, c.id)
		WHERE NOT lc.estornado
		  AND lc.competencia >= $1 AND lc.competencia < $2
		  AND c.linha_dre = ANY($3::text[])` + empresas + `
		GROUP BY 1, 2, 3, 4, 5
		ORDER BY 1, 2, 3, 4
	`

	return r.selecionarPorCategoria(ctx, query, argumentos)
}

func (r *DREResumoRepository) selecionarFixas(ctx context.Context, query string, argumentos []any) ([]linhaDREFixa, error) {
	var linhas []linhaDREFixa
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, fmt.Errorf("consultar a DRE: %w", err)
	}

	return linhas, nil
}

func (r *DREResumoRepository) selecionarPorCategoria(ctx context.Context, query string, argumentos []any) ([]linhaDRECategoria, error) {
	var linhas []linhaDRECategoria
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, fmt.Errorf("consultar a DRE por categoria: %w", err)
	}

	return linhas, nil
}

type linhaConsumoSemCusto struct {
	Mes        time.Time `db:"mes"`
	EmpresaID  uuid.UUID `db:"empresa_id"`
	Quantidade int64     `db:"quantidade"`
}

func (r *DREResumoRepository) ConsumosSemCusto(ctx context.Context, filtro portsout.FiltroResumo) ([]domaindre.ConsumoSemCusto, error) {
	argumentos := []any{filtro.Intervalo.InicioInstante(), filtro.Intervalo.FimInstante()}
	empresas, argumentos := filtroDREPorEmpresa("l.empresa_id", filtro.EmpresaIDs, argumentos)

	query := `
		SELECT ` + mesNoFusoDoGrupo("m.criado_em") + ` AS mes,
		       l.empresa_id,
		       sum(ml.quantidade) AS quantidade
		FROM produto_movimento m
		JOIN produto_movimento_lote ml ON ml.movimento_id = m.id
		JOIN estoque_local l ON l.id = m.local_id
		WHERE m.tipo IN ('saida', 'perda', 'ajuste')
		  AND ml.custo_unitario_centavos IS NULL
		  AND m.criado_em >= $1 AND m.criado_em < $2` + empresas + `
		GROUP BY 1, 2
		ORDER BY 1, 2
	`

	var linhas []linhaConsumoSemCusto
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, fmt.Errorf("consultar consumo sem custo: %w", err)
	}

	consumos := make([]domaindre.ConsumoSemCusto, 0, len(linhas))
	for _, linha := range linhas {
		consumos = append(consumos, domaindre.ConsumoSemCusto{
			Mes:        domaindre.MesDe(linha.Mes),
			EmpresaID:  linha.EmpresaID,
			Quantidade: linha.Quantidade,
		})
	}

	return consumos, nil
}

func (r *DREResumoRepository) Faturamento(ctx context.Context, filtro portsout.FiltroResumo) ([]domaindre.Faturamento, error) {
	argumentos := []any{filtro.Intervalo.InicioInstante(), filtro.Intervalo.FimInstante()}
	empresas, argumentos := filtroDREPorEmpresa("empresa_id", filtro.EmpresaIDs, argumentos)

	query := `
		SELECT ` + mesNoFusoDoGrupo("emitida_em") + ` AS mes,
		       empresa_id,
		       '` + domaindre.ItemNFVenda + `' AS chave,
		       sum(valor_total_centavos) AS valor,
		       count(*) FILTER (WHERE finalidade NOT IN ('complementar', 'ajuste')) AS quantidade
		FROM nota_fiscal
		WHERE status = 'autorizada'
		  AND NOT entre_empresas
		  AND natureza = 'venda'
		  AND emitida_em >= $1 AND emitida_em < $2` + empresas + `
		GROUP BY 1, 2
		ORDER BY 1, 2
	`

	linhas, err := r.selecionarFixas(ctx, query, argumentos)
	if err != nil {
		return nil, err
	}

	faturamentos := make([]domaindre.Faturamento, 0, len(linhas))
	for _, linha := range linhas {
		faturamentos = append(faturamentos, domaindre.Faturamento{
			Mes:           domaindre.MesDe(linha.Mes),
			EmpresaID:     linha.EmpresaID,
			Quantidade:    linha.Quantidade,
			ValorCentavos: linha.Valor,
		})
	}

	return faturamentos, nil
}
