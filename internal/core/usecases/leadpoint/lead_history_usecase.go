package leadpoint

import (
	"context"
	"errors"

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
		return nil, errors.New("lead não informado")
	}

	return h.repo.ListByLead(ctx, leadID, paginacao.Normalizada())
}

func (h *LeadHistoryUsecaseImpl) RegistrarMovimentacao(ctx context.Context, leadID uuid.UUID, etapaAnteriorID uuid.UUID, etapaAtualID uuid.UUID) error {
	if leadID == uuid.Nil {
		return errors.New("lead não informado")
	}
	if etapaAnteriorID == uuid.Nil {
		return errors.New("etapa anterior não informada")
	}
	if etapaAtualID == uuid.Nil {
		return errors.New("etapa atual não informada")
	}
	if etapaAnteriorID == etapaAtualID {
		return errors.New("etapa anterior e atual não podem ser iguais")
	}

	return h.repo.RegistrarMovimentacao(ctx, leadID, etapaAnteriorID, etapaAtualID)
}
