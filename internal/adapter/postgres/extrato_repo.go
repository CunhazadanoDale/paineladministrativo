package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/caixa"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
)

var _ portsout.ExtratoRepository = (*ExtratoRepository)(nil)

type ExtratoRepository struct {
	db *sqlx.DB
}

func NewExtratoRepository(db *sqlx.DB) *ExtratoRepository {
	return &ExtratoRepository{db: db}
}

type extratoImportacaoLinha struct {
	ID                 uuid.UUID  `db:"id"`
	ContaBancariaID    uuid.UUID  `db:"conta_bancaria_id"`
	ArquivoID          uuid.UUID  `db:"arquivo_id"`
	PeriodoInicio      time.Time  `db:"periodo_inicio"`
	PeriodoFim         time.Time  `db:"periodo_fim"`
	SaldoBancoCentavos *int64     `db:"saldo_banco_centavos"`
	SaldoBancoEm       *time.Time `db:"saldo_banco_em"`
	LinhasNovas        int        `db:"linhas_novas"`
	LinhasRepetidas    int        `db:"linhas_repetidas"`
	LinhasIgnoradas    int        `db:"linhas_ignoradas"`
	UsuarioID          uuid.UUID  `db:"usuario_id"`
	CriadoEm           time.Time  `db:"criado_em"`
}

type extratoLancamentoLinha struct {
	ID              uuid.UUID  `db:"id"`
	ContaBancariaID uuid.UUID  `db:"conta_bancaria_id"`
	ImportacaoID    uuid.UUID  `db:"importacao_id"`
	Sequencia       int        `db:"sequencia"`
	Data            time.Time  `db:"data"`
	ValorCentavos   int64      `db:"valor_centavos"`
	Descricao       string     `db:"descricao"`
	Documento       *string    `db:"documento"`
	Identificador   string     `db:"identificador"`
	CriadoEm        time.Time  `db:"criado_em"`
	ConciliacaoID   *uuid.UUID `db:"conciliacao_id"`
	Situacao        string     `db:"situacao"`
}

const consultaLancamentosComSituacao = `
	SELECT l.id, l.conta_bancaria_id, l.importacao_id, l.sequencia, l.data, l.valor_centavos, l.descricao, l.documento,
	       l.identificador, l.criado_em, co.id AS conciliacao_id,
	       CASE
	           WHEN co.id IS NULL THEN 'pendente'
	           WHEN EXISTS (SELECT 1 FROM conciliacao_item ci WHERE ci.conciliacao_id = co.id AND ` + alvoEstornado + `)
	               THEN 'invalidado'
	           ELSE 'conciliado'
	       END AS situacao
	FROM extrato_lancamento l
	LEFT JOIN conciliacao co ON co.extrato_lancamento_id = l.id AND co.vigente`

const (
	colunasExtratoImportacao = `id, conta_bancaria_id, arquivo_id, periodo_inicio, periodo_fim, saldo_banco_centavos,
		saldo_banco_em, linhas_novas, linhas_repetidas, linhas_ignoradas, usuario_id, criado_em`
	colunasExtratoLancamento = `id, conta_bancaria_id, importacao_id, sequencia, data, valor_centavos, descricao, documento,
		identificador, criado_em`
	colunasExtratoImportacaoComPrefixo = `i.id, i.conta_bancaria_id, i.arquivo_id, i.periodo_inicio, i.periodo_fim,
		i.saldo_banco_centavos, i.saldo_banco_em, i.linhas_novas, i.linhas_repetidas, i.linhas_ignoradas, i.usuario_id,
		i.criado_em`
	formatoDataDoBanco = "2006-01-02"
)

