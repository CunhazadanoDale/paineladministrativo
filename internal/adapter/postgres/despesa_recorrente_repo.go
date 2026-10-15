package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/solicitacao"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ portsout.DespesaRecorrenteRepository = (*DespesaRecorrenteRepository)(nil)

type DespesaRecorrenteRepository struct {
	db     *sqlx.DB
	guarda guardaDeVersao
}

func NewDespesaRecorrenteRepository(db *sqlx.DB) *DespesaRecorrenteRepository {
	return &DespesaRecorrenteRepository{
		db: db,
		guarda: guardaDeVersao{
			tabela:        "despesa_recorrente",
			naoEncontrado: domainsolicitacao.ErroDespesaRecorrenteNaoEncontrada,
			alterado:      domainsolicitacao.ErroDespesaRecorrenteAlterada,
		},
	}
}

type despesaRecorrenteLinha struct {
	ID                    uuid.UUID  `db:"id"`
	EmpresaID             uuid.UUID  `db:"empresa_id"`
	FornecedorID          *uuid.UUID `db:"fornecedor_id"`
	CategoriaFinanceiraID uuid.UUID  `db:"categoria_financeira_id"`
	Descricao             string     `db:"descricao"`
	ValorCentavos         int64      `db:"valor_centavos"`
	Estimado              bool       `db:"estimado"`
	FormaPagamento        string     `db:"forma_pagamento"`
	DiaVencimento         int        `db:"dia_vencimento"`
	VencimentoMesSeguinte bool       `db:"vencimento_mes_seguinte"`
	Inicio                time.Time  `db:"inicio"`
	Fim                   *time.Time `db:"fim"`
	Ativa                 bool       `db:"ativa"`
	Versao                int        `db:"versao"`
	CriadoEm              time.Time  `db:"criado_em"`
	AtualizadoEm          time.Time  `db:"atualizado_em"`
}

const colunasDespesaRecorrente = `id, empresa_id, fornecedor_id, categoria_financeira_id, descricao, valor_centavos, estimado,
	forma_pagamento, dia_vencimento, vencimento_mes_seguinte, inicio, fim, ativa, versao, criado_em, atualizado_em`

func (r *DespesaRecorrenteRepository) Criar(ctx context.Context, despesa *domainsolicitacao.DespesaRecorrente) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := exigirEmpresaAtivaParaRecorrencia(ctx, tx, despesa.EmpresaID); err != nil {
		return err
	}
	if err := exigirClassificacaoDaRecorrencia(ctx, tx, despesa); err != nil {
		return err
	}

	if _, err := tx.NamedExecContext(ctx, `
		INSERT INTO despesa_recorrente (`+colunasDespesaRecorrente+`)
		VALUES (
			:id, :empresa_id, :fornecedor_id, :categoria_financeira_id, :descricao, :valor_centavos, :estimado,
			:forma_pagamento, :dia_vencimento, :vencimento_mes_seguinte, :inicio, :fim, :ativa, :versao, :criado_em, :atualizado_em
		)
	`, paraLinhaDespesaRecorrente(despesa)); err != nil {
		return tratarErroDeGravacao(err)
	}

	return tratarErroDeGravacao(tx.Commit())
}

