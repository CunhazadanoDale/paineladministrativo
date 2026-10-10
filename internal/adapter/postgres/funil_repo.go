package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/leads"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var _ leads.FunilRepository = (*FunilRepository)(nil)

type FunilRepository struct {
	db *sqlx.DB
}

func NewFunilRepository(db *sqlx.DB) *FunilRepository {
	return &FunilRepository{db: db}
}

func (f *FunilRepository) Criar(ctx context.Context, funil *lead.Funil) (uuid.UUID, error) {
	query := `
		INSERT INTO funil (funil_id, nome, ativo)
		VALUES (:funil_id, :nome, :ativo)
		RETURNING funil_id
	`

	rows, err := f.db.NamedQueryContext(ctx, query, funil)
	if err != nil {
		return uuid.Nil, tratarErroDeGravacao(err)
	}
	defer rows.Close()

	if !rows.Next() {
		return uuid.Nil, errors.New("nenhum registro retornado na criação do funil")
	}

	var id uuid.UUID
	if err := rows.Scan(&id); err != nil {
		return uuid.Nil, tratarErroDeGravacao(err)
	}

	return id, nil
}

func (f *FunilRepository) Remover(ctx context.Context, funilID uuid.UUID) error {
	_, err := f.db.ExecContext(ctx, `DELETE FROM funil WHERE funil_id = $1`, funilID)
	return tratarErro(err)
}

func (f *FunilRepository) Existe(ctx context.Context, funilID uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM funil WHERE funil_id = $1)`
	if err := f.db.GetContext(ctx, &exists, query, funilID); err != nil {
		return false, err
	}
	return exists, nil
}

func (f *FunilRepository) Obter(ctx context.Context, funilID uuid.UUID) (*lead.Funil, error) {
	query := `SELECT funil_id, nome, ativo FROM funil WHERE funil_id = $1`
	var funil lead.Funil
	if err := f.db.GetContext(ctx, &funil, query, funilID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &funil, nil
}

func (f *FunilRepository) Listar(ctx context.Context, filtro domain.PaginacaoFiltro) ([]lead.Funil, error) {
	offset := (filtro.Page - 1) * filtro.Size
	query := `
		SELECT funil_id, nome, ativo
		FROM funil
		ORDER BY nome ASC, funil_id ASC
		LIMIT $1 OFFSET $2
	`

	var funis []lead.Funil
	if err := f.db.SelectContext(ctx, &funis, query, filtro.Size, offset); err != nil {
		return nil, err
	}
	return funis, nil
}

func (f *FunilRepository) ListarAtivos(ctx context.Context, filtro domain.PaginacaoFiltro) ([]lead.Funil, error) {
	offset := (filtro.Page - 1) * filtro.Size
	query := `
		SELECT funil_id, nome, ativo
		FROM funil
		WHERE ativo = TRUE
		ORDER BY nome ASC, funil_id ASC
		LIMIT $1 OFFSET $2
	`

	var funis []lead.Funil
	if err := f.db.SelectContext(ctx, &funis, query, filtro.Size, offset); err != nil {
		return nil, err
	}
	return funis, nil
}

func (f *FunilRepository) Atualizar(ctx context.Context, funil *lead.Funil) error {
	query := `
		UPDATE funil
		SET nome = :nome, ativo = :ativo
		WHERE funil_id = :funil_id
	`

	_, err := f.db.NamedExecContext(ctx, query, funil)
	return tratarErroDeGravacao(err)
}
