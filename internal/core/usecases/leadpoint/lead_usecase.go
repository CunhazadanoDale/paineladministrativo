package leadpoint

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/leads"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/leads"
	"github.com/google/uuid"
)

var _ portsin.LeadUseCase = (*LeadUsecaseImpl)(nil)

type LeadUsecaseImpl struct {
	repo portsout.LeadRepository
}

func NewLeadUsecase(repo portsout.LeadRepository) *LeadUsecaseImpl {
	return &LeadUsecaseImpl{repo: repo}
}

func (l *LeadUsecaseImpl) Create(ctx context.Context, lead *lead.Lead) (uuid.UUID, error) {
	if lead == nil {
		return uuid.Nil, errors.New("lead não informado")
	}

	lead.Nome = strings.TrimSpace(lead.Nome)
	lead.Email = strings.TrimSpace(lead.Email)

	if lead.Nome == "" {
		return uuid.Nil, errors.New("nome do lead é obrigatório")
	}
	if lead.EtapaID == uuid.Nil {
		return uuid.Nil, errors.New("etapa do lead é obrigatória")
	}
	if lead.ID == uuid.Nil {
		lead.ID = uuid.New()
	}

	agora := time.Now().UTC()
	if lead.CriadoEm.IsZero() {
		lead.CriadoEm = agora
	}
	lead.AtualizadoEm = agora
	lead.Ativo = true

	return l.repo.Create(ctx, lead)
}

func (l *LeadUsecaseImpl) Update(ctx context.Context, lead *lead.Lead) error {
	if lead == nil || lead.ID == uuid.Nil {
		return errors.New("lead inválido")
	}

	lead.Nome = strings.TrimSpace(lead.Nome)
	lead.Email = strings.TrimSpace(lead.Email)

	if lead.Nome == "" {
		return errors.New("nome do lead é obrigatório")
	}

	atual, err := l.repo.GetByID(ctx, lead.ID)
	if err != nil {
		return err
	}
	if atual == nil {
		return domain.ErrNotFound
	}

	lead.AtualizadoEm = time.Now().UTC()

	return l.repo.Update(ctx, lead)
}

func (l *LeadUsecaseImpl) GetByID(ctx context.Context, id uuid.UUID) (*lead.Lead, error) {
	if id == uuid.Nil {
		return nil, errors.New("id do lead não informado")
	}

	item, err := l.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, domain.ErrNotFound
	}

	return item, nil
}

func (l *LeadUsecaseImpl) ListByFunil(ctx context.Context, funilID uuid.UUID) ([]*lead.Lead, error) {
	if funilID == uuid.Nil {
		return nil, errors.New("funil não informado")
	}

	return l.repo.ListByFunil(ctx, funilID)
}

func (l *LeadUsecaseImpl) ListByEtapa(ctx context.Context, etapaID uuid.UUID) ([]*lead.Lead, error) {
	if etapaID == uuid.Nil {
		return nil, errors.New("etapa não informada")
	}

	return l.repo.ListByEtapa(ctx, etapaID)
}

func (l *LeadUsecaseImpl) ListAtivos(ctx context.Context, paginacao domain.PaginacaoFiltro) ([]*lead.Lead, error) {
	return l.repo.ListAtivos(ctx, paginacao.Normalizada())
}

func (l *LeadUsecaseImpl) Search(ctx context.Context, query string, paginacao domain.PaginacaoFiltro) ([]*lead.Lead, error) {
	return l.repo.Search(ctx, strings.TrimSpace(query), paginacao.Normalizada())
}

func (l *LeadUsecaseImpl) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("id do lead não informado")
	}

	atual, err := l.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if atual == nil {
		return domain.ErrNotFound
	}

	return l.repo.Delete(ctx, id)
}

func (l *LeadUsecaseImpl) CountByEtapa(ctx context.Context, etapaID uuid.UUID) (int, error) {
	if etapaID == uuid.Nil {
		return 0, errors.New("etapa não informada")
	}

	return l.repo.CountByEtapa(ctx, etapaID)
}

func (l *LeadUsecaseImpl) CountByFunil(ctx context.Context, funilID uuid.UUID) (int, error) {
	if funilID == uuid.Nil {
		return 0, errors.New("funil não informado")
	}

	return l.repo.CountByFunil(ctx, funilID)
}

func (l *LeadUsecaseImpl) UpdateEtapa(ctx context.Context, leadID uuid.UUID, newEtapaID uuid.UUID) error {
	if leadID == uuid.Nil {
		return errors.New("lead não informado")
	}
	if newEtapaID == uuid.Nil {
		return errors.New("etapa de destino não informada")
	}

	atual, err := l.repo.GetByID(ctx, leadID)
	if err != nil {
		return err
	}
	if atual == nil {
		return domain.ErrNotFound
	}
	if atual.EtapaID == newEtapaID {
		return nil
	}

	return l.repo.MoverParaEtapa(ctx, leadID, atual.EtapaID, newEtapaID)
}
