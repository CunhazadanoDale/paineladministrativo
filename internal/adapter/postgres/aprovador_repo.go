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

var _ portsout.AprovadorRepository = (*AprovadorRepository)(nil)

type AprovadorRepository struct {
	db *sqlx.DB
}

func NewAprovadorRepository(db *sqlx.DB) *AprovadorRepository {
	return &AprovadorRepository{db: db}
}

func (a *AprovadorRepository) Criar(ctx context.Context, aprovador *domainsolicitacao.Aprovador) (uuid.UUID, error) {
	query := `
		INSERT INTO aprovador (id, usuario_id, criado_em)
		VALUES (:id, :usuario_id, :criado_em)
		RETURNING id
	`

	rows, err := a.db.NamedQueryContext(ctx, query, aprovador)
	if err != nil {
		return uuid.Nil, tratarErroDeGravacao(err)
	}
	defer rows.Close()

	if !rows.Next() {
		return uuid.Nil, errors.New("nenhum registro retornado na designação do aprovador")
	}

	var id uuid.UUID
	if err := rows.Scan(&id); err != nil {
		return uuid.Nil, tratarErroDeGravacao(err)
	}

	return id, nil
}

func (a *AprovadorRepository) Obter(ctx context.Context, id uuid.UUID) (*domainsolicitacao.Aprovador, error) {
	query := `
		SELECT id, usuario_id, criado_em
		FROM aprovador
		WHERE id = $1
	`

	var item domainsolicitacao.Aprovador
	if err := a.db.GetContext(ctx, &item, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (a *AprovadorRepository) ObterPorUsuarioID(ctx context.Context, usuarioID uuid.UUID) (*domainsolicitacao.Aprovador, error) {
	query := `
		SELECT id, usuario_id, criado_em
		FROM aprovador
		WHERE usuario_id = $1
	`

	var item domainsolicitacao.Aprovador
	if err := a.db.GetContext(ctx, &item, query, usuarioID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (a *AprovadorRepository) Listar(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*domainsolicitacao.Aprovador, error) {
	offset := (filtro.Page - 1) * filtro.Size
	query := `
		SELECT id, usuario_id, criado_em
		FROM aprovador
		ORDER BY criado_em ASC
		LIMIT $1 OFFSET $2
	`

	var itens []*domainsolicitacao.Aprovador
	if err := a.db.SelectContext(ctx, &itens, query, filtro.Size, offset); err != nil {
		return nil, err
	}
	return itens, nil
}

func (a *AprovadorRepository) Remover(ctx context.Context, id uuid.UUID) error {
	_, err := a.db.ExecContext(ctx, `DELETE FROM aprovador WHERE id = $1`, id)
	return tratarErro(err)
}
