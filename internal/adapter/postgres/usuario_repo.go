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
	return tratarErroDeGravacao(err)
}

func (u *UsuarioRepository) Criar(ctx context.Context, usuario *domainusuarios.Usuario) (uuid.UUID, error) {
	query := `
		INSERT INTO usuario (id, nome, email, senha, cargo_id, ativo, ultimo_login, criado_em, atualizado_em)
		VALUES (:id, :nome, :email, :senha, :cargo_id, :ativo, :ultimo_login, :criado_em, :atualizado_em)
		RETURNING id
	`

	rows, err := u.db.NamedQueryContext(ctx, query, usuario)
	if err != nil {
		return uuid.Nil, tratarErroDeGravacao(err)
	}
	defer rows.Close()

	if !rows.Next() {
		return uuid.Nil, errors.New("nenhum registro retornado na criação do usuário")
	}

	var id uuid.UUID
	if err := rows.Scan(&id); err != nil {
		return uuid.Nil, tratarErroDeGravacao(err)
	}

	return id, nil
}

func (u *UsuarioRepository) Remover(ctx context.Context, id uuid.UUID) error {
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
	return tratarErroDeGravacao(err)
}

func (u *UsuarioRepository) ObterPorEmail(ctx context.Context, email string) (*domainusuarios.Usuario, error) {
	query := `
		SELECT id, nome, email, senha, cargo_id, ativo, versao_sessao, ultimo_login, criado_em, atualizado_em
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

func (u *UsuarioRepository) Obter(ctx context.Context, id uuid.UUID) (*domainusuarios.Usuario, error) {
	query := `
		SELECT id, nome, email, senha, cargo_id, ativo, versao_sessao, ultimo_login, criado_em, atualizado_em
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

func (u *UsuarioRepository) Listar(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*domainusuarios.Usuario, error) {
	offset := (filtro.Page - 1) * filtro.Size
	query := `
		SELECT id, nome, email, senha, cargo_id, ativo, versao_sessao, ultimo_login, criado_em, atualizado_em
		FROM usuario
		ORDER BY nome ASC, id ASC
		LIMIT $1 OFFSET $2
	`

	var itens []*domainusuarios.Usuario
	if err := u.db.SelectContext(ctx, &itens, query, filtro.Size, offset); err != nil {
		return nil, err
	}
	return itens, nil
}

func (u *UsuarioRepository) ListarAtivos(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*domainusuarios.Usuario, error) {
	offset := (filtro.Page - 1) * filtro.Size
	query := `
		SELECT id, nome, email, senha, cargo_id, ativo, versao_sessao, ultimo_login, criado_em, atualizado_em
		FROM usuario
		WHERE ativo = TRUE
		ORDER BY nome ASC, id ASC
		LIMIT $1 OFFSET $2
	`

	var itens []*domainusuarios.Usuario
	if err := u.db.SelectContext(ctx, &itens, query, filtro.Size, offset); err != nil {
		return nil, err
	}
	return itens, nil
}

func (u *UsuarioRepository) Buscar(ctx context.Context, termo string, filtro domain.PaginacaoFiltro) ([]*domainusuarios.Usuario, error) {
	offset := (filtro.Page - 1) * filtro.Size
	query := `
		SELECT id, nome, email, senha, cargo_id, ativo, versao_sessao, ultimo_login, criado_em, atualizado_em
		FROM usuario
		WHERE LOWER(nome) LIKE '%' || LOWER($1) || '%'
		   OR LOWER(email) LIKE '%' || LOWER($1) || '%'
		ORDER BY nome ASC, id ASC
		LIMIT $2 OFFSET $3
	`

	var itens []*domainusuarios.Usuario
	if err := u.db.SelectContext(ctx, &itens, query, termo, filtro.Size, offset); err != nil {
		return nil, err
	}
	return itens, nil
}

func (u *UsuarioRepository) Atualizar(ctx context.Context, usuario *domainusuarios.Usuario) error {
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
	return tratarErroDeGravacao(err)
}

func (u *UsuarioRepository) AtualizarUltimoLogin(ctx context.Context, id uuid.UUID, ultimoLogin time.Time) error {
	query := `
		UPDATE usuario
		SET ultimo_login = $1
		WHERE id = $2
	`

	_, err := u.db.ExecContext(ctx, query, ultimoLogin, id)
	return tratarErroDeGravacao(err)
}

func (u *UsuarioRepository) AtualizarSenha(ctx context.Context, id uuid.UUID, senha string, atualizadoEm time.Time) error {
	query := `
		UPDATE usuario
		SET senha = $1,
		    versao_sessao = versao_sessao + 1,
		    atualizado_em = $2
		WHERE id = $3
	`

	_, err := u.db.ExecContext(ctx, query, senha, atualizadoEm, id)
	return tratarErroDeGravacao(err)
}

func (u *UsuarioRepository) EncerrarSessoes(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE usuario
		SET versao_sessao = versao_sessao + 1
		WHERE id = $1
	`

	_, err := u.db.ExecContext(ctx, query, id)
	return tratarErroDeGravacao(err)
}

func (u *UsuarioRepository) ContarAdministradoresAtivos(ctx context.Context) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM usuario u
		JOIN cargo c ON c.id = u.cargo_id
		WHERE u.ativo = TRUE
		  AND c.administrador = TRUE
	`

	var total int
	if err := u.db.GetContext(ctx, &total, query); err != nil {
		return 0, err
	}
	return total, nil
}

func (u *UsuarioRepository) ContarAtivosPorCargo(ctx context.Context, cargoID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM usuario WHERE cargo_id = $1 AND ativo = TRUE`

	var total int
	if err := u.db.GetContext(ctx, &total, query, cargoID); err != nil {
		return 0, err
	}
	return total, nil
}
