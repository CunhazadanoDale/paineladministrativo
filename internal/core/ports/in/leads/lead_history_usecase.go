package leads

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	"github.com/google/uuid"
)

type LeadHistoryUseCase interface {
	ListByLead(ctx context.Context, leadID uuid.UUID, paginacao domain.PaginacaoFiltro) ([]lead.LeadHistorico, error)
	RegistrarMovimentacao(ctx context.Context, leadID uuid.UUID, etapaAnteriorID uuid.UUID, etapaAtualID uuid.UUID) error
}