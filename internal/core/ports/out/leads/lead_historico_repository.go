package leads

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	"github.com/google/uuid"
)

type LeadHistoricoRepository interface {
	ListarPorLead(ctx context.Context, leadID uuid.UUID, paginacao domain.PaginacaoFiltro) ([]lead.LeadHistorico, error)
}
