package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/caixa"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
)

var _ portsout.FechamentoRepository = (*CaixaFechamentoRepository)(nil)

type CaixaFechamentoRepository struct {
	db *sqlx.DB
}

func NewCaixaFechamentoRepository(db *sqlx.DB) *CaixaFechamentoRepository {
	return &CaixaFechamentoRepository{db: db}
}

type caixaFechamentoLinha struct {
	ID                   uuid.UUID  `db:"id"`
	ContaBancariaID      uuid.UUID  `db:"conta_bancaria_id"`
	Competencia          time.Time  `db:"competencia"`
	SaldoInicialCentavos int64      `db:"saldo_inicial_centavos"`
	EntradasCentavos     int64      `db:"entradas_centavos"`
	SaidasCentavos       int64      `db:"saidas_centavos"`
	SaldoFinalCentavos   int64      `db:"saldo_final_centavos"`
	SaldoBancoCentavos   int64      `db:"saldo_banco_centavos"`
	Vigente              bool       `db:"vigente"`
	ReabertoEm           *time.Time `db:"reaberto_em"`
	ReabertoPor          *uuid.UUID `db:"reaberto_por"`
	MotivoReabertura     *string    `db:"motivo_reabertura"`
	UsuarioID            uuid.UUID  `db:"usuario_id"`
	CriadoEm             time.Time  `db:"criado_em"`
}

type movimentoDoMesLinha struct {
	Entradas    int64 `db:"entradas"`
	Saidas      int64 `db:"saidas"`
	Pendentes   int   `db:"pendentes"`
	Invalidadas int   `db:"invalidadas"`
}

const (
	colunasCaixaFechamento = `id, conta_bancaria_id, competencia, saldo_inicial_centavos, entradas_centavos, saidas_centavos,
		saldo_final_centavos, saldo_banco_centavos, vigente, reaberto_em, reaberto_por, motivo_reabertura, usuario_id, criado_em`
	colunasCaixaFechamentoComPrefixo = `f.id, f.conta_bancaria_id, f.competencia, f.saldo_inicial_centavos, f.entradas_centavos,
		f.saidas_centavos, f.saldo_final_centavos, f.saldo_banco_centavos, f.vigente, f.reaberto_em, f.reaberto_por,
		f.motivo_reabertura, f.usuario_id, f.criado_em`
)

func (r *CaixaFechamentoRepository) Ultimo(ctx context.Context, contaBancariaID uuid.UUID) (*domaincaixa.Fechamento, error) {
	return ultimoFechamento(ctx, r.db, contaBancariaID)
}

func (r *CaixaFechamentoRepository) MovimentoDoMes(
	ctx context.Context,
	contaBancariaID uuid.UUID,
	competencia, dataCorte time.Time,
) (*domaincaixa.MovimentoDoMes, error) {
	return movimentoDoMes(ctx, r.db, contaBancariaID, competencia, dataCorte)
}

