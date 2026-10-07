package usuarios

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	"github.com/google/uuid"
)

type UsuarioUseCase interface {
	Create(ctx context.Context, nome string, email string, senha string, cargoID uuid.UUID) (uuid.UUID, error)
	Update(ctx context.Context, usuario *usuarios.Usuario) error
	GetByID(ctx context.Context, id uuid.UUID) (*usuarios.Usuario, error)
	GetByEmail(ctx context.Context, email string) (*usuarios.Usuario, error)
	List(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*usuarios.Usuario, error)
	ListAtivos(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*usuarios.Usuario, error)
	Search(ctx context.Context, termo string, filtro domain.PaginacaoFiltro) ([]*usuarios.Usuario, error)
	Authenticate(ctx context.Context, email string, senha string) (*usuarios.Usuario, error)
	UpdateSenha(ctx context.Context, id uuid.UUID, novaSenha string) error
	UpdateUltimoLogin(ctx context.Context, id uuid.UUID, ultimoLogin time.Time) error
	Ativar(ctx context.Context, id uuid.UUID) error
	Desativar(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}
