package usuarios

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	"github.com/google/uuid"
)

type CargoUseCase interface {
	Create(ctx context.Context, nome string, descricao string, administrador bool, financeiro bool) (uuid.UUID, error)
	Update(ctx context.Context, cargo *usuarios.Cargo) error
	GetByID(ctx context.Context, id uuid.UUID) (*usuarios.Cargo, error)
	GetByNome(ctx context.Context, nome string) (*usuarios.Cargo, error)
	List(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*usuarios.Cargo, error)
	ListAtivos(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*usuarios.Cargo, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
