package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/solicitacao"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var _ portsout.ArquivoRepository = (*ArquivoRepository)(nil)

type ArquivoRepository struct {
	db *sqlx.DB
}

func NewArquivoRepository(db *sqlx.DB) *ArquivoRepository {
	return &ArquivoRepository{db: db}
}

func (a *ArquivoRepository) Create(ctx context.Context, arquivo *domainsolicitacao.Arquivo) (uuid.UUID, error) {
	query := `
		INSERT INTO arquivo (id, proprietario_id, nome, chave, content_type, tamanho, criado_em)
		VALUES (:id, :proprietario_id, :nome, :chave, :content_type, :tamanho, :criado_em)
		RETURNING id
	`

	rows, err := a.db.NamedQueryContext(ctx, query, arquivo)
	if err != nil {
		return uuid.Nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return uuid.Nil, errors.New("nenhum registro retornado na criação do arquivo")
	}

	var id uuid.UUID
	if err := rows.Scan(&id); err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

func (a *ArquivoRepository) GetByID(ctx context.Context, id uuid.UUID) (*domainsolicitacao.Arquivo, error) {
	query := `
		SELECT id, proprietario_id, nome, chave, content_type, tamanho, criado_em
		FROM arquivo
		WHERE id = $1
	`

	var item domainsolicitacao.Arquivo
	if err := a.db.GetContext(ctx, &item, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (a *ArquivoRepository) ListByProprietario(ctx context.Context, proprietarioID uuid.UUID, filtro domain.PaginacaoFiltro) ([]*domainsolicitacao.Arquivo, error) {
	offset := (filtro.Page - 1) * filtro.Size
	query := `
		SELECT id, proprietario_id, nome, chave, content_type, tamanho, criado_em
		FROM arquivo
		WHERE proprietario_id = $1
		ORDER BY criado_em DESC
		LIMIT $2 OFFSET $3
	`

	var itens []*domainsolicitacao.Arquivo
	if err := a.db.SelectContext(ctx, &itens, query, proprietarioID, filtro.Size, offset); err != nil {
		return nil, err
	}
	return itens, nil
}

func (a *ArquivoRepository) VinculadoASolicitacao(ctx context.Context, arquivoID uuid.UUID) (bool, error) {
	var existe bool
	err := a.db.GetContext(ctx, &existe, `
		SELECT EXISTS (
			SELECT 1 FROM solicitacao_arquivo WHERE arquivo_id = $1
		)
	`, arquivoID)

	return existe, err
}

func (a *ArquivoRepository) SolicitacaoDoArquivo(ctx context.Context, arquivoID uuid.UUID) (*uuid.UUID, error) {
	var id uuid.UUID
	err := a.db.GetContext(ctx, &id, `
		SELECT solicitacao_id
		FROM solicitacao_arquivo
		WHERE arquivo_id = $1
	`, arquivoID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &id, nil
}

func (a *ArquivoRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := a.db.ExecContext(ctx, `DELETE FROM arquivo WHERE id = $1`, id)
	return tratarErro(err)
}
