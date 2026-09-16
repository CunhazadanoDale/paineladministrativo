package leads

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	"github.com/google/uuid"
)

type EtapaUseCase interface {
	Create(ctx context.Context, etapa *lead.Etapa) (uuid.UUID, error)
	Update(ctx context.Context, etapa *lead.Etapa) error
	GetByID(ctx context.Context, id uuid.UUID) (*lead.Etapa, error)
	ListByFunilID(ctx context.Context, funilID uuid.UUID) ([]*lead.Etapa, error)
	ListByFunilOrdenado(ctx context.Context, funilID uuid.UUID) ([]*lead.Etapa, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Reordenar(ctx context.Context, funilID uuid.UUID, etapas []*lead.Etapa) error
	GetNextEtapa(ctx context.Context, currentEtapaID uuid.UUID) (*lead.Etapa, error)
	GetPreviousEtapa(ctx context.Context, currentEtapaID uuid.UUID) (*lead.Etapa, error)
	ExistsByFunil(ctx context.Context, funilID uuid.UUID) (bool, error)
}
