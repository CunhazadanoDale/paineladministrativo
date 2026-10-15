package postgres

import (
	"context"
	"time"

	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/solicitacao"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ portsout.ResumoContasAPagarRepository = (*ContasAPagarResumoRepository)(nil)

type ContasAPagarResumoRepository struct {
	db *sqlx.DB
}

func NewContasAPagarResumoRepository(db *sqlx.DB) *ContasAPagarResumoRepository {
	return &ContasAPagarResumoRepository{db: db}
}

type linhaResumoSemanal struct {
	Semana             time.Time `db:"semana"`
	EmpresaID          uuid.UUID `db:"empresa_id"`
	PrevistoQuantidade int       `db:"previsto_quantidade"`
	PrevistoValor      int64     `db:"previsto_valor"`
	APagarQuantidade   int       `db:"a_pagar_quantidade"`
	APagarValor        int64     `db:"a_pagar_valor"`
	VencidoQuantidade  int       `db:"vencido_quantidade"`
	VencidoValor       int64     `db:"vencido_valor"`
	EstimadoValor      int64     `db:"estimado_valor"`
}

type linhaAtraso struct {
	EmpresaID  uuid.UUID `db:"empresa_id"`
	Quantidade int       `db:"quantidade"`
	Valor      int64     `db:"valor"`
}

func (r *ContasAPagarResumoRepository) ResumoSemanal(ctx context.Context, filtro portsout.FiltroResumo) ([]domainsolicitacao.LinhaResumoSemanal, error) {
	query := `
		SELECT date_trunc('week', prazo_pagamento)::date AS semana,
		       empresa_id,
		       count(*) FILTER (WHERE status = 'pendente_aprovacao' AND prazo_pagamento >= $3) AS previsto_quantidade,
		       coalesce(sum(valor_estimado) FILTER (WHERE status = 'pendente_aprovacao' AND prazo_pagamento >= $3), 0) AS previsto_valor,
		       count(*) FILTER (WHERE status = 'aprovado' AND prazo_pagamento >= $3) AS a_pagar_quantidade,
		       coalesce(sum(valor_estimado) FILTER (WHERE status = 'aprovado' AND prazo_pagamento >= $3), 0) AS a_pagar_valor,
		       count(*) FILTER (WHERE prazo_pagamento < $3) AS vencido_quantidade,
		       coalesce(sum(valor_estimado) FILTER (WHERE prazo_pagamento < $3), 0) AS vencido_valor,
		       coalesce(sum(valor_estimado) FILTER (WHERE estimado), 0) AS estimado_valor
		FROM solicitacao
		WHERE status IN ('pendente_aprovacao', 'aprovado')
		  AND prazo_pagamento BETWEEN $1 AND $2
	`
	argumentos := []any{filtro.De, filtro.Ate, filtro.Hoje}
	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		query += ` AND empresa_id = ANY($4::uuid[])`
	}
	query += ` GROUP BY 1, 2 ORDER BY 1, 2`

	var linhas []linhaResumoSemanal
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	semanas := make([]domainsolicitacao.LinhaResumoSemanal, 0, len(linhas))
	for _, linha := range linhas {
		semanas = append(semanas, domainsolicitacao.LinhaResumoSemanal{
			Semana:           linha.Semana,
			EmpresaID:        linha.EmpresaID,
			Previsto:         domainsolicitacao.TotalResumo{Quantidade: linha.PrevistoQuantidade, ValorCentavos: linha.PrevistoValor},
			APagar:           domainsolicitacao.TotalResumo{Quantidade: linha.APagarQuantidade, ValorCentavos: linha.APagarValor},
			Vencido:          domainsolicitacao.TotalResumo{Quantidade: linha.VencidoQuantidade, ValorCentavos: linha.VencidoValor},
			EstimadoCentavos: linha.EstimadoValor,
		})
	}

	return semanas, nil
}

func (r *ContasAPagarResumoRepository) Atrasadas(ctx context.Context, filtro portsout.FiltroResumo) ([]domainsolicitacao.AtrasoPorEmpresa, error) {
	query := `
		SELECT empresa_id, count(*) AS quantidade, coalesce(sum(valor_estimado), 0) AS valor
		FROM solicitacao
		WHERE status IN ('pendente_aprovacao', 'aprovado')
		  AND prazo_pagamento < $1
		  AND prazo_pagamento < $2
	`
	argumentos := []any{filtro.De, filtro.Hoje}
	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		query += ` AND empresa_id = ANY($3::uuid[])`
	}
	query += ` GROUP BY empresa_id ORDER BY empresa_id`

	var linhas []linhaAtraso
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	atrasos := make([]domainsolicitacao.AtrasoPorEmpresa, 0, len(linhas))
	for _, linha := range linhas {
		atrasos = append(atrasos, domainsolicitacao.AtrasoPorEmpresa{
			EmpresaID: linha.EmpresaID,
			Vencido:   domainsolicitacao.TotalResumo{Quantidade: linha.Quantidade, ValorCentavos: linha.Valor},
		})
	}

	return atrasos, nil
}
