package leads

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	"github.com/google/uuid"
)

type FunilUseCase interface {
	Create(ctx context.Context, funil *lead.Funil) (uuid.UUID, error)
	Update(ctx context.Context, funil *lead.Funil) error
	GetByID(ctx context.Context, funilID uuid.UUID) (*lead.Funil, error)
	List(ctx context.Context, filtro domain.PaginacaoFiltro) ([]lead.Funil, error)
	ListAtivos(ctx context.Context, filtro domain.PaginacaoFiltro) ([]lead.Funil, error)
	Delete(ctx context.Context, funilID uuid.UUID) error
	ExistsByID(ctx context.Context, funilID uuid.UUID) (bool, error)
}
