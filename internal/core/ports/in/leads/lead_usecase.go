package leads

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	"github.com/google/uuid"
)

type LeadUseCase interface {
	Create(ctx context.Context, lead *lead.Lead) (uuid.UUID, error)
	Update(ctx context.Context, lead *lead.Lead) error
	GetByID(ctx context.Context, id uuid.UUID) (*lead.Lead, error)
	ListByFunnil(ctx context.Context, funilID uuid.UUID) ([]*lead.Lead, error)
	ListByEtapa(ctx context.Context, etapaID uuid.UUID) ([]*lead.Lead, error)
	ListAtivos(ctx context.Context, paginacao domain.PaginacaoFiltro) ([]*lead.Lead, error)
	Search(ctx context.Context, query string, paginacao domain.PaginacaoFiltro) ([]*lead.Lead, error)
	Delete(ctx context.Context, id uuid.UUID) error
	CountByEtapa(ctx context.Context, etapaID uuid.UUID) (int, error)
	CountByFunil(ctx context.Context, funilID uuid.UUID) (int, error)
	UpdateEtapa(ctx context.Context, leadID uuid.UUID, newEtapaID uuid.UUID) error
}