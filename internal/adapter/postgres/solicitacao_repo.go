package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/solicitacao"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var _ portsout.SolicitacaoRepository = (*SolicitacaoRepository)(nil)

type SolicitacaoRepository struct {
	db *sqlx.DB
}

func NewSolicitacaoRepository(db *sqlx.DB) *SolicitacaoRepository {
	return &SolicitacaoRepository{db: db}
}

type solicitacaoLinha struct {
	ID               uuid.UUID  `db:"id"`
	SolicitanteID    uuid.UUID  `db:"solicitante_id"`
	AprovadorID      *uuid.UUID `db:"aprovador_id"`
	ValorCentavos    int64      `db:"valor_estimado"`
	PrazoPagamento   time.Time  `db:"prazo_pagamento"`
	Observacao       string     `db:"observacao"`
	FormaPagamento   string     `db:"forma_pagamento"`
	Status           string     `db:"status"`
	RejeitacaoMotivo *string    `db:"rejeitacao_motivo"`
	AprovadoEm       *time.Time `db:"aprovado_em"`
	RejeitadoEm      *time.Time `db:"rejeitado_em"`
	CanceladoEm      *time.Time `db:"cancelado_em"`
	CriadoEm         time.Time  `db:"criado_em"`
	AtualizadoEm     time.Time  `db:"atualizado_em"`
}

type pagamentoLinha struct {
	ID                   uuid.UUID  `db:"id"`
	SolicitacaoID        uuid.UUID  `db:"solicitacao_id"`
	ComprovanteArquivoID *uuid.UUID `db:"comprovante_arquivo_id"`
	ValorCentavos        int64      `db:"valor"`
	PagoEm               time.Time  `db:"pago_em"`
	CriadoEm             time.Time  `db:"criado_em"`
}

const colunasSolicitacao = `
	id, solicitante_id, aprovador_id, valor_estimado, prazo_pagamento, observacao,
	forma_pagamento, status, rejeitacao_motivo, aprovado_em, rejeitado_em,
	cancelado_em, criado_em, atualizado_em
`