func (r *CaixaFechamentoRepository) Fechar(ctx context.Context, gravacao *portsout.GravacaoFechamento) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	fechamento := gravacao.Fechamento
	if _, err := tx.ExecContext(ctx, `SELECT id FROM conta_bancaria WHERE id = $1 FOR NO KEY UPDATE`, fechamento.ContaBancariaID); err != nil {
		return tratarErroDeGravacao(err)
	}

	var abertura struct {
		Versao    int       `db:"versao"`
		DataCorte time.Time `db:"data_corte"`
	}
	err = tx.GetContext(ctx, &abertura, `SELECT versao, data_corte FROM caixa_abertura WHERE conta_bancaria_id = $1 FOR SHARE`,
		fechamento.ContaBancariaID)
	if errors.Is(err, sql.ErrNoRows) {
		return domaincaixa.ErroContaSemAbertura()
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if abertura.Versao != gravacao.VersaoDaAbertura {
		return domaincaixa.ErroAberturaAlterada()
	}

	ultimo, err := ultimoFechamento(ctx, tx, fechamento.ContaBancariaID)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if (ultimo == nil) != (gravacao.UltimoID == nil) || (ultimo != nil && ultimo.ID != *gravacao.UltimoID) {
		return domaincaixa.ErroExtratoDoMesMudou()
	}

	movimento, err := movimentoDoMes(ctx, tx, fechamento.ContaBancariaID, fechamento.Competencia, domaincaixa.DataDe(abertura.DataCorte))
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if movimento.LancamentosPendentes > 0 || movimento.ConciliacoesInvalidadas > 0 ||
		movimento.EntradasCentavos != fechamento.EntradasCentavos || movimento.SaidasCentavos != fechamento.SaidasCentavos {
		return domaincaixa.ErroExtratoDoMesMudou()
	}

	if _, err := tx.NamedExecContext(ctx, `
		INSERT INTO caixa_fechamento (`+colunasCaixaFechamento+`)
		VALUES (
			:id, :conta_bancaria_id, :competencia, :saldo_inicial_centavos, :entradas_centavos, :saidas_centavos,
			:saldo_final_centavos, :saldo_banco_centavos, :vigente, :reaberto_em, :reaberto_por, :motivo_reabertura,
			:usuario_id, :criado_em
		)
	`, paraLinhaCaixaFechamento(fechamento)); err != nil {
		var erroPostgres *pq.Error
		if errors.As(err, &erroPostgres) && erroPostgres.Code == pqerror.UniqueViolation {
			return domaincaixa.ErroExtratoDoMesMudou()
		}
		return tratarErroDeGravacao(err)
	}

	if err := tx.Commit(); err != nil {
		return tratarErroDeGravacao(err)
	}

	return nil
}

