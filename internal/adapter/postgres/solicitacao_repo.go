package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/solicitacao"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ portsout.SolicitacaoRepository = (*SolicitacaoRepository)(nil)

type SolicitacaoRepository struct {
	db     *sqlx.DB
	guarda guardaDeVersao
}

func NewSolicitacaoRepository(db *sqlx.DB) *SolicitacaoRepository {
	return &SolicitacaoRepository{
		db: db,
		guarda: guardaDeVersao{
			tabela:        "solicitacao",
			naoEncontrado: func() error { return domain.ErrNotFound },
			alterado:      domainsolicitacao.ErroSolicitacaoAlterada,
		},
	}
}

type solicitacaoLinha struct {
	ID                    uuid.UUID  `db:"id"`
	SolicitanteID         uuid.UUID  `db:"solicitante_id"`
	EmpresaID             uuid.UUID  `db:"empresa_id"`
	AprovadorID           *uuid.UUID `db:"aprovador_id"`
	ValorCentavos         int64      `db:"valor_estimado"`
	PrazoPagamento        time.Time  `db:"prazo_pagamento"`
	Observacao            string     `db:"observacao"`
	FormaPagamento        string     `db:"forma_pagamento"`
	FornecedorID          *uuid.UUID `db:"fornecedor_id"`
	CategoriaFinanceiraID *uuid.UUID `db:"categoria_financeira_id"`
	Competencia           time.Time  `db:"competencia"`
	Estimado              bool       `db:"estimado"`
	Origem                string     `db:"origem"`
	OrigemID              *uuid.UUID `db:"origem_id"`
	Status                string     `db:"status"`
	RejeitacaoMotivo      *string    `db:"rejeitacao_motivo"`
	AprovadoEm            *time.Time `db:"aprovado_em"`
	RejeitadoEm           *time.Time `db:"rejeitado_em"`
	CanceladoEm           *time.Time `db:"cancelado_em"`
	Versao                int        `db:"versao"`
	CriadoEm              time.Time  `db:"criado_em"`
	AtualizadoEm          time.Time  `db:"atualizado_em"`
}

const colunasSolicitacao = `
	id, solicitante_id, empresa_id, aprovador_id, valor_estimado, prazo_pagamento, observacao,
	forma_pagamento, fornecedor_id, categoria_financeira_id, competencia, estimado, origem, origem_id,
	status, rejeitacao_motivo, aprovado_em, rejeitado_em, cancelado_em, versao, criado_em, atualizado_em
`

