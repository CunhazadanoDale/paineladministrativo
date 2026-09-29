package leads

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	"github.com/google/uuid"
)

type LeadHistoryRepository interface {
	RegistrarMovimentacao(ctx context.Context, leadID uuid.UUID, etapaAnteriorID uuid.UUID, etapaAtualID uuid.UUID) error
	ListByLead(ctx context.Context, leadID uuid.UUID, paginacao domain.PaginacaoFiltro) ([]lead.LeadHistorico, error)
}
