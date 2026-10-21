package postgres

import (
	"context"
	"time"

	domaindivida "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/divida"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/divida"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ portsout.ConsultaRepository = (*DividaConsultaRepository)(nil)

type DividaConsultaRepository struct {
	db *sqlx.DB
}

func NewDividaConsultaRepository(db *sqlx.DB) *DividaConsultaRepository {
	return &DividaConsultaRepository{db: db}
}

type parcelaEmAbertoLinha struct {
	DividaID            uuid.UUID `db:"divida_id"`
	EmpresaID           uuid.UUID `db:"empresa_id"`
	CredorTipo          string    `db:"credor_tipo"`
	Vencimento          time.Time `db:"vencimento"`
	AmortizacaoCentavos int64     `db:"amortizacao_centavos"`
	JurosCentavos       int64     `db:"juros_centavos"`
}

func (r *DividaConsultaRepository) ParcelasEmAberto(ctx context.Context, empresaIDs []uuid.UUID) ([]domaindivida.ParcelaEmAberto, error) {
	query := `
		SELECT divida_id, empresa_id, credor_tipo, vencimento, amortizacao_centavos, juros_centavos
		FROM (
			SELECT d.id AS divida_id, d.empresa_id, d.credor_tipo, p.vencimento, p.id AS parcela_id,
			       CASE WHEN COALESCE(sa.status, '') = 'pago' THEN 0 ELSE p.amortizacao_centavos END AS amortizacao_centavos,
			       CASE WHEN COALESCE(sj.status, '') = 'pago' THEN 0 ELSE p.juros_centavos END AS juros_centavos
			FROM divida d
			JOIN divida_parcela p ON p.divida_id = d.id
			LEFT JOIN solicitacao sa ON sa.origem = 'divida_amortizacao' AND sa.origem_id = p.id
			LEFT JOIN solicitacao sj ON sj.origem = 'divida_juros' AND sj.origem_id = p.id
			WHERE d.status = 'ativa' AND p.status <> 'cancelada'
	`
	argumentos := []any{}
	if empresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(empresaIDs)))
		query += ` AND d.empresa_id = ANY($1::uuid[])`
	}
	query += `
		) em_aberto
		WHERE amortizacao_centavos + juros_centavos > 0
		ORDER BY vencimento, divida_id, parcela_id
	`

	var linhas []parcelaEmAbertoLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	parcelas := make([]domaindivida.ParcelaEmAberto, 0, len(linhas))
	for _, linha := range linhas {
		parcelas = append(parcelas, domaindivida.ParcelaEmAberto{
			DividaID:            linha.DividaID,
			EmpresaID:           linha.EmpresaID,
			CredorTipo:          domaindivida.TipoCredorDe(linha.CredorTipo),
			Vencimento:          domaindivida.DataDe(linha.Vencimento),
			AmortizacaoCentavos: linha.AmortizacaoCentavos,
			JurosCentavos:       linha.JurosCentavos,
		})
	}

	return parcelas, nil
}
