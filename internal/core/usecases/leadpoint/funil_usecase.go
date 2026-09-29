package leadpoint

import (
	"context"
	"errors"
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/leads"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/leads"
	"github.com/google/uuid"
)

var _ portsin.FunilUseCase = (*FunilUsecaseImpl)(nil)

type FunilUsecaseImpl struct {
	repo portsout.FunilRepository
}

func NewFunilUsecase(repo portsout.FunilRepository) *FunilUsecaseImpl {
	return &FunilUsecaseImpl{repo: repo}
}

func (f *FunilUsecaseImpl) Create(ctx context.Context, funil *lead.Funil) (uuid.UUID, error) {
	if funil == nil {
		return uuid.Nil, errors.New("funil não informado")
	}

	funil.Nome = strings.TrimSpace(funil.Nome)

	if funil.Nome == "" {
		return uuid.Nil, errors.New("nome do funil é obrigatório")
	}
	if funil.FunilID == uuid.Nil {
		funil.FunilID = uuid.New()
	}

	funil.Ativo = true

	return f.repo.Create(ctx, funil)
}

func (f *FunilUsecaseImpl) Update(ctx context.Context, funil *lead.Funil) error {
	if funil == nil || funil.FunilID == uuid.Nil {
		return errors.New("funil inválido")
	}

	funil.Nome = strings.TrimSpace(funil.Nome)

	if funil.Nome == "" {
		return errors.New("nome do funil é obrigatório")
	}

	atual, err := f.repo.GetByID(ctx, funil.FunilID)
	if err != nil {
		return err
	}
	if atual == nil {
		return domain.ErrNotFound
	}

	return f.repo.Update(ctx, funil)
}

func (f *FunilUsecaseImpl) GetByID(ctx context.Context, funilID uuid.UUID) (*lead.Funil, error) {
	if funilID == uuid.Nil {
		return nil, errors.New("id do funil não informado")
	}

	item, err := f.repo.GetByID(ctx, funilID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, domain.ErrNotFound
	}

	return item, nil
}

func (f *FunilUsecaseImpl) List(ctx context.Context, filtro domain.PaginacaoFiltro) ([]lead.Funil, error) {
	return f.repo.List(ctx, filtro.Normalizada())
}

func (f *FunilUsecaseImpl) ListAtivos(ctx context.Context, filtro domain.PaginacaoFiltro) ([]lead.Funil, error) {
	return f.repo.ListAtivos(ctx, filtro.Normalizada())
}

func (f *FunilUsecaseImpl) Delete(ctx context.Context, funilID uuid.UUID) error {
	if funilID == uuid.Nil {
		return errors.New("id do funil não informado")
	}

	atual, err := f.repo.GetByID(ctx, funilID)
	if err != nil {
		return err
	}
	if atual == nil {
		return domain.ErrNotFound
	}

	return f.repo.Delete(ctx, funilID)
}

func (f *FunilUsecaseImpl) ExistsByID(ctx context.Context, funilID uuid.UUID) (bool, error) {
	if funilID == uuid.Nil {
		return false, errors.New("id do funil não informado")
	}

	return f.repo.ExistsByID(ctx, funilID)
}
