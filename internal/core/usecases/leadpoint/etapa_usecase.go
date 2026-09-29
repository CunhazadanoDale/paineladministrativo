package leadpoint

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/leads"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/leads"
	"github.com/google/uuid"
)

var _ portsin.EtapaUseCase = (*EtapaUsecaseImpl)(nil)

type EtapaUsecaseImpl struct {
	repo portsout.EtapaRepository
}

func NewEtapaUsecase(repo portsout.EtapaRepository) *EtapaUsecaseImpl {
	return &EtapaUsecaseImpl{repo: repo}
}

func (e *EtapaUsecaseImpl) Create(ctx context.Context, etapa *lead.Etapa) (uuid.UUID, error) {
	if etapa == nil {
		return uuid.Nil, errors.New("etapa não informada")
	}

	etapa.Nome = strings.TrimSpace(etapa.Nome)

	if etapa.Nome == "" {
		return uuid.Nil, errors.New("nome da etapa é obrigatório")
	}
	if etapa.FunilID == uuid.Nil {
		return uuid.Nil, errors.New("funil da etapa é obrigatório")
	}
	if etapa.EtapaID == uuid.Nil {
		etapa.EtapaID = uuid.New()
	}

	if etapa.Ordem <= 0 {
		etapas, err := e.repo.ListByFunilOrdenado(ctx, etapa.FunilID)
		if err != nil {
			return uuid.Nil, err
		}
		etapa.Ordem = len(etapas) + 1
	}

	etapa.Ativo = true

	return e.repo.Create(ctx, etapa)
}

func (e *EtapaUsecaseImpl) Update(ctx context.Context, etapa *lead.Etapa) error {
	if etapa == nil || etapa.EtapaID == uuid.Nil {
		return errors.New("etapa inválida")
	}

	etapa.Nome = strings.TrimSpace(etapa.Nome)

	if etapa.Nome == "" {
		return errors.New("nome da etapa é obrigatório")
	}
	if etapa.FunilID == uuid.Nil {
		return errors.New("funil da etapa é obrigatório")
	}
	if etapa.Ordem <= 0 {
		return errors.New("ordem da etapa é obrigatória")
	}

	atual, err := e.repo.GetByID(ctx, etapa.EtapaID)
	if err != nil {
		return err
	}
	if atual == nil {
		return domain.ErrNotFound
	}

	return e.repo.Update(ctx, etapa)
}

func (e *EtapaUsecaseImpl) GetByID(ctx context.Context, id uuid.UUID) (*lead.Etapa, error) {
	if id == uuid.Nil {
		return nil, errors.New("id da etapa não informado")
	}

	item, err := e.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, domain.ErrNotFound
	}

	return item, nil
}

func (e *EtapaUsecaseImpl) ListByFunilID(ctx context.Context, funilID uuid.UUID) ([]*lead.Etapa, error) {
	if funilID == uuid.Nil {
		return nil, errors.New("funil não informado")
	}

	return e.repo.ListByFunilID(ctx, funilID)
}

func (e *EtapaUsecaseImpl) ListByFunilOrdenado(ctx context.Context, funilID uuid.UUID) ([]*lead.Etapa, error) {
	if funilID == uuid.Nil {
		return nil, errors.New("funil não informado")
	}

	return e.repo.ListByFunilOrdenado(ctx, funilID)
}

func (e *EtapaUsecaseImpl) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("id da etapa não informado")
	}

	atual, err := e.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if atual == nil {
		return domain.ErrNotFound
	}

	return e.repo.Delete(ctx, id)
}

func (e *EtapaUsecaseImpl) Reordenar(ctx context.Context, funilID uuid.UUID, etapas []*lead.Etapa) error {
	if funilID == uuid.Nil {
		return errors.New("funil não informado")
	}
	if len(etapas) == 0 {
		return errors.New("nenhuma etapa informada para reordenar")
	}
	for i, etapa := range etapas {
		if etapa == nil {
			return fmt.Errorf("etapa inválida na posição %d", i)
		}
		if etapa.FunilID != funilID {
			return errors.New("etapa " + etapa.Nome + " não pertence ao funil informado")
		}
	}

	return e.repo.Reordenar(ctx, funilID, etapas)
}

func (e *EtapaUsecaseImpl) GetNextEtapa(ctx context.Context, currentEtapaID uuid.UUID) (*lead.Etapa, error) {
	if currentEtapaID == uuid.Nil {
		return nil, errors.New("etapa atual não informada")
	}

	return e.repo.GetNextEtapa(ctx, currentEtapaID)
}

func (e *EtapaUsecaseImpl) GetPreviousEtapa(ctx context.Context, currentEtapaID uuid.UUID) (*lead.Etapa, error) {
	if currentEtapaID == uuid.Nil {
		return nil, errors.New("etapa atual não informada")
	}

	return e.repo.GetPreviousEtapa(ctx, currentEtapaID)
}

func (e *EtapaUsecaseImpl) ExistsByFunil(ctx context.Context, funilID uuid.UUID) (bool, error) {
	if funilID == uuid.Nil {
		return false, errors.New("funil não informado")
	}

	return e.repo.ExistsByFunil(ctx, funilID)
}
