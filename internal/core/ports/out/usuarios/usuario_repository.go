package usuarios

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	"github.com/google/uuid"
)

type UsuarioRepository interface {
	Create(ctx context.Context, usuario *usuarios.Usuario) (uuid.UUID, error)
	Update(ctx context.Context, usuario *usuarios.Usuario) error
	GetByID(ctx context.Context, id uuid.UUID) (*usuarios.Usuario, error)
	GetByEmail(ctx context.Context, email string) (*usuarios.Usuario, error)
	List(ctx context.Context) ([]*usuarios.Usuario, error)
	ListAtivos(ctx context.Context) ([]*usuarios.Usuario, error)
	Search(ctx context.Context, termo string) ([]*usuarios.Usuario, error)
	UpdateUltimoLogin(ctx context.Context, id uuid.UUID, ultimoLogin time.Time) error
	Ativar(ctx context.Context, id uuid.UUID) error
	Desativar(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}
