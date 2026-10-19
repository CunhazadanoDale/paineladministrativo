package postgres

import (
	"context"
	"time"

	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/caixa"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ portsout.SaldoRepository = (*CaixaSaldoRepository)(nil)

type CaixaSaldoRepository struct {
	db *sqlx.DB
}

func NewCaixaSaldoRepository(db *sqlx.DB) *CaixaSaldoRepository {
	return &CaixaSaldoRepository{db: db}
}

type caixaSaldoLinha struct {
	ContaBancariaID                  uuid.UUID  `db:"conta_bancaria_id"`
	EmpresaID                        uuid.UUID  `db:"empresa_id"`
	Apelido                          string     `db:"apelido"`
	Banco                            string     `db:"banco"`
	Ativa                            bool       `db:"ativa"`
	DataCorte                        *time.Time `db:"data_corte"`
	SaldoInicialCentavos             *int64     `db:"saldo_inicial_centavos"`
	MovimentoCentavos                int64      `db:"movimento_centavos"`
	UltimoLancamentoEm               *time.Time `db:"ultimo_lancamento_em"`
	SaldoBancoCentavos               *int64     `db:"saldo_banco_centavos"`
	SaldoBancoEm                     *time.Time `db:"saldo_banco_em"`
	MovimentoAteSaldoDoBancoCentavos int64      `db:"movimento_ate_saldo_do_banco_centavos"`
	LancamentosPendentes             int        `db:"lancamentos_pendentes"`
}

func (r *CaixaSaldoRepository) Saldos(ctx context.Context, empresaIDs []uuid.UUID) ([]*domaincaixa.SaldoConta, error) {
	query := `
		SELECT c.id AS conta_bancaria_id, c.empresa_id, c.apelido, c.banco, c.ativa,
		       a.data_corte, a.saldo_centavos AS saldo_inicial_centavos,
		       COALESCE(movimento.total, 0) AS movimento_centavos,
		       movimento.ultimo AS ultimo_lancamento_em,
		       banco.saldo_banco_centavos, banco.saldo_banco_em,
		       COALESCE(ate_banco.total, 0) AS movimento_ate_saldo_do_banco_centavos,
		       COALESCE(pendentes.quantidade, 0) AS lancamentos_pendentes
		FROM conta_bancaria c
		LEFT JOIN caixa_abertura a ON a.conta_bancaria_id = c.id
		LEFT JOIN LATERAL (
			SELECT sum(l.valor_centavos) AS total, max(l.data) AS ultimo
			FROM extrato_lancamento l
			WHERE l.conta_bancaria_id = c.id AND l.data > a.data_corte
		) movimento ON true
		LEFT JOIN LATERAL (
			SELECT i.saldo_banco_centavos, i.saldo_banco_em
			FROM extrato_importacao i
			WHERE i.conta_bancaria_id = c.id AND i.saldo_banco_em IS NOT NULL
			ORDER BY i.saldo_banco_em DESC, i.criado_em DESC, i.id DESC
			LIMIT 1
		) banco ON true
		LEFT JOIN LATERAL (
			SELECT sum(l.valor_centavos) AS total
			FROM extrato_lancamento l
			WHERE l.conta_bancaria_id = c.id AND l.data > a.data_corte AND l.data <= banco.saldo_banco_em
		) ate_banco ON true
		LEFT JOIN LATERAL (
			SELECT count(*) AS quantidade
			FROM extrato_lancamento l
			LEFT JOIN conciliacao co ON co.extrato_lancamento_id = l.id AND co.vigente
			WHERE l.conta_bancaria_id = c.id AND l.data > a.data_corte AND co.id IS NULL
		) pendentes ON true`
	argumentos := make([]any, 0, 1)

	if empresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(empresaIDs)))
		query += ` WHERE c.empresa_id = ANY($1::uuid[])`
	}
	query += ` ORDER BY c.empresa_id ASC, lower(c.apelido) ASC, c.id ASC`

	var linhas []*caixaSaldoLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	saldos := make([]*domaincaixa.SaldoConta, 0, len(linhas))
	for _, linha := range linhas {
		saldos = append(saldos, paraDominioCaixaSaldo(linha))
	}

	return saldos, nil
}

func paraDominioCaixaSaldo(linha *caixaSaldoLinha) *domaincaixa.SaldoConta {
	return &domaincaixa.SaldoConta{
		ContaBancariaID:                  linha.ContaBancariaID,
		EmpresaID:                        linha.EmpresaID,
		Apelido:                          linha.Apelido,
		Banco:                            domainfinanceiro.BancoDe(linha.Banco),
		Ativa:                            linha.Ativa,
		DataCorte:                        dataOpcionalDoCaixa(linha.DataCorte),
		SaldoInicialCentavos:             linha.SaldoInicialCentavos,
		MovimentoCentavos:                linha.MovimentoCentavos,
		UltimoLancamentoEm:               dataOpcionalDoCaixa(linha.UltimoLancamentoEm),
		SaldoBancoCentavos:               linha.SaldoBancoCentavos,
		SaldoBancoEm:                     dataOpcionalDoCaixa(linha.SaldoBancoEm),
		MovimentoAteSaldoDoBancoCentavos: linha.MovimentoAteSaldoDoBancoCentavos,
		LancamentosPendentes:             linha.LancamentosPendentes,
	}
}

func dataOpcionalDoCaixa(valor *time.Time) *time.Time {
	if valor == nil {
		return nil
	}

	data := domaincaixa.DataDe(*valor)

	return &data
}