func (r *DespesaRecorrenteRepository) Obter(ctx context.Context, id uuid.UUID) (*domainsolicitacao.DespesaRecorrente, error) {
	var linha despesaRecorrenteLinha
	err := r.db.GetContext(ctx, &linha, `SELECT `+colunasDespesaRecorrente+` FROM despesa_recorrente WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return paraDominioDespesaRecorrente(&linha), nil
}

func (r *DespesaRecorrenteRepository) Listar(ctx context.Context, filtro portsout.DespesaRecorrenteFiltro) ([]*domainsolicitacao.DespesaRecorrente, error) {
	query := `SELECT ` + colunasDespesaRecorrente + ` FROM despesa_recorrente`
	condicoes := make([]string, 0, 3)
	argumentos := make([]any, 0, 5)

	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		condicoes = append(condicoes, "empresa_id = ANY($"+strconv.Itoa(len(argumentos))+"::uuid[])")
	}
	if filtro.Ativa != nil {
		argumentos = append(argumentos, *filtro.Ativa)
		condicoes = append(condicoes, condicao("ativa", len(argumentos)))
	}
	if filtro.Fornecedor != nil {
		argumentos = append(argumentos, *filtro.Fornecedor)
		condicoes = append(condicoes, condicao("fornecedor_id", len(argumentos)))
	}
	if len(condicoes) > 0 {
		query += " WHERE " + unirCondicoes(condicoes)
	}

	argumentos = append(argumentos, filtro.Size, (filtro.Page-1)*filtro.Size)
	query += ` ORDER BY lower(descricao), id` + limiteOffset(argumentos)

	return r.selecionar(ctx, query, argumentos...)
}

func (r *DespesaRecorrenteRepository) ListarVigentes(ctx context.Context, competencia domainsolicitacao.Competencia, empresaIDs []uuid.UUID) ([]*domainsolicitacao.DespesaRecorrente, error) {
	query := `
		SELECT ` + colunasDespesaRecorrente + `
		FROM despesa_recorrente
		WHERE ativa
		  AND inicio <= $1
		  AND (fim IS NULL OR fim >= $1)
		  AND empresa_id IN (SELECT id FROM empresa WHERE ativa)
	`
	argumentos := []any{competencia.Inicio()}
	if empresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(empresaIDs)))
		query += ` AND empresa_id = ANY($2::uuid[])`
	}
	query += ` ORDER BY empresa_id, lower(descricao), id`

	return r.selecionar(ctx, query, argumentos...)
}

func (r *DespesaRecorrenteRepository) Atualizar(ctx context.Context, despesa *domainsolicitacao.DespesaRecorrente) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := r.guarda.conferirVersaoTravada(ctx, tx, despesa.ID, despesa.Versao); err != nil {
		return err
	}
	if err := exigirClassificacaoDaRecorrencia(ctx, tx, despesa); err != nil {
		return err
	}

	linha := paraLinhaDespesaRecorrente(despesa)
	versao, err := r.guarda.atualizar(ctx, tx, despesa.ID, `
		UPDATE despesa_recorrente
		SET fornecedor_id = $3,
		    categoria_financeira_id = $4,
		    descricao = $5,
		    valor_centavos = $6,
		    estimado = $7,
		    forma_pagamento = $8,
		    dia_vencimento = $9,
		    vencimento_mes_seguinte = $10,
		    inicio = $11,
		    fim = $12,
		    atualizado_em = $13,
		    versao = versao + 1
		WHERE id = $1 AND versao = $2
		RETURNING versao
	`, linha.ID, linha.Versao, linha.FornecedorID, linha.CategoriaFinanceiraID, linha.Descricao, linha.ValorCentavos,
		linha.Estimado, linha.FormaPagamento, linha.DiaVencimento, linha.VencimentoMesSeguinte, linha.Inicio, linha.Fim,
		linha.AtualizadoEm)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return tratarErroDeGravacao(err)
	}
	despesa.Versao = versao

	return nil
}

func (r *DespesaRecorrenteRepository) Ativar(ctx context.Context, despesa *domainsolicitacao.DespesaRecorrente) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := exigirEmpresaAtivaParaRecorrencia(ctx, tx, despesa.EmpresaID); err != nil {
		return err
	}

	versao, err := r.trocarAtiva(ctx, tx, despesa)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return tratarErroDeGravacao(err)
	}
	despesa.Versao = versao

	return nil
}

func (r *DespesaRecorrenteRepository) Desativar(ctx context.Context, despesa *domainsolicitacao.DespesaRecorrente) error {
	versao, err := r.trocarAtiva(ctx, r.db, despesa)
	if err != nil {
		return err
	}
	despesa.Versao = versao

	return nil
}

func (r *DespesaRecorrenteRepository) Remover(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErro(err)
	}
	defer func() { _ = tx.Rollback() }()

	var travada uuid.UUID
	err = tx.GetContext(ctx, &travada, `SELECT id FROM despesa_recorrente WHERE id = $1 FOR UPDATE`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return domainsolicitacao.ErroDespesaRecorrenteNaoEncontrada()
	}
	if err != nil {
		return tratarErro(err)
	}

	var gerou bool
	if err := tx.GetContext(ctx, &gerou, `
		SELECT EXISTS (SELECT 1 FROM solicitacao WHERE origem = 'recorrencia' AND origem_id = $1)
	`, id); err != nil {
		return tratarErro(err)
	}
	if gerou {
		return domainsolicitacao.ErroDespesaRecorrenteComSolicitacoes()
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM despesa_recorrente WHERE id = $1`, id); err != nil {
		return tratarErro(err)
	}

	return tratarErro(tx.Commit())
}

