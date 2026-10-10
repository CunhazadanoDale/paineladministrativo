package leadpoint

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/leads"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/leads"
	"github.com/google/uuid"
)

var _ portsin.LeadHistoryUseCase = (*LeadHistoryUsecaseImpl)(nil)

type LeadHistoryUsecaseImpl struct {
	repo portsout.LeadHistoryRepository
}

func NewLeadHistoryUsecase(repo portsout.LeadHistoryRepository) *LeadHistoryUsecaseImpl {
	return &LeadHistoryUsecaseImpl{repo: repo}
}

func (h *LeadHistoryUsecaseImpl) ListByLead(ctx context.Context, leadID uuid.UUID, paginacao domain.PaginacaoFiltro) ([]lead.LeadHistorico, error) {
	if leadID == uuid.Nil {
		return nil, domain.ErroValidacao("lead não informado")
	}

	return h.repo.ListByLead(ctx, leadID, paginacao.Normalizada())
}
