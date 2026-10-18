package postgres

import (
	"context"
	"time"

	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/receber"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ portsout.ResumoContasAReceberRepository = (*ContasAReceberResumoRepository)(nil)

type ContasAReceberResumoRepository struct {
	db *sqlx.DB
}

func NewContasAReceberResumoRepository(db *sqlx.DB) *ContasAReceberResumoRepository {
	return &ContasAReceberResumoRepository{db: db}
}

type linhaResumoAReceber struct {
	Semana               time.Time `db:"semana"`
	EmpresaID            uuid.UUID `db:"empresa_id"`
	AReceberQuantidade   int       `db:"a_receber_quantidade"`
	AReceberSaldo        int64     `db:"a_receber_saldo"`
	AntecipadoQuantidade int       `db:"antecipado_quantidade"`
	AntecipadoSaldo      int64     `db:"antecipado_saldo"`
	VencidoQuantidade    int       `db:"vencido_quantidade"`
	VencidoSaldo         int64     `db:"vencido_saldo"`
}

type linhaAtrasoAReceber struct {
	EmpresaID  uuid.UUID `db:"empresa_id"`
	Quantidade int       `db:"quantidade"`
	Saldo      int64     `db:"saldo"`
}

func (r *ContasAReceberResumoRepository) ResumoSemanal(ctx context.Context, filtro portsout.FiltroResumo) ([]domainreceber.LinhaResumoSemanal, error) {
	query := `
		SELECT date_trunc('week', vencimento)::date AS semana,
		       empresa_id,
		       count(*) FILTER (WHERE status IN ('carteira', 'em_banco') AND vencimento >= $3) AS a_receber_quantidade,
		       coalesce(sum(valor_centavos - recebido_centavos) FILTER (WHERE status IN ('carteira', 'em_banco') AND vencimento >= $3), 0) AS a_receber_saldo,
		       count(*) FILTER (WHERE status = 'antecipado' AND vencimento >= $3) AS antecipado_quantidade,
		       coalesce(sum(valor_centavos - recebido_centavos) FILTER (WHERE status = 'antecipado' AND vencimento >= $3), 0) AS antecipado_saldo,
		       count(*) FILTER (WHERE vencimento < $3) AS vencido_quantidade,
		       coalesce(sum(valor_centavos - recebido_centavos) FILTER (WHERE vencimento < $3), 0) AS vencido_saldo
		FROM titulo_receber
		WHERE status IN ('carteira', 'em_banco', 'antecipado')
		  AND recebido_centavos < valor_centavos
		  AND vencimento BETWEEN $1 AND $2
	`
	argumentos := []any{filtro.De, filtro.Ate, filtro.Hoje}
	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		query += ` AND empresa_id = ANY($4::uuid[])`
	}
	query += ` GROUP BY 1, 2 ORDER BY 1, 2`

	var linhas []linhaResumoAReceber
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	semanas := make([]domainreceber.LinhaResumoSemanal, 0, len(linhas))
	for _, linha := range linhas {
		semanas = append(semanas, domainreceber.LinhaResumoSemanal{
			Semana:     domainreceber.DataDe(linha.Semana),
			EmpresaID:  linha.EmpresaID,
			AReceber:   domainreceber.TotalResumo{Quantidade: linha.AReceberQuantidade, SaldoCentavos: linha.AReceberSaldo},
			Antecipado: domainreceber.TotalResumo{Quantidade: linha.AntecipadoQuantidade, SaldoCentavos: linha.AntecipadoSaldo},
			Vencido:    domainreceber.TotalResumo{Quantidade: linha.VencidoQuantidade, SaldoCentavos: linha.VencidoSaldo},
		})
	}

	return semanas, nil
}

func (r *ContasAReceberResumoRepository) Atrasados(ctx context.Context, filtro portsout.FiltroResumo) ([]domainreceber.AtrasoPorEmpresa, error) {
	query := `
		SELECT empresa_id, count(*) AS quantidade, coalesce(sum(valor_centavos - recebido_centavos), 0) AS saldo
		FROM titulo_receber
		WHERE status IN ('carteira', 'em_banco', 'antecipado')
		  AND recebido_centavos < valor_centavos
		  AND vencimento < $1
		  AND vencimento < $2
	`
	argumentos := []any{filtro.De, filtro.Hoje}
	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		query += ` AND empresa_id = ANY($3::uuid[])`
	}
	query += ` GROUP BY empresa_id ORDER BY empresa_id`

	var linhas []linhaAtrasoAReceber
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	atrasos := make([]domainreceber.AtrasoPorEmpresa, 0, len(linhas))
	for _, linha := range linhas {
		atrasos = append(atrasos, domainreceber.AtrasoPorEmpresa{
			EmpresaID: linha.EmpresaID,
			Vencido:   domainreceber.TotalResumo{Quantidade: linha.Quantidade, SaldoCentavos: linha.Saldo},
		})
	}

	return atrasos, nil
}