func (r *CaixaFechamentoRepository) Obter(ctx context.Context, id uuid.UUID) (*domaincaixa.Fechamento, error) {
	var linha caixaFechamentoLinha
	err := r.db.GetContext(ctx, &linha, `SELECT `+colunasCaixaFechamento+` FROM caixa_fechamento WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return paraDominioCaixaFechamento(&linha), nil
}

func (r *CaixaFechamentoRepository) Listar(ctx context.Context, filtro portsout.FechamentoFiltro) ([]*domaincaixa.Fechamento, error) {
	argumentos := make([]any, 0, 5)
	condicoes := make([]string, 0, 3)

	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		condicoes = append(condicoes, "c.empresa_id = ANY($"+strconv.Itoa(len(argumentos))+"::uuid[])")
	}
	if filtro.ContaBancariaID != nil {
		argumentos = append(argumentos, *filtro.ContaBancariaID)
		condicoes = append(condicoes, condicao("f.conta_bancaria_id", len(argumentos)))
	}
	if filtro.Competencia != nil {
		argumentos = append(argumentos, filtro.Competencia.Format(formatoDataDoBanco))
		condicoes = append(condicoes, "f.competencia = $"+strconv.Itoa(len(argumentos))+"::date")
	}

	query := `SELECT ` + colunasCaixaFechamentoComPrefixo + `
		FROM caixa_fechamento f
		JOIN conta_bancaria c ON c.id = f.conta_bancaria_id`
	if len(condicoes) > 0 {
		query += " WHERE " + unirCondicoes(condicoes)
	}
	argumentos = append(argumentos, filtro.Limite, filtro.Deslocamento)
	query += ` ORDER BY f.competencia DESC, f.criado_em DESC, f.id DESC` + limiteOffset(argumentos)

	var linhas []*caixaFechamentoLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	fechamentos := make([]*domaincaixa.Fechamento, 0, len(linhas))
	for _, linha := range linhas {
		fechamentos = append(fechamentos, paraDominioCaixaFechamento(linha))
	}

	return fechamentos, nil
}

func (r *CaixaFechamentoRepository) Reabrir(ctx context.Context, fechamento *domaincaixa.Fechamento) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `SELECT id FROM conta_bancaria WHERE id = $1 FOR NO KEY UPDATE`, fechamento.ContaBancariaID); err != nil {
		return tratarErroDeGravacao(err)
	}

	var posterior bool
	if err := tx.GetContext(ctx, &posterior, `
		SELECT EXISTS (
			SELECT 1 FROM caixa_fechamento WHERE conta_bancaria_id = $1 AND vigente AND competencia > $2::date
		)
	`, fechamento.ContaBancariaID, fechamento.Competencia.Format(formatoDataDoBanco)); err != nil {
		return tratarErroDeGravacao(err)
	}
	if posterior {
		return domaincaixa.ErroFechamentoPosterior()
	}

	resultado, err := tx.ExecContext(ctx, `
		UPDATE caixa_fechamento
		SET vigente = false, reaberto_em = $1, reaberto_por = $2, motivo_reabertura = $3
		WHERE id = $4 AND vigente
	`, fechamento.ReabertoEm, fechamento.ReabertoPor, fechamento.MotivoReabertura, fechamento.ID)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	linhas, err := resultado.RowsAffected()
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if linhas == 0 {
		return domaincaixa.ErroFechamentoJaReaberto()
	}

	if err := tx.Commit(); err != nil {
		return tratarErroDeGravacao(err)
	}

	return nil
}

func ultimoFechamento(ctx context.Context, consulta sqlx.QueryerContext, contaBancariaID uuid.UUID) (*domaincaixa.Fechamento, error) {
	var linha caixaFechamentoLinha
	err := sqlx.GetContext(ctx, consulta, &linha, `
		SELECT `+colunasCaixaFechamento+` FROM caixa_fechamento
		WHERE conta_bancaria_id = $1 AND vigente
		ORDER BY competencia DESC
		LIMIT 1
	`, contaBancariaID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return paraDominioCaixaFechamento(&linha), nil
}

func movimentoDoMes(
	ctx context.Context,
	consulta sqlx.QueryerContext,
	contaBancariaID uuid.UUID,
	competencia, dataCorte time.Time,
) (*domaincaixa.MovimentoDoMes, error) {
	inicio := domaincaixa.CompetenciaDe(competencia)
	fim := domaincaixa.FimDaCompetencia(competencia)

	var linha movimentoDoMesLinha
	if err := sqlx.GetContext(ctx, consulta, &linha, `
		SELECT COALESCE(SUM(l.valor_centavos) FILTER (WHERE l.valor_centavos > 0), 0) AS entradas,
		       COALESCE(-SUM(l.valor_centavos) FILTER (WHERE l.valor_centavos < 0), 0) AS saidas,
		       COUNT(*) FILTER (WHERE co.id IS NULL) AS pendentes,
		       COUNT(*) FILTER (
		           WHERE co.id IS NOT NULL
		             AND EXISTS (SELECT 1 FROM conciliacao_item ci WHERE ci.conciliacao_id = co.id AND `+alvoEstornado+`)
		       ) AS invalidadas
		FROM extrato_lancamento l
		LEFT JOIN conciliacao co ON co.extrato_lancamento_id = l.id AND co.vigente
		WHERE l.conta_bancaria_id = $1
		  AND l.data >= $2::date
		  AND l.data <= $3::date
		  AND l.data > $4::date
	`, contaBancariaID, inicio.Format(formatoDataDoBanco), fim.Format(formatoDataDoBanco), dataCorte.Format(formatoDataDoBanco)); err != nil {
		return nil, err
	}

	movimento := &domaincaixa.MovimentoDoMes{
		EntradasCentavos:        linha.Entradas,
		SaidasCentavos:          linha.Saidas,
		LancamentosPendentes:    linha.Pendentes,
		ConciliacoesInvalidadas: linha.Invalidadas,
	}

	var saldoBanco int64
	err := sqlx.GetContext(ctx, consulta, &saldoBanco, `
		SELECT saldo_banco_centavos FROM extrato_importacao
		WHERE conta_bancaria_id = $1 AND saldo_banco_em = $2::date
		ORDER BY criado_em DESC, id DESC
		LIMIT 1
	`, contaBancariaID, fim.Format(formatoDataDoBanco))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		movimento.SaldoBancoCentavos = &saldoBanco
	}

	return movimento, nil
}

func competenciasFechadas(ctx context.Context, consulta sqlx.QueryerContext, contaBancariaID uuid.UUID) ([]time.Time, error) {
	var competencias []time.Time
	if err := sqlx.SelectContext(ctx, consulta, &competencias, `
		SELECT competencia FROM caixa_fechamento WHERE conta_bancaria_id = $1 AND vigente ORDER BY competencia
	`, contaBancariaID); err != nil {
		return nil, err
	}

	for i, competencia := range competencias {
		competencias[i] = domaincaixa.DataDe(competencia)
	}

	return competencias, nil
}

func exigirMesAberto(ctx context.Context, tx *sqlx.Tx, contaBancariaID uuid.UUID, data time.Time) error {
	if _, err := tx.ExecContext(ctx, `SELECT id FROM conta_bancaria WHERE id = $1 FOR SHARE`, contaBancariaID); err != nil {
		return tratarErroDeGravacao(err)
	}

	var fechado bool
	if err := tx.GetContext(ctx, &fechado, `
		SELECT EXISTS (SELECT 1 FROM caixa_fechamento WHERE conta_bancaria_id = $1 AND vigente AND competencia = $2::date)
	`, contaBancariaID, domaincaixa.CompetenciaDe(data).Format(formatoDataDoBanco)); err != nil {
		return tratarErroDeGravacao(err)
	}
	if fechado {
		return domaincaixa.ErroMesFechado()
	}

	return nil
}

func paraLinhaCaixaFechamento(fechamento *domaincaixa.Fechamento) *caixaFechamentoLinha {
	return &caixaFechamentoLinha{
		ID:                   fechamento.ID,
		ContaBancariaID:      fechamento.ContaBancariaID,
		Competencia:          fechamento.Competencia,
		SaldoInicialCentavos: fechamento.SaldoInicialCentavos,
		EntradasCentavos:     fechamento.EntradasCentavos,
		SaidasCentavos:       fechamento.SaidasCentavos,
		SaldoFinalCentavos:   fechamento.SaldoFinalCentavos,
		SaldoBancoCentavos:   fechamento.SaldoBancoCentavos,
		Vigente:              fechamento.Vigente,
		ReabertoEm:           fechamento.ReabertoEm,
		ReabertoPor:          fechamento.ReabertoPor,
		MotivoReabertura:     fechamento.MotivoReabertura,
		UsuarioID:            fechamento.UsuarioID,
		CriadoEm:             fechamento.CriadoEm,
	}
}

func paraDominioCaixaFechamento(linha *caixaFechamentoLinha) *domaincaixa.Fechamento {
	return &domaincaixa.Fechamento{
		ID:                   linha.ID,
		ContaBancariaID:      linha.ContaBancariaID,
		Competencia:          domaincaixa.DataDe(linha.Competencia),
		SaldoInicialCentavos: linha.SaldoInicialCentavos,
		EntradasCentavos:     linha.EntradasCentavos,
		SaidasCentavos:       linha.SaidasCentavos,
		SaldoFinalCentavos:   linha.SaldoFinalCentavos,
		SaldoBancoCentavos:   linha.SaldoBancoCentavos,
		Vigente:              linha.Vigente,
		ReabertoEm:           linha.ReabertoEm,
		ReabertoPor:          linha.ReabertoPor,
		MotivoReabertura:     linha.MotivoReabertura,
		UsuarioID:            linha.UsuarioID,
		CriadoEm:             linha.CriadoEm,
	}
}
