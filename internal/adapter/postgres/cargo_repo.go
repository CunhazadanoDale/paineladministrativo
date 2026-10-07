package postgres

import (
	"context"
	"database/sql"
	"errors"

	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/usuarios"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var _ portsout.CargoRepository = (*CargoRepository)(nil)

type CargoRepository struct {
	db *sqlx.DB
}

func NewCargoRepository(db *sqlx.DB) *CargoRepository {
	return &CargoRepository{db: db}
}

func (c *CargoRepository) Create(ctx context.Context, cargo *domainusuarios.Cargo) (uuid.UUID, error) {
	query := `
		INSERT INTO cargo (id, nome, descricao, ativo)
		VALUES (:id, :nome, :descricao, :ativo)
		RETURNING id
	`

	rows, err := c.db.NamedQueryContext(ctx, query, cargo)
	if err != nil {
		return uuid.Nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return uuid.Nil, errors.New("nenhum registro retornado na criação do cargo")
	}

	var id uuid.UUID
	if err := rows.Scan(&id); err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

func (c *CargoRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := c.db.ExecContext(ctx, `DELETE FROM cargo WHERE id = $1`, id)
	return tratarErro(err)
}

func (c *CargoRepository) GetByID(ctx context.Context, id uuid.UUID) (*domainusuarios.Cargo, error) {
	query := `SELECT id, nome, descricao, ativo FROM cargo WHERE id = $1`

	var item domainusuarios.Cargo
	if err := c.db.GetContext(ctx, &item, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (c *CargoRepository) GetByNome(ctx context.Context, nome string) (*domainusuarios.Cargo, error) {
	query := `SELECT id, nome, descricao, ativo FROM cargo WHERE nome = $1`

	var item domainusuarios.Cargo
	if err := c.db.GetContext(ctx, &item, query, nome); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (c *CargoRepository) List(ctx context.Context) ([]*domainusuarios.Cargo, error) {
	query := `
		SELECT id, nome, descricao, ativo
		FROM cargo
		ORDER BY nome ASC
	`

	var itens []*domainusuarios.Cargo
	if err := c.db.SelectContext(ctx, &itens, query); err != nil {
		return nil, err
	}
	return itens, nil
}

func (c *CargoRepository) ListAtivos(ctx context.Context) ([]*domainusuarios.Cargo, error) {
	query := `
		SELECT id, nome, descricao, ativo
		FROM cargo
		WHERE ativo = TRUE
		ORDER BY nome ASC
	`

	var itens []*domainusuarios.Cargo
	if err := c.db.SelectContext(ctx, &itens, query); err != nil {
		return nil, err
	}
	return itens, nil
}

func (c *CargoRepository) Update(ctx context.Context, cargo *domainusuarios.Cargo) error {
	query := `
		UPDATE cargo
		SET nome = :nome,
		    descricao = :descricao,
		    ativo = :ativo
		WHERE id = :id
	`

	_, err := c.db.NamedExecContext(ctx, query, cargo)
	return err
}