func (s *SolicitacaoRepository) Criar(ctx context.Context, solicitacao *domainsolicitacao.Solicitacao, arquivoIDs []uuid.UUID, historico *domainsolicitacao.Historico) (uuid.UUID, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return uuid.Nil, err
	}
	defer func() { _ = tx.Rollback() }()

	linha := paraLinha(solicitacao)
	query := `
		INSERT INTO solicitacao (
			id, solicitante_id, aprovador_id, valor_estimado, prazo_pagamento, observacao,
			forma_pagamento, status, rejeitacao_motivo, aprovado_em, rejeitado_em,
			cancelado_em, criado_em, atualizado_em
		) VALUES (
			:id, :solicitante_id, :aprovador_id, :valor_estimado, :prazo_pagamento, :observacao,
			:forma_pagamento, :status, :rejeitacao_motivo, :aprovado_em, :rejeitado_em,
			:cancelado_em, :criado_em, :atualizado_em
		)
	`

	if _, err := tx.NamedExecContext(ctx, query, linha); err != nil {
		return uuid.Nil, err
	}

	if err := inserirHistorico(ctx, tx, historico); err != nil {
		return uuid.Nil, err
	}

	for _, arquivoID := range arquivoIDs {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO solicitacao_arquivo (solicitacao_id, arquivo_id) VALUES ($1, $2)
		`, solicitacao.ID, arquivoID); err != nil {
			return uuid.Nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return uuid.Nil, err
	}

	return solicitacao.ID, nil
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

	query := `SELECT ` + colunasSolicitacao + ` FROM solicitacao`
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
	if len(condicoes) > 0 {
		query += " WHERE " + unirCondicoes(condicoes)
	}

	argumentos = append(argumentos, filtro.Size, offset)
	query += ` ORDER BY criado_em DESC` + limiteOffset(argumentos)

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

func (s *SolicitacaoRepository) AtualizarStatus(ctx context.Context, solicitacao *domainsolicitacao.Solicitacao, historico *domainsolicitacao.Historico) (bool, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	atualizado, err := atualizarStatusComGuarda(ctx, tx, solicitacao, historico.DeStatus)
	if err != nil {
		return false, err
	}
	if !atualizado {
		return false, nil
	}

	if err := inserirHistorico(ctx, tx, historico); err != nil {
		return false, err
	}

	if err := tx.Commit(); err != nil {
		return false, err
	}

	return true, nil
}

func (s *SolicitacaoRepository) CriarPagamento(ctx context.Context, solicitacao *domainsolicitacao.Solicitacao, pagamento *domainsolicitacao.Pagamento, historico *domainsolicitacao.Historico) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	atualizado, err := atualizarStatusComGuarda(ctx, tx, solicitacao, historico.DeStatus)
	if err != nil {
		return err
	}
	if !atualizado {
		return domain.ErroConflito("solicitação alterada por outra operação, recarregue e tente novamente")
	}

	linhaPagamento := pagamentoLinha{
		ID:                   pagamento.ID,
		SolicitacaoID:        pagamento.SolicitacaoID,
		ComprovanteArquivoID: pagamento.ComprovanteArquivoID,
		ValorCentavos:        pagamento.Valor.Centavos(),
		PagoEm:               pagamento.PagoEm,
		CriadoEm:             pagamento.CriadoEm,
	}

	if _, err := tx.NamedExecContext(ctx, `
		INSERT INTO pagamento (id, solicitacao_id, comprovante_arquivo_id, valor, pago_em, criado_em)
		VALUES (:id, :solicitacao_id, :comprovante_arquivo_id, :valor, :pago_em, :criado_em)
	`, linhaPagamento); err != nil {
		return err
	}

	if pagamento.ComprovanteArquivoID != nil {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO solicitacao_arquivo (solicitacao_id, arquivo_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, pagamento.SolicitacaoID, *pagamento.ComprovanteArquivoID); err != nil {
			return err
		}
	}

	if err := inserirHistorico(ctx, tx, historico); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *SolicitacaoRepository) ListarArquivos(ctx context.Context, solicitacaoID uuid.UUID) ([]*domainsolicitacao.Arquivo, error) {
	query := `
		SELECT a.id, a.proprietario_id, a.nome, a.chave, a.content_type, a.tamanho, a.criado_em
		FROM arquivo a
		JOIN solicitacao_arquivo sa ON sa.arquivo_id = a.id
		WHERE sa.solicitacao_id = $1
		ORDER BY a.criado_em ASC
	`

	var itens []*domainsolicitacao.Arquivo
	if err := s.db.SelectContext(ctx, &itens, query, solicitacaoID); err != nil {
		return nil, err
	}
	return itens, nil
}

func (s *SolicitacaoRepository) ListarHistorico(ctx context.Context, solicitacaoID uuid.UUID) ([]*domainsolicitacao.Historico, error) {
	query := `
		SELECT id, solicitacao_id, usuario_id, de_status, para_status, descricao, criado_em
		FROM solicitacao_historico
		WHERE solicitacao_id = $1
		ORDER BY criado_em DESC
	`

	var itens []*domainsolicitacao.Historico
	if err := s.db.SelectContext(ctx, &itens, query, solicitacaoID); err != nil {
		return nil, err
	}
	return itens, nil
}

func (s *SolicitacaoRepository) ObterPagamento(ctx context.Context, solicitacaoID uuid.UUID) (*domainsolicitacao.Pagamento, error) {
	query := `
		SELECT id, solicitacao_id, comprovante_arquivo_id, valor, pago_em, criado_em
		FROM pagamento
		WHERE solicitacao_id = $1
	`

	var linha pagamentoLinha
	if err := s.db.GetContext(ctx, &linha, query, solicitacaoID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &domainsolicitacao.Pagamento{
		ID:                   linha.ID,
		SolicitacaoID:        linha.SolicitacaoID,
		ComprovanteArquivoID: linha.ComprovanteArquivoID,
		Valor:                domainsolicitacao.ValorDe(linha.ValorCentavos),
		PagoEm:               linha.PagoEm,
		CriadoEm:             linha.CriadoEm,
	}, nil
}

func paraLinha(solicitacao *domainsolicitacao.Solicitacao) solicitacaoLinha {
	return solicitacaoLinha{
		ID:               solicitacao.ID,
		SolicitanteID:    solicitacao.SolicitanteID,
		AprovadorID:      solicitacao.AprovadorID,
		ValorCentavos:    solicitacao.Valor.Centavos(),
		PrazoPagamento:   solicitacao.Prazo.Data(),
		Observacao:       solicitacao.Observacao.Texto(),
		FormaPagamento:   string(solicitacao.FormaPagamento),
		Status:           string(solicitacao.Status),
		RejeitacaoMotivo: motivoRejeicao(solicitacao),
		AprovadoEm:       solicitacao.AprovadoEm,
		RejeitadoEm:      solicitacao.RejeitadoEm,
		CanceladoEm:      solicitacao.CanceladoEm,
		CriadoEm:         solicitacao.CriadoEm,
		AtualizadoEm:     solicitacao.AtualizadoEm,
	}
}

func paraDominio(linha *solicitacaoLinha) *domainsolicitacao.Solicitacao {
	solicitacao := &domainsolicitacao.Solicitacao{
		ID:             linha.ID,
		SolicitanteID:  linha.SolicitanteID,
		AprovadorID:    linha.AprovadorID,
		Valor:          domainsolicitacao.ValorDe(linha.ValorCentavos),
		Prazo:          domainsolicitacao.PrazoPagamentoDe(linha.PrazoPagamento),
		Observacao:     domainsolicitacao.ObservacaoDe(linha.Observacao),
		FormaPagamento: domainsolicitacao.FormaPagamento(linha.FormaPagamento),
		Status:         domainsolicitacao.StatusDe(linha.Status),
		AprovadoEm:     linha.AprovadoEm,
		RejeitadoEm:    linha.RejeitadoEm,
		CanceladoEm:    linha.CanceladoEm,
		CriadoEm:       linha.CriadoEm,
		AtualizadoEm:   linha.AtualizadoEm,
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
		INSERT INTO solicitacao_historico (id, solicitacao_id, usuario_id, de_status, para_status, descricao, criado_em)
		VALUES (:id, :solicitacao_id, :usuario_id, :de_status, :para_status, :descricao, :criado_em)
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
		    atualizado_em = :atualizado_em
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