func (r *ExtratoRepository) ContasDoBanco(ctx context.Context, banco domainfinanceiro.Banco) ([]*domainfinanceiro.ContaBancaria, error) {
	var linhas []*contaBancariaLinha
	if err := r.db.SelectContext(ctx, &linhas, `
		SELECT `+colunasContaBancaria+` FROM conta_bancaria WHERE banco = $1 ORDER BY id
	`, banco.String()); err != nil {
		return nil, err
	}

	contas := make([]*domainfinanceiro.ContaBancaria, 0, len(linhas))
	for _, linha := range linhas {
		contas = append(contas, paraDominioContaBancaria(linha))
	}

	return contas, nil
}

func (r *ExtratoRepository) ArquivoImportado(ctx context.Context, arquivoID uuid.UUID) (bool, error) {
	var importado bool
	err := r.db.GetContext(ctx, &importado, `SELECT EXISTS (SELECT 1 FROM extrato_importacao WHERE arquivo_id = $1)`, arquivoID)

	return importado, err
}

func (r *ExtratoRepository) ContarExistentes(ctx context.Context, contaBancariaID uuid.UUID, identificadores []string) (int, error) {
	if len(identificadores) == 0 {
		return 0, nil
	}

	var existentes int
	err := r.db.GetContext(ctx, &existentes, `
		SELECT count(*) FROM extrato_lancamento
		WHERE conta_bancaria_id = $1 AND identificador = ANY($2::text[])
	`, contaBancariaID, pq.Array(identificadores))

	return existentes, err
}

func (r *ExtratoRepository) MesesFechados(ctx context.Context, contaBancariaID uuid.UUID) ([]time.Time, error) {
	return competenciasFechadas(ctx, r.db, contaBancariaID)
}

func (r *ExtratoRepository) Importar(ctx context.Context, preparado *domaincaixa.ExtratoPreparado, versaoDaAbertura int) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	importacao := preparado.Importacao
	if err := exigirContaAtivaNoCaixa(ctx, tx, importacao.ContaBancariaID); err != nil {
		return err
	}
	if err := conferirAberturaTravada(ctx, tx, importacao.ContaBancariaID, versaoDaAbertura); err != nil {
		return err
	}
	if err := recusarLinhasEmMesFechado(ctx, tx, importacao.ContaBancariaID, preparado.Lancamentos); err != nil {
		return err
	}

	if _, err := tx.NamedExecContext(ctx, `
		INSERT INTO extrato_importacao (`+colunasExtratoImportacao+`)
		VALUES (
			:id, :conta_bancaria_id, :arquivo_id, :periodo_inicio, :periodo_fim, :saldo_banco_centavos,
			:saldo_banco_em, :linhas_novas, :linhas_repetidas, :linhas_ignoradas, :usuario_id, :criado_em
		)
	`, paraLinhaExtratoImportacao(importacao)); err != nil {
		var erroPostgres *pq.Error
		if errors.As(err, &erroPostgres) && erroPostgres.Code == pqerror.UniqueViolation {
			return domaincaixa.ErroArquivoJaImportado()
		}
		return tratarErroDeGravacao(err)
	}

	novas, err := inserirLancamentosDoExtrato(ctx, tx, importacao, preparado.Lancamentos)
	if err != nil {
		return err
	}
	preparado.RegistrarNovas(novas)

	if _, err := tx.ExecContext(ctx, `
		UPDATE extrato_importacao SET linhas_novas = $1, linhas_repetidas = $2 WHERE id = $3
	`, importacao.LinhasNovas, importacao.LinhasRepetidas, importacao.ID); err != nil {
		return tratarErroDeGravacao(err)
	}

	if err := tx.Commit(); err != nil {
		return tratarErroDeGravacao(err)
	}

	return nil
}

