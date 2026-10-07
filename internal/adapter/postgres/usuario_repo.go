package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/usuarios"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var _ portsout.UsuarioRepository = (*UsuarioRepository)(nil)

type UsuarioRepository struct {
	db *sqlx.DB
}

func NewUsuarioRepository(db *sqlx.DB) *UsuarioRepository {
	return &UsuarioRepository{db: db}
}

func (u *UsuarioRepository) Ativar(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE usuario
		SET ativo = TRUE,
		    atualizado_em = NOW()
		WHERE id = $1
	`

	_, err := u.db.ExecContext(ctx, query, id)
	return err
}

func (u *UsuarioRepository) Create(ctx context.Context, usuario *domainusuarios.Usuario) (uuid.UUID, error) {
	query := `
		INSERT INTO usuario (id, nome, email, senha, cargo_id, ativo, ultimo_login, criado_em, atualizado_em)
		VALUES (:id, :nome, :email, :senha, :cargo_id, :ativo, :ultimo_login, :criado_em, :atualizado_em)
		RETURNING id
	`

	rows, err := u.db.NamedQueryContext(ctx, query, usuario)
	if err != nil {
		return uuid.Nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return uuid.Nil, errors.New("nenhum registro retornado na criação do usuário")
	}

	var id uuid.UUID
	if err := rows.Scan(&id); err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

func (u *UsuarioRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := u.db.ExecContext(ctx, `DELETE FROM usuario WHERE id = $1`, id)
	return tratarErro(err)
}

func (u *UsuarioRepository) Desativar(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE usuario
		SET ativo = FALSE,
		    atualizado_em = NOW()
		WHERE id = $1
	`

	_, err := u.db.ExecContext(ctx, query, id)
	return err
}

func (u *UsuarioRepository) GetByEmail(ctx context.Context, email string) (*domainusuarios.Usuario, error) {
	query := `
		SELECT id, nome, email, senha, cargo_id, ativo, ultimo_login, criado_em, atualizado_em
		FROM usuario
		WHERE email = $1
	`

	var item domainusuarios.Usuario
	if err := u.db.GetContext(ctx, &item, query, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (u *UsuarioRepository) GetByID(ctx context.Context, id uuid.UUID) (*domainusuarios.Usuario, error) {
	query := `
		SELECT id, nome, email, senha, cargo_id, ativo, ultimo_login, criado_em, atualizado_em
		FROM usuario
		WHERE id = $1
	`

	var item domainusuarios.Usuario
	if err := u.db.GetContext(ctx, &item, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (u *UsuarioRepository) List(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*domainusuarios.Usuario, error) {
	offset := (filtro.Page - 1) * filtro.Size
	query := `
		SELECT id, nome, email, senha, cargo_id, ativo, ultimo_login, criado_em, atualizado_em
		FROM usuario
		ORDER BY nome ASC
		LIMIT $1 OFFSET $2
	`

	var itens []*domainusuarios.Usuario
	if err := u.db.SelectContext(ctx, &itens, query, filtro.Size, offset); err != nil {
		return nil, err
	}
	return itens, nil
}

func (u *UsuarioRepository) ListAtivos(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*domainusuarios.Usuario, error) {
	offset := (filtro.Page - 1) * filtro.Size
	query := `
		SELECT id, nome, email, senha, cargo_id, ativo, ultimo_login, criado_em, atualizado_em
		FROM usuario
		WHERE ativo = TRUE
		ORDER BY nome ASC
		LIMIT $1 OFFSET $2
	`

	var itens []*domainusuarios.Usuario
	if err := u.db.SelectContext(ctx, &itens, query, filtro.Size, offset); err != nil {
		return nil, err
	}
	return itens, nil
}

func (u *UsuarioRepository) Search(ctx context.Context, termo string, filtro domain.PaginacaoFiltro) ([]*domainusuarios.Usuario, error) {
	offset := (filtro.Page - 1) * filtro.Size
	query := `
		SELECT id, nome, email, senha, cargo_id, ativo, ultimo_login, criado_em, atualizado_em
		FROM usuario
		WHERE LOWER(nome) LIKE '%' || LOWER($1) || '%'
		   OR LOWER(email) LIKE '%' || LOWER($1) || '%'
		ORDER BY nome ASC
		LIMIT $2 OFFSET $3
	`

	var itens []*domainusuarios.Usuario
	if err := u.db.SelectContext(ctx, &itens, query, termo, filtro.Size, offset); err != nil {
		return nil, err
	}
	return itens, nil
}

func (u *UsuarioRepository) Update(ctx context.Context, usuario *domainusuarios.Usuario) error {
	query := `
		UPDATE usuario
		SET nome = :nome,
		    email = :email,
		    senha = :senha,
		    cargo_id = :cargo_id,
		    ativo = :ativo,
		    ultimo_login = :ultimo_login,
		    atualizado_em = :atualizado_em
		WHERE id = :id
	`

	_, err := u.db.NamedExecContext(ctx, query, usuario)
	return err
}

func (u *UsuarioRepository) UpdateUltimoLogin(ctx context.Context, id uuid.UUID, ultimoLogin time.Time) error {
	query := `
		UPDATE usuario
		SET ultimo_login = $1
		WHERE id = $2
	`

	_, err := u.db.ExecContext(ctx, query, ultimoLogin, id)
	return err
}