func (r *DespesaRecorrenteRepository) trocarAtiva(ctx context.Context, consulta sqlx.QueryerContext, despesa *domainsolicitacao.DespesaRecorrente) (int, error) {
	return r.guarda.atualizar(ctx, consulta, despesa.ID, `
		UPDATE despesa_recorrente
		SET ativa = $3,
		    atualizado_em = $4,
		    versao = versao + 1
		WHERE id = $1 AND versao = $2
		RETURNING versao
	`, despesa.ID, despesa.Versao, despesa.Ativa, despesa.AtualizadoEm)
}

func (r *DespesaRecorrenteRepository) selecionar(ctx context.Context, query string, argumentos ...any) ([]*domainsolicitacao.DespesaRecorrente, error) {
	var linhas []*despesaRecorrenteLinha
	if err := r.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	despesas := make([]*domainsolicitacao.DespesaRecorrente, 0, len(linhas))
	for _, linha := range linhas {
		despesas = append(despesas, paraDominioDespesaRecorrente(linha))
	}

	return despesas, nil
}

func exigirEmpresaAtivaParaRecorrencia(ctx context.Context, tx *sqlx.Tx, empresaID uuid.UUID) error {
	var ativa bool
	err := tx.GetContext(ctx, &ativa, `SELECT ativa FROM empresa WHERE id = $1 FOR SHARE`, empresaID)
	if errors.Is(err, sql.ErrNoRows) {
		return domainsolicitacao.ErroEmpresaDaRecorrenciaNaoEncontrada()
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if !ativa {
		return domainsolicitacao.ErroEmpresaInativaNaRecorrencia()
	}

	return nil
}

func exigirClassificacaoDaRecorrencia(ctx context.Context, tx *sqlx.Tx, despesa *domainsolicitacao.DespesaRecorrente) error {
	if err := exigirFornecedorAtivo(ctx, tx, despesa.FornecedorID); err != nil {
		return err
	}

	categoriaID := despesa.CategoriaFinanceiraID

	return exigirCategoriaDeDespesaLancavel(ctx, tx, &categoriaID)
}

func paraLinhaDespesaRecorrente(despesa *domainsolicitacao.DespesaRecorrente) despesaRecorrenteLinha {
	linha := despesaRecorrenteLinha{
		ID:                    despesa.ID,
		EmpresaID:             despesa.EmpresaID,
		FornecedorID:          despesa.FornecedorID,
		CategoriaFinanceiraID: despesa.CategoriaFinanceiraID,
		Descricao:             despesa.Descricao.Texto(),
		ValorCentavos:         despesa.Valor.Centavos(),
		Estimado:              despesa.Estimado,
		FormaPagamento:        despesa.FormaPagamento.String(),
		DiaVencimento:         despesa.DiaVencimento.Dia(),
		VencimentoMesSeguinte: despesa.VencimentoMesSeguinte,
		Inicio:                despesa.Inicio.Inicio(),
		Ativa:                 despesa.Ativa,
		Versao:                despesa.Versao,
		CriadoEm:              despesa.CriadoEm,
		AtualizadoEm:          despesa.AtualizadoEm,
	}
	if despesa.Fim != nil {
		fim := despesa.Fim.Inicio()
		linha.Fim = &fim
	}

	return linha
}

func paraDominioDespesaRecorrente(linha *despesaRecorrenteLinha) *domainsolicitacao.DespesaRecorrente {
	despesa := &domainsolicitacao.DespesaRecorrente{
		ID:                    linha.ID,
		EmpresaID:             linha.EmpresaID,
		FornecedorID:          linha.FornecedorID,
		CategoriaFinanceiraID: linha.CategoriaFinanceiraID,
		Descricao:             domainsolicitacao.ObservacaoDe(linha.Descricao),
		Valor:                 domainsolicitacao.ValorDe(linha.ValorCentavos),
		Estimado:              linha.Estimado,
		FormaPagamento:        domainsolicitacao.FormaPagamento(linha.FormaPagamento),
		DiaVencimento:         domainsolicitacao.DiaVencimentoDe(linha.DiaVencimento),
		VencimentoMesSeguinte: linha.VencimentoMesSeguinte,
		Inicio:                domainsolicitacao.CompetenciaDe(linha.Inicio),
		Ativa:                 linha.Ativa,
		Versao:                linha.Versao,
		CriadoEm:              linha.CriadoEm,
		AtualizadoEm:          linha.AtualizadoEm,
	}
	if linha.Fim != nil {
		fim := domainsolicitacao.CompetenciaDe(*linha.Fim)
		despesa.Fim = &fim
	}

	return despesa
}