func (s *SolicitacaoRepository) Criar(ctx context.Context, solicitacao *domainsolicitacao.Solicitacao, arquivoIDs []uuid.UUID, historico *domainsolicitacao.Historico) (uuid.UUID, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return uuid.Nil, tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := exigirEmpresaAtivaParaSolicitacao(ctx, tx, solicitacao.EmpresaID); err != nil {
		return uuid.Nil, err
	}
	if err := exigirClassificacaoGravavel(ctx, tx, solicitacao); err != nil {
		return uuid.Nil, err
	}

	linha := paraLinha(solicitacao)
	query := `
		INSERT INTO solicitacao (` + colunasSolicitacao + `) VALUES (
			:id, :solicitante_id, :empresa_id, :aprovador_id, :valor_estimado, :prazo_pagamento, :observacao,
			:forma_pagamento, :fornecedor_id, :categoria_financeira_id, :competencia, :estimado, :origem, :origem_id,
			:status, :rejeitacao_motivo, :aprovado_em, :rejeitado_em, :cancelado_em, :versao, :criado_em, :atualizado_em
		)
	`

	if _, err := tx.NamedExecContext(ctx, query, linha); err != nil {
		return uuid.Nil, tratarErroDeGravacao(err)
	}

	if err := inserirHistorico(ctx, tx, historico); err != nil {
		return uuid.Nil, tratarErroDeGravacao(err)
	}

	for _, arquivoID := range arquivoIDs {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO solicitacao_arquivo (solicitacao_id, arquivo_id) VALUES ($1, $2)
		`, solicitacao.ID, arquivoID); err != nil {
			return uuid.Nil, tratarErroDeGravacao(err)
		}
	}

	if err := tx.Commit(); err != nil {
		return uuid.Nil, tratarErroDeGravacao(err)
	}

	return solicitacao.ID, nil
}

func (s *SolicitacaoRepository) GerarDeRecorrencia(ctx context.Context, solicitacao *domainsolicitacao.Solicitacao, historico *domainsolicitacao.Historico) (bool, error) {
	if solicitacao.Origem != domainsolicitacao.OrigemRecorrencia || solicitacao.OrigemID == nil {
		return false, domain.ErroValidacao("solicitação gerada precisa da despesa recorrente de origem")
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return false, tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	var ativa bool
	err = tx.GetContext(ctx, &ativa, `SELECT ativa FROM despesa_recorrente WHERE id = $1 FOR SHARE`, *solicitacao.OrigemID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !ativa) {
		return false, domainsolicitacao.ErroDespesaRecorrenteIndisponivel()
	}
	if err != nil {
		return false, tratarErroDeGravacao(err)
	}

	if err := exigirEmpresaAtivaParaSolicitacao(ctx, tx, solicitacao.EmpresaID); err != nil {
		return false, err
	}
	if err := exigirClassificacaoGravavel(ctx, tx, solicitacao); err != nil {
		return false, err
	}

	resultado, err := tx.NamedExecContext(ctx, `
		INSERT INTO solicitacao (`+colunasSolicitacao+`) VALUES (
			:id, :solicitante_id, :empresa_id, :aprovador_id, :valor_estimado, :prazo_pagamento, :observacao,
			:forma_pagamento, :fornecedor_id, :categoria_financeira_id, :competencia, :estimado, :origem, :origem_id,
			:status, :rejeitacao_motivo, :aprovado_em, :rejeitado_em, :cancelado_em, :versao, :criado_em, :atualizado_em
		)
		ON CONFLICT (origem_id, competencia) WHERE origem = 'recorrencia' DO NOTHING
	`, paraLinha(solicitacao))
	if err != nil {
		return false, tratarErroDeGravacao(err)
	}
	linhas, err := resultado.RowsAffected()
	if err != nil {
		return false, tratarErroDeGravacao(err)
	}
	if linhas == 0 {
		return false, nil
	}

	if err := inserirHistorico(ctx, tx, historico); err != nil {
		return false, tratarErroDeGravacao(err)
	}
	if err := tx.Commit(); err != nil {
		return false, tratarErroDeGravacao(err)
	}

	return true, nil
}

func (s *SolicitacaoRepository) Obter(ctx context.Context, id uuid.UUID) (*domainsolicitacao.Solicitacao, error) {
	query := `SELECT ` + colunasSolicitacao + ` FROM solicitacao WHERE id = $1`

	var linha solicitacaoLinha
	if err := s.db.GetContext(ctx, &linha, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return paraDominio(&linha), nil
}

func (s *SolicitacaoRepository) Listar(ctx context.Context, filtro portsout.SolicitacaoFiltro) ([]*domainsolicitacao.Solicitacao, error) {
	offset := (filtro.Page - 1) * filtro.Size

	onde, argumentos := condicoesDaSolicitacao(filtro)
	query := `SELECT ` + colunasSolicitacao + ` FROM solicitacao` + onde

	argumentos = append(argumentos, filtro.Size, offset)
	query += ` ORDER BY criado_em DESC, id DESC` + limiteOffset(argumentos)

	var linhas []*solicitacaoLinha
	if err := s.db.SelectContext(ctx, &linhas, query, argumentos...); err != nil {
		return nil, err
	}

	itens := make([]*domainsolicitacao.Solicitacao, 0, len(linhas))
	for _, linha := range linhas {
		itens = append(itens, paraDominio(linha))
	}

	return itens, nil
}

func (s *SolicitacaoRepository) Contar(ctx context.Context, filtro portsout.SolicitacaoFiltro) (int, error) {
	onde, argumentos := condicoesDaSolicitacao(filtro)

	var total int
	if err := s.db.GetContext(ctx, &total, `SELECT count(*) FROM solicitacao`+onde, argumentos...); err != nil {
		return 0, err
	}

	return total, nil
}

func condicoesDaSolicitacao(filtro portsout.SolicitacaoFiltro) (string, []any) {
	condicoes := make([]string, 0, 2)
	argumentos := make([]any, 0, 3)

	if filtro.Solicitante != nil {
		argumentos = append(argumentos, *filtro.Solicitante)
		condicoes = append(condicoes, condicao("solicitante_id", len(argumentos)))
	}
	if filtro.Status != "" {
		argumentos = append(argumentos, filtro.Status)
		condicoes = append(condicoes, condicao("status", len(argumentos)))
	}
	if filtro.Empresa != nil {
		argumentos = append(argumentos, *filtro.Empresa)
		condicoes = append(condicoes, condicao("empresa_id", len(argumentos)))
	}
	if filtro.EmpresaIDs != nil {
		argumentos = append(argumentos, pq.Array(textosDeIDs(filtro.EmpresaIDs)))
		condicoes = append(condicoes, "empresa_id = ANY($"+strconv.Itoa(len(argumentos))+"::uuid[])")
	}
	if filtro.Fornecedor != nil {
		argumentos = append(argumentos, *filtro.Fornecedor)
		condicoes = append(condicoes, condicao("fornecedor_id", len(argumentos)))
	}
	if filtro.Categoria != nil {
		argumentos = append(argumentos, *filtro.Categoria)
		condicoes = append(condicoes, condicao("categoria_financeira_id", len(argumentos)))
	}
	if filtro.Competencia != nil {
		argumentos = append(argumentos, filtro.Competencia.Inicio())
		condicoes = append(condicoes, condicao("competencia", len(argumentos)))
	}
	if filtro.Origem != "" {
		argumentos = append(argumentos, filtro.Origem.String())
		condicoes = append(condicoes, condicao("origem", len(argumentos)))
	}
	if filtro.VenceDe != nil {
		argumentos = append(argumentos, *filtro.VenceDe)
		condicoes = append(condicoes, "prazo_pagamento >= $"+strconv.Itoa(len(argumentos)))
	}
	if filtro.VenceAte != nil {
		argumentos = append(argumentos, *filtro.VenceAte)
		condicoes = append(condicoes, "prazo_pagamento <= $"+strconv.Itoa(len(argumentos)))
	}
	if filtro.VencidasEm != nil {
		argumentos = append(argumentos, *filtro.VencidasEm)
		condicoes = append(condicoes, "prazo_pagamento < $"+strconv.Itoa(len(argumentos))+
			" AND status IN ('pendente_aprovacao', 'aprovado')")
	}
	if len(condicoes) == 0 {
		return "", argumentos
	}

	return " WHERE " + unirCondicoes(condicoes), argumentos
}

func (s *SolicitacaoRepository) AtualizarStatus(ctx context.Context, solicitacao *domainsolicitacao.Solicitacao, historico *domainsolicitacao.Historico) (bool, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return false, tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	atualizado, err := atualizarStatusComGuarda(ctx, tx, solicitacao, historico.DeStatus)
	if err != nil {
		return false, tratarErroDeGravacao(err)
	}
	if !atualizado {
		return false, nil
	}

	if err := inserirHistorico(ctx, tx, historico); err != nil {
		return false, tratarErroDeGravacao(err)
	}

	if err := tx.Commit(); err != nil {
		return false, tratarErroDeGravacao(err)
	}

	return true, nil
}

func (s *SolicitacaoRepository) Classificar(ctx context.Context, solicitacao *domainsolicitacao.Solicitacao, versaoLida int, historico *domainsolicitacao.Historico) error {
	return s.gravarComGuarda(ctx, solicitacao, historico, true, `
		UPDATE solicitacao
		SET fornecedor_id = $3,
		    categoria_financeira_id = $4,
		    competencia = $5,
		    atualizado_em = $6,
		    versao = versao + 1
		WHERE id = $1 AND versao = $2 AND status = 'aprovado'
		RETURNING versao
	`, solicitacao.ID, versaoLida, solicitacao.FornecedorID, solicitacao.CategoriaFinanceiraID,
		solicitacao.Competencia.Inicio(), solicitacao.AtualizadoEm)
}

func (s *SolicitacaoRepository) Editar(ctx context.Context, solicitacao *domainsolicitacao.Solicitacao, versaoLida int, historico *domainsolicitacao.Historico) error {
	return s.gravarComGuarda(ctx, solicitacao, historico, true, `
		UPDATE solicitacao
		SET valor_estimado = $3,
		    prazo_pagamento = $4,
		    observacao = $5,
		    forma_pagamento = $6,
		    fornecedor_id = $7,
		    categoria_financeira_id = $8,
		    competencia = $9,
		    estimado = $10,
		    atualizado_em = $11,
		    versao = versao + 1
		WHERE id = $1 AND versao = $2 AND status = 'pendente_aprovacao'
		RETURNING versao
	`, solicitacao.ID, versaoLida, solicitacao.Valor.Centavos(), solicitacao.Prazo.Data(), solicitacao.Observacao.Texto(),
		solicitacao.FormaPagamento.String(), solicitacao.FornecedorID, solicitacao.CategoriaFinanceiraID,
		solicitacao.Competencia.Inicio(), solicitacao.Estimado, solicitacao.AtualizadoEm)
}

func (s *SolicitacaoRepository) Adiar(ctx context.Context, solicitacao *domainsolicitacao.Solicitacao, versaoLida int, historico *domainsolicitacao.Historico) error {
	return s.gravarComGuarda(ctx, solicitacao, historico, false, `
		UPDATE solicitacao
		SET prazo_pagamento = $3,
		    atualizado_em = $4,
		    versao = versao + 1
		WHERE id = $1 AND versao = $2 AND status = 'aprovado'
		RETURNING versao
	`, solicitacao.ID, versaoLida, solicitacao.Prazo.Data(), solicitacao.AtualizadoEm)
}

func (s *SolicitacaoRepository) gravarComGuarda(
	ctx context.Context,
	solicitacao *domainsolicitacao.Solicitacao,
	historico *domainsolicitacao.Historico,
	conferirClassificacao bool,
	query string,
	argumentos ...any,
) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	defer func() { _ = tx.Rollback() }()

	if conferirClassificacao {
		if err := exigirClassificacaoGravavel(ctx, tx, solicitacao); err != nil {
			return err
		}
	}

	if _, err := s.guarda.atualizar(ctx, tx, solicitacao.ID, query, argumentos...); err != nil {
		return err
	}

	if err := inserirHistorico(ctx, tx, historico); err != nil {
		return tratarErroDeGravacao(err)
	}

	return tratarErroDeGravacao(tx.Commit())
}

func (s *SolicitacaoRepository) ListarArquivos(ctx context.Context, solicitacaoID uuid.UUID) ([]*domainsolicitacao.Arquivo, error) {
	query := `
		SELECT a.id, a.proprietario_id, a.nome, a.chave, a.content_type, a.tamanho, a.criado_em
		FROM arquivo a
		JOIN solicitacao_arquivo sa ON sa.arquivo_id = a.id
		WHERE sa.solicitacao_id = $1
		ORDER BY a.criado_em ASC, a.id ASC
	`

	var itens []*domainsolicitacao.Arquivo
	if err := s.db.SelectContext(ctx, &itens, query, solicitacaoID); err != nil {
		return nil, err
	}
	return itens, nil
}

func (s *SolicitacaoRepository) ListarHistorico(ctx context.Context, solicitacaoID uuid.UUID) ([]*domainsolicitacao.Historico, error) {
	query := `
		SELECT id, solicitacao_id, usuario_id, de_status, para_status, evento, descricao, criado_em
		FROM solicitacao_historico
		WHERE solicitacao_id = $1
		ORDER BY criado_em DESC, id DESC
	`

	var itens []*domainsolicitacao.Historico
	if err := s.db.SelectContext(ctx, &itens, query, solicitacaoID); err != nil {
		return nil, err
	}
	return itens, nil
}

func paraLinha(solicitacao *domainsolicitacao.Solicitacao) solicitacaoLinha {
	return solicitacaoLinha{
		ID:                    solicitacao.ID,
		SolicitanteID:         solicitacao.SolicitanteID,
		EmpresaID:             solicitacao.EmpresaID,
		AprovadorID:           solicitacao.AprovadorID,
		ValorCentavos:         solicitacao.Valor.Centavos(),
		PrazoPagamento:        solicitacao.Prazo.Data(),
		Observacao:            solicitacao.Observacao.Texto(),
		FormaPagamento:        string(solicitacao.FormaPagamento),
		FornecedorID:          solicitacao.FornecedorID,
		CategoriaFinanceiraID: solicitacao.CategoriaFinanceiraID,
		Competencia:           solicitacao.Competencia.Inicio(),
		Estimado:              solicitacao.Estimado,
		Origem:                solicitacao.Origem.String(),
		OrigemID:              solicitacao.OrigemID,
		Status:                string(solicitacao.Status),
		RejeitacaoMotivo:      motivoRejeicao(solicitacao),
		AprovadoEm:            solicitacao.AprovadoEm,
		RejeitadoEm:           solicitacao.RejeitadoEm,
		CanceladoEm:           solicitacao.CanceladoEm,
		Versao:                solicitacao.Versao,
		CriadoEm:              solicitacao.CriadoEm,
		AtualizadoEm:          solicitacao.AtualizadoEm,
	}
}

func paraDominio(linha *solicitacaoLinha) *domainsolicitacao.Solicitacao {
	solicitacao := &domainsolicitacao.Solicitacao{
		ID:                    linha.ID,
		SolicitanteID:         linha.SolicitanteID,
		EmpresaID:             linha.EmpresaID,
		AprovadorID:           linha.AprovadorID,
		Valor:                 domainsolicitacao.ValorDe(linha.ValorCentavos),
		Prazo:                 domainsolicitacao.PrazoPagamentoDe(linha.PrazoPagamento),
		Observacao:            domainsolicitacao.ObservacaoDe(linha.Observacao),
		FormaPagamento:        domainsolicitacao.FormaPagamento(linha.FormaPagamento),
		FornecedorID:          linha.FornecedorID,
		CategoriaFinanceiraID: linha.CategoriaFinanceiraID,
		Competencia:           domainsolicitacao.CompetenciaDe(linha.Competencia),
		Estimado:              linha.Estimado,
		Origem:                domainsolicitacao.OrigemDe(linha.Origem),
		OrigemID:              linha.OrigemID,
		Status:                domainsolicitacao.StatusDe(linha.Status),
		AprovadoEm:            linha.AprovadoEm,
		RejeitadoEm:           linha.RejeitadoEm,
		CanceladoEm:           linha.CanceladoEm,
		Versao:                linha.Versao,
		CriadoEm:              linha.CriadoEm,
		AtualizadoEm:          linha.AtualizadoEm,
	}

	if linha.RejeitacaoMotivo != nil {
		motivo := domainsolicitacao.MotivoRejeicaoDe(*linha.RejeitacaoMotivo)
		solicitacao.MotivoRejeicao = &motivo
	}

	return solicitacao
}

func motivoRejeicao(solicitacao *domainsolicitacao.Solicitacao) *string {
	if solicitacao.MotivoRejeicao == nil {
		return nil
	}

	texto := solicitacao.MotivoRejeicao.Texto()

	return &texto
}

func inserirHistorico(ctx context.Context, tx *sqlx.Tx, historico *domainsolicitacao.Historico) error {
	_, err := tx.NamedExecContext(ctx, `
		INSERT INTO solicitacao_historico (id, solicitacao_id, usuario_id, de_status, para_status, evento, descricao, criado_em)
		VALUES (:id, :solicitacao_id, :usuario_id, :de_status, :para_status, :evento, :descricao, :criado_em)
	`, historico)

	return err
}

type transacional interface {
	NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error)
}

func atualizarStatusComGuarda(ctx context.Context, tx transacional, solicitacao *domainsolicitacao.Solicitacao, statusAnterior *domainsolicitacao.Status) (bool, error) {
	if statusAnterior == nil {
		return false, domain.ErroValidacao("status anterior da solicitação não informado")
	}

	linha := paraLinha(solicitacao)
	argumentos := map[string]any{
		"id":                linha.ID,
		"aprovador_id":      linha.AprovadorID,
		"status":            linha.Status,
		"rejeitacao_motivo": linha.RejeitacaoMotivo,
		"aprovado_em":       linha.AprovadoEm,
		"rejeitado_em":      linha.RejeitadoEm,
		"cancelado_em":      linha.CanceladoEm,
		"atualizado_em":     linha.AtualizadoEm,
		"status_anterior":   string(*statusAnterior),
	}

	query := `
		UPDATE solicitacao
		SET aprovador_id = :aprovador_id,
		    status = :status,
		    rejeitacao_motivo = :rejeitacao_motivo,
		    aprovado_em = :aprovado_em,
		    rejeitado_em = :rejeitado_em,
		    cancelado_em = :cancelado_em,
		    atualizado_em = :atualizado_em,
		    versao = versao + 1
		WHERE id = :id AND status = :status_anterior
	`

	resultado, err := tx.NamedExecContext(ctx, query, argumentos)
	if err != nil {
		return false, err
	}

	linhas, err := resultado.RowsAffected()
	if err != nil {
		return false, err
	}

	return linhas > 0, nil
}

func exigirEmpresaAtivaParaSolicitacao(ctx context.Context, tx *sqlx.Tx, empresaID uuid.UUID) error {
	var ativa bool
	err := tx.GetContext(ctx, &ativa, `SELECT ativa FROM empresa WHERE id = $1 FOR SHARE`, empresaID)
	if errors.Is(err, sql.ErrNoRows) {
		return domainsolicitacao.ErroEmpresaNaoEncontrada()
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if !ativa {
		return domainsolicitacao.ErroEmpresaInativa()
	}

	return nil
}

type categoriaParaLancamento struct {
	Ativa        bool   `db:"ativa"`
	Subcategoria bool   `db:"subcategoria"`
	Natureza     string `db:"natureza"`
}

func exigirClassificacaoGravavel(ctx context.Context, tx *sqlx.Tx, solicitacao *domainsolicitacao.Solicitacao) error {
	if err := exigirFornecedorAtivo(ctx, tx, solicitacao.FornecedorID); err != nil {
		return err
	}

	return exigirCategoriaDeDespesaLancavel(ctx, tx, solicitacao.CategoriaFinanceiraID)
}

func exigirFornecedorAtivo(ctx context.Context, tx *sqlx.Tx, fornecedorID *uuid.UUID) error {
	if fornecedorID == nil {
		return nil
	}

	var ativo bool
	err := tx.GetContext(ctx, &ativo, `SELECT ativo FROM fornecedor WHERE id = $1 FOR SHARE`, *fornecedorID)
	if errors.Is(err, sql.ErrNoRows) {
		return domainsolicitacao.ErroFornecedorNaoEncontrado()
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if !ativo {
		return domainsolicitacao.ErroFornecedorInativo()
	}

	return nil
}

func exigirCategoriaDeDespesaLancavel(ctx context.Context, tx *sqlx.Tx, categoriaID *uuid.UUID) error {
	if categoriaID == nil {
		return nil
	}

	var categoria categoriaParaLancamento
	err := tx.GetContext(ctx, &categoria, `
		SELECT ativa, categoria_pai_id IS NOT NULL AS subcategoria, natureza
		FROM categoria_financeira
		WHERE id = $1
		FOR SHARE
	`, *categoriaID)
	if errors.Is(err, sql.ErrNoRows) {
		return domainsolicitacao.ErroCategoriaFinanceiraNaoEncontrada()
	}
	if err != nil {
		return tratarErroDeGravacao(err)
	}
	if !categoria.Ativa || !categoria.Subcategoria {
		return domainsolicitacao.ErroCategoriaFinanceiraNaoLancavel()
	}
	if categoria.Natureza != domainfinanceiro.NaturezaDespesa.String() {
		return domainsolicitacao.ErroCategoriaFinanceiraNaoEhDespesa()
	}

	return nil
}

func condicao(coluna string, posicao int) string {
	return coluna + " = $" + strconv.Itoa(posicao)
}

func unirCondicoes(condicoes []string) string {
	resultado := ""
	for i, condicao := range condicoes {
		if i > 0 {
			resultado += " AND "
		}
		resultado += condicao
	}

	return resultado
}

func limiteOffset(argumentos []any) string {
	return " LIMIT $" + strconv.Itoa(len(argumentos)-1) + " OFFSET $" + strconv.Itoa(len(argumentos))
}
