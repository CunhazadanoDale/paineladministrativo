package lead

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainlead "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/leads"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/leads"
	"github.com/google/uuid"
)

var _ portsin.LeadHistoricoUseCase = (*LeadHistoricoUsecaseImpl)(nil)

type LeadHistoricoUsecaseImpl struct {
	repo portsout.LeadHistoricoRepository
}

func NewLeadHistoricoUsecase(repo portsout.LeadHistoricoRepository) *LeadHistoricoUsecaseImpl {
	return &LeadHistoricoUsecaseImpl{repo: repo}
}

func (h *LeadHistoricoUsecaseImpl) ListarPorLead(ctx context.Context, leadID uuid.UUID, paginacao domain.PaginacaoFiltro) ([]domainlead.LeadHistorico, error) {
	if leadID == uuid.Nil {
		return nil, domain.ErroValidacao("lead não informado")
	}

	return h.repo.ListarPorLead(ctx, leadID, paginacao.Normalizada())
}
