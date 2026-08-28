package leads

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
)

type LeadHistoryRepository interface {
	RegistrarMovimentacao(ctx context.Context, leadID string, 
		etapaAnteriorID string, etapaAtualID string) error
	ListByLead(ctx context.Context, leadID string, paginacao domain.PaginacaoFiltro) ([]lead.LeadHistorico, error)
}