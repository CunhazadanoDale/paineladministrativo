package postgres

import (
	"context"
	"strconv"
	"time"

	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/caixa"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ portsout.ProjecaoRepository = (*CaixaProjecaoRepository)(nil)

type CaixaProjecaoRepository struct {
	db *sqlx.DB
}

func NewCaixaProjecaoRepository(db *sqlx.DB) *CaixaProjecaoRepository {
	return &CaixaProjecaoRepository{db: db}
}

type previstoLinha struct {
	Data          time.Time `db:"data"`
	ValorCentavos int64     `db:"valor_centavos"`
	Estimado      bool      `db:"estimado"`
}

type movimentoDiarioLinha struct {
	ContaBancariaID uuid.UUID `db:"conta_bancaria_id"`
	Data            time.Time `db:"data"`
	Entradas        int64     `db:"entradas"`
	Saidas          int64     `db:"saidas"`
	Saldo           int64     `db:"saldo"`
}

func (r *CaixaProjecaoRepository) SaldoDePartida(ctx context.Context, empresaIDs []uuid.UUID) (*domaincaixa.SaldoDePartida, error) {
	condicaoDaEmpresa, argumentos := filtroDeEmpresa("c.empresa_id", empresaIDs, nil)

	var resumo struct {
		Consideradas int   `db:"consideradas"`
		SemAbertura  int   `db:"sem_abertura"`
		Extrato      int64 `db:"extrato"`
	}
	if err := r.db.GetContext(ctx, &resumo, `
		SELECT COUNT(*) FILTER (WHERE a.conta_bancaria_id IS NOT NULL) AS consideradas,
		       COUNT(*) FILTER (WHERE a.conta_bancaria_id IS NULL) AS sem_abertura,
		       COALESCE(SUM(a.saldo_centavos + COALESCE((
		           SELECT SUM(l.valor_centavos) FROM extrato_lancamento l
		           WHERE l.conta_bancaria_id = c.id AND l.data > a.data_corte
		       ), 0)) FILTER (WHERE a.conta_bancaria_id IS NOT NULL), 0) AS extrato
		FROM conta_bancaria c
		LEFT JOIN caixa_abertura a ON a.conta_bancaria_id = c.id
		WHERE c.ativa`+condicaoDaEmpresa, argumentos...); err != nil {
		return nil, err
	}

	var contas []string
	if err := r.db.SelectContext(ctx, &contas, `
		SELECT c.id::text FROM conta_bancaria c
		JOIN caixa_abertura a ON a.conta_bancaria_id = c.id
		WHERE c.ativa`+condicaoDaEmpresa, argumentos...); err != nil {
		return nil, err
	}

	var emTransito int64
	if len(contas) > 0 {
		if err := r.db.GetContext(ctx, &emTransito, `
			SELECT COALESCE(SUM(x.valor_centavos), 0)
			FROM (`+consultaMovimentosDasContas+`) x
			LEFT JOIN LATERAL (
				SELECT max(i.periodo_fim) AS fim FROM extrato_importacao i WHERE i.conta_bancaria_id = x.conta_bancaria_id
			) cobertura ON true
			WHERE NOT x.estornado AND x.conciliacao_id IS NULL AND (cobertura.fim IS NULL OR x.data > cobertura.fim)
		`, pq.Array(contas)); err != nil {
			return nil, err
		}
	}

	return &domaincaixa.SaldoDePartida{
		ExtratoCentavos:    resumo.Extrato,
		EmTransitoCentavos: emTransito,
		ContasConsideradas: resumo.Consideradas,
		ContasSemAbertura:  resumo.SemAbertura,
	}, nil
}

func (r *CaixaProjecaoRepository) TitulosAReceber(ctx context.Context, empresaIDs []uuid.UUID, ate time.Time) ([]domaincaixa.Previsto, error) {
	condicaoDaEmpresa, argumentos := filtroDeEmpresa("empresa_id", empresaIDs, []any{ate.Format(formatoDataDoBanco)})

	return r.selecionarPrevistos(ctx, `
		SELECT vencimento AS data, valor_centavos - recebido_centavos AS valor_centavos, false AS estimado
		FROM titulo_receber
		WHERE status IN ('carteira', 'em_banco')
		  AND valor_centavos > recebido_centavos
		  AND vencimento <= $1::date`+condicaoDaEmpresa+`
		ORDER BY vencimento ASC, id ASC
	`, argumentos...)
}

func (r *CaixaProjecaoRepository) ContasAPagar(ctx context.Context, empresaIDs []uuid.UUID, ate time.Time) ([]domaincaixa.Previsto, error) {
	condicaoDaEmpresa, argumentos := filtroDeEmpresa("empresa_id", empresaIDs, []any{ate.Format(formatoDataDoBanco)})

	return r.selecionarPrevistos(ctx, `
		SELECT prazo_pagamento AS data, valor_estimado AS valor_centavos, estimado
		FROM solicitacao
		WHERE status IN ('pendente_aprovacao', 'aprovado')
		  AND prazo_pagamento <= $1::date`+condicaoDaEmpresa+`
		ORDER BY prazo_pagamento ASC, id ASC
	`, argumentos...)
}

func (r *CaixaProjecaoRepository) RecorrenciasGeradas(
	ctx context.Context,
	candidatas []portsout.RecorrenciaNaCompetencia,
) (map[portsout.RecorrenciaNaCompetencia]bool, error) {
	geradas := make(map[portsout.RecorrenciaNaCompetencia]bool, len(candidatas))
	if len(candidatas) == 0 {
		return geradas, nil
	}

	ids := make([]string, 0, len(candidatas))
	competencias := make([]string, 0, len(candidatas))
	for _, candidata := range candidatas {
		ids = append(ids, candidata.RecorrenciaID.String())
		competencias = append(competencias, candidata.Competencia.Format(formatoDataDoBanco))
	}

	var linhas []struct {
		RecorrenciaID uuid.UUID `db:"origem_id"`
		Competencia   time.Time `db:"competencia"`
	}
	if err := r.db.SelectContext(ctx, &linhas, `
		SELECT origem_id, competencia FROM solicitacao
		WHERE origem = 'recorrencia'
		  AND (origem_id, competencia) IN (SELECT * FROM unnest($1::uuid[], $2::date[]))
	`, pq.Array(ids), pq.Array(competencias)); err != nil {
		return nil, err
	}

	for _, linha := range linhas {
		geradas[portsout.RecorrenciaNaCompetencia{
			RecorrenciaID: linha.RecorrenciaID,
			Competencia:   domaincaixa.DataDe(linha.Competencia),
		}] = true
	}

	return geradas, nil
}

func (r *CaixaProjecaoRepository) MovimentoDiario(ctx context.Context, filtro portsout.MovimentoDiarioFiltro) ([]*domaincaixa.MovimentoDiario, error) {
	argumentos := []any{filtro.De.Format(formatoDataDoBanco), filtro.Ate.Format(formatoDataDoBanco)}
	condicoes := ""
	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		condicoes += " AND c.empresa_id = ANY($" + strconv.Itoa(len(argumentos)) + "::uuid[])"
	}
	if filtro.ContaBancariaID != nil {
		argumentos = append(argumentos, *filtro.ContaBancariaID)
		condicoes += " AND l.conta_bancaria_id = $" + strconv.Itoa(len(argumentos))
	}

	var linhas []*movimentoDiarioLinha
	if err := r.db.SelectContext(ctx, &linhas, `
		WITH dia AS (
			SELECT l.conta_bancaria_id, l.data,
			       COALESCE(SUM(l.valor_centavos) FILTER (WHERE l.valor_centavos > 0), 0) AS entradas,
			       COALESCE(-SUM(l.valor_centavos) FILTER (WHERE l.valor_centavos < 0), 0) AS saidas,
			       SUM(l.valor_centavos) AS movimento
			FROM extrato_lancamento l
			JOIN conta_bancaria c ON c.id = l.conta_bancaria_id
			JOIN caixa_abertura a ON a.conta_bancaria_id = l.conta_bancaria_id AND l.data > a.data_corte
			WHERE l.data <= $2::date`+condicoes+`
			GROUP BY l.conta_bancaria_id, l.data
		), acumulado AS (
			SELECT dia.conta_bancaria_id, dia.data, dia.entradas, dia.saidas,
			       a.saldo_centavos + SUM(dia.movimento) OVER (PARTITION BY dia.conta_bancaria_id ORDER BY dia.data) AS saldo
			FROM dia
			JOIN caixa_abertura a ON a.conta_bancaria_id = dia.conta_bancaria_id
		)
		SELECT conta_bancaria_id, data, entradas, saidas, saldo
		FROM acumulado
		WHERE data >= $1::date
		ORDER BY data ASC, conta_bancaria_id ASC
	`, argumentos...); err != nil {
		return nil, err
	}

	movimentos := make([]*domaincaixa.MovimentoDiario, 0, len(linhas))
	for _, linha := range linhas {
		movimentos = append(movimentos, &domaincaixa.MovimentoDiario{
			Data:             domaincaixa.DataDe(linha.Data),
			ContaBancariaID:  linha.ContaBancariaID,
			EntradasCentavos: linha.Entradas,
			SaidasCentavos:   linha.Saidas,
			SaldoCentavos:    linha.Saldo,
		})
	}

	return movimentos, nil
}

func (r *CaixaProjecaoRepository) selecionarPrevistos(ctx context.Context, query string, argumentos ...any) ([]domaincaixa.Previsto, error) {
	var linhas []previstoLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	previstos := make([]domaincaixa.Previsto, 0, len(linhas))
	for _, linha := range linhas {
		previstos = append(previstos, domaincaixa.Previsto{
			Data:          domaincaixa.DataDe(linha.Data),
			ValorCentavos: linha.ValorCentavos,
			Estimado:      linha.Estimado,
		})
	}

	return previstos, nil
}

func filtroDeEmpresa(coluna string, empresaIDs []uuid.UUID, argumentos []any) (string, []any) {
	if empresaIDs == nil {
		return "", argumentos
	}

	argumentos = append(argumentos, pq.Array(textosDeIDs(empresaIDs)))

	return " AND " + coluna + " = ANY($" + strconv.Itoa(len(argumentos)) + "::uuid[])", argumentos
}