func (r *ExtratoRepository) ObterImportacao(ctx context.Context, id uuid.UUID) (*domaincaixa.ImportacaoExtrato, error) {
	var linha extratoImportacaoLinha
	err := r.db.GetContext(ctx, &linha, `SELECT `+colunasExtratoImportacao+` FROM extrato_importacao WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return paraDominioExtratoImportacao(&linha), nil
}

func (r *ExtratoRepository) ListarImportacoes(ctx context.Context, filtro portsout.ImportacaoFiltro) ([]*domaincaixa.ImportacaoExtrato, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	query := `SELECT ` + colunasExtratoImportacaoComPrefixo + `
		FROM extrato_importacao i
		JOIN conta_bancaria c ON c.id = i.conta_bancaria_id`
	condicoes := make([]string, 0, 2)
	argumentos := make([]any, 0, 4)

	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		condicoes = append(condicoes, "c.empresa_id = ANY($"+strconv.Itoa(len(argumentos))+"::uuid[])")
	}
	if filtro.ContaBancariaID != nil {
		argumentos = append(argumentos, *filtro.ContaBancariaID)
		condicoes = append(condicoes, condicao("i.conta_bancaria_id", len(argumentos)))
	}
	if len(condicoes) > 0 {
		query += " WHERE " + unirCondicoes(condicoes)
	}

	argumentos = append(argumentos, filtro.Size, offset)
	query += ` ORDER BY i.criado_em DESC, i.id DESC` + limiteOffset(argumentos)

	var linhas []*extratoImportacaoLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	importacoes := make([]*domaincaixa.ImportacaoExtrato, 0, len(linhas))
	for _, linha := range linhas {
		importacoes = append(importacoes, paraDominioExtratoImportacao(linha))
	}

	return importacoes, nil
}

func (r *ExtratoRepository) DesfazerImportacao(ctx context.Context, importacao *domaincaixa.ImportacaoExtrato) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `SELECT id FROM conta_bancaria WHERE id = $1 FOR NO KEY UPDATE`, importacao.ContaBancariaID); err != nil {
		return tratarErro(err)
	}

	var ultima uuid.UUID
	err = tx.GetContext(ctx, &ultima, `
		SELECT id FROM extrato_importacao
		WHERE conta_bancaria_id = $1
		ORDER BY criado_em DESC, id DESC
		LIMIT 1
	`, importacao.ContaBancariaID)
	if errors.Is(err, sql.ErrNoRows) {
		return domaincaixa.ErroImportacaoNaoEncontrada()
	}
	if err != nil {
		return tratarErro(err)
	}
	if ultima != importacao.ID {
		var existe bool
		if err := tx.GetContext(ctx, &existe, `SELECT EXISTS (SELECT 1 FROM extrato_importacao WHERE id = $1)`, importacao.ID); err != nil {
			return tratarErro(err)
		}
		if !existe {
			return domaincaixa.ErroImportacaoNaoEncontrada()
		}
		return domaincaixa.ErroImportacaoNaoEhAUltima()
	}

	var conciliada bool
	if err := tx.GetContext(ctx, &conciliada, `
		SELECT EXISTS (
			SELECT 1 FROM conciliacao c
			JOIN extrato_lancamento l ON l.id = c.extrato_lancamento_id
			WHERE l.importacao_id = $1
		)
	`, importacao.ID); err != nil {
		return tratarErro(err)
	}
	if conciliada {
		return domaincaixa.ErroImportacaoComConciliacao()
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM extrato_lancamento WHERE importacao_id = $1`, importacao.ID); err != nil {
		return tratarErro(err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM extrato_importacao WHERE id = $1`, importacao.ID); err != nil {
		return tratarErro(err)
	}

	if err := tx.Commit(); err != nil {
		return tratarErro(err)
	}

	return nil
}

func (r *ExtratoRepository) ListarLancamentos(ctx context.Context, filtro portsout.LancamentoFiltro) ([]*domaincaixa.LancamentoExtrato, error) {
	filtro.PaginacaoFiltro = filtro.Normalizada()
	offset := (filtro.Page - 1) * filtro.Size

	argumentos := []any{filtro.ContaBancariaID}
	condicoes := []string{condicao("l.conta_bancaria_id", 1)}

	if filtro.De != nil {
		argumentos = append(argumentos, filtro.De.Format(formatoDataDoBanco))
		condicoes = append(condicoes, "l.data >= $"+strconv.Itoa(len(argumentos))+"::date")
	}
	if filtro.Ate != nil {
		argumentos = append(argumentos, filtro.Ate.Format(formatoDataDoBanco))
		condicoes = append(condicoes, "l.data <= $"+strconv.Itoa(len(argumentos))+"::date")
	}

	query := `SELECT * FROM (` + consultaLancamentosComSituacao + ` WHERE ` + unirCondicoes(condicoes) + `) lancamento`
	if filtro.Situacao != nil {
		argumentos = append(argumentos, filtro.Situacao.String())
		query += ` WHERE situacao = $` + strconv.Itoa(len(argumentos))
	}

	argumentos = append(argumentos, filtro.Size, offset)
	query += ` ORDER BY data ASC, criado_em ASC, sequencia ASC, id ASC` + limiteOffset(argumentos)

	var linhas []*extratoLancamentoLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	lancamentos := make([]*domaincaixa.LancamentoExtrato, 0, len(linhas))
	for _, linha := range linhas {
		lancamentos = append(lancamentos, paraDominioExtratoLancamento(linha))
	}

	return lancamentos, nil
}

func recusarLinhasEmMesFechado(
	ctx context.Context,
	tx *sqlx.Tx,
	contaBancariaID uuid.UUID,
	lancamentos []*domaincaixa.LancamentoExtrato,
) error {
	fechados, err := competenciasFechadas(ctx, tx, contaBancariaID)
	if err != nil {
		return tratarErroDeGravacao(err)
	}

	competencias := make(map[time.Time]bool, len(fechados))
	for _, competencia := range fechados {
		competencias[competencia] = true
	}
	for _, lancamento := range lancamentos {
		if competencias[domaincaixa.CompetenciaDe(lancamento.Data)] {
			return domaincaixa.ErroMesFechado()
		}
	}

	return nil
}

func conferirAberturaTravada(ctx context.Context, tx *sqlx.Tx, contaBancariaID uuid.UUID, versaoLida int) error {
	var versao int
	err := tx.GetContext(ctx, &versao, `SELECT versao FROM caixa_abertura WHERE conta_bancaria_id = $1 FOR SHARE`, contaBancariaID)
	if errors.Is(err, sql.ErrNoRows) {
		return domaincaixa.ErroContaSemAbertura()
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if versao != versaoLida {
		return domaincaixa.ErroAberturaAlterada()
	}

	return nil
}

func inserirLancamentosDoExtrato(
	ctx context.Context,
	tx *sqlx.Tx,
	importacao *domaincaixa.ImportacaoExtrato,
	lancamentos []*domaincaixa.LancamentoExtrato,
) (int, error) {
	if len(lancamentos) == 0 {
		return 0, nil
	}

	ids := make([]string, 0, len(lancamentos))
	sequencias := make([]int64, 0, len(lancamentos))
	datas := make([]string, 0, len(lancamentos))
	valores := make([]int64, 0, len(lancamentos))
	descricoes := make([]string, 0, len(lancamentos))
	documentos := make([]sql.NullString, 0, len(lancamentos))
	identificadores := make([]string, 0, len(lancamentos))

	for _, lancamento := range lancamentos {
		ids = append(ids, lancamento.ID.String())
		sequencias = append(sequencias, int64(lancamento.Sequencia))
		datas = append(datas, lancamento.Data.Format(formatoDataDoBanco))
		valores = append(valores, lancamento.ValorCentavos)
		descricoes = append(descricoes, lancamento.Descricao)
		documento := sql.NullString{}
		if lancamento.Documento != nil {
			documento = sql.NullString{String: *lancamento.Documento, Valid: true}
		}
		documentos = append(documentos, documento)
		identificadores = append(identificadores, lancamento.Identificador)
	}

	resultado, err := tx.ExecContext(ctx, `
		INSERT INTO extrato_lancamento (`+colunasExtratoLancamento+`)
		SELECT novo.id, $1, $2, novo.sequencia, novo.data, novo.valor_centavos, novo.descricao, novo.documento,
		       novo.identificador, $3
		FROM unnest($4::uuid[], $5::int[], $6::date[], $7::bigint[], $8::text[], $9::text[], $10::text[])
		    AS novo (id, sequencia, data, valor_centavos, descricao, documento, identificador)
		ON CONFLICT (conta_bancaria_id, identificador) DO NOTHING
	`,
		importacao.ContaBancariaID, importacao.ID, importacao.CriadoEm,
		pq.Array(ids), pq.Array(sequencias), pq.Array(datas), pq.Array(valores),
		pq.Array(descricoes), pq.Array(documentos), pq.Array(identificadores),
	)
	if err != nil {
		return 0, tratarErroDeGravacao(err)
	}

	novas, err := resultado.RowsAffected()
	if err != nil {
		return 0, tratarErroDeGravacao(err)
	}

	return int(novas), nil
}

func paraLinhaExtratoImportacao(importacao *domaincaixa.ImportacaoExtrato) *extratoImportacaoLinha {
	return &extratoImportacaoLinha{
		ID:                 importacao.ID,
		ContaBancariaID:    importacao.ContaBancariaID,
		ArquivoID:          importacao.ArquivoID,
		PeriodoInicio:      importacao.PeriodoInicio,
		PeriodoFim:         importacao.PeriodoFim,
		SaldoBancoCentavos: importacao.SaldoBancoCentavos,
		SaldoBancoEm:       importacao.SaldoBancoEm,
		LinhasNovas:        importacao.LinhasNovas,
		LinhasRepetidas:    importacao.LinhasRepetidas,
		LinhasIgnoradas:    importacao.LinhasIgnoradas,
		UsuarioID:          importacao.UsuarioID,
		CriadoEm:           importacao.CriadoEm,
	}
}

func paraDominioExtratoImportacao(linha *extratoImportacaoLinha) *domaincaixa.ImportacaoExtrato {
	importacao := &domaincaixa.ImportacaoExtrato{
		ID:                 linha.ID,
		ContaBancariaID:    linha.ContaBancariaID,
		ArquivoID:          linha.ArquivoID,
		PeriodoInicio:      domaincaixa.DataDe(linha.PeriodoInicio),
		PeriodoFim:         domaincaixa.DataDe(linha.PeriodoFim),
		SaldoBancoCentavos: linha.SaldoBancoCentavos,
		LinhasNovas:        linha.LinhasNovas,
		LinhasRepetidas:    linha.LinhasRepetidas,
		LinhasIgnoradas:    linha.LinhasIgnoradas,
		UsuarioID:          linha.UsuarioID,
		CriadoEm:           linha.CriadoEm,
	}
	if linha.SaldoBancoEm != nil {
		em := domaincaixa.DataDe(*linha.SaldoBancoEm)
		importacao.SaldoBancoEm = &em
	}

	return importacao
}

func paraDominioExtratoLancamento(linha *extratoLancamentoLinha) *domaincaixa.LancamentoExtrato {
	return &domaincaixa.LancamentoExtrato{
		ID:              linha.ID,
		ContaBancariaID: linha.ContaBancariaID,
		ImportacaoID:    linha.ImportacaoID,
		Sequencia:       linha.Sequencia,
		Data:            domaincaixa.DataDe(linha.Data),
		ValorCentavos:   linha.ValorCentavos,
		Descricao:       linha.Descricao,
		Documento:       linha.Documento,
		Identificador:   linha.Identificador,
		CriadoEm:        linha.CriadoEm,
		ConciliacaoID:   linha.ConciliacaoID,
		Situacao:        situacaoDoLancamento(linha.Situacao),
	}
}

func situacaoDoLancamento(valor string) domaincaixa.SituacaoLancamento {
	if valor == "" {
		return domaincaixa.LancamentoPendente
	}

	return domaincaixa.SituacaoLancamentoDe(valor)
}
