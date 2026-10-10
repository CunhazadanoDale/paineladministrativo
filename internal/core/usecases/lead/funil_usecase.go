package lead

import (
	"context"
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainlead "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
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

func (f *FunilUsecaseImpl) Criar(ctx context.Context, funil *domainlead.Funil) (uuid.UUID, error) {
	if funil == nil {
		return uuid.Nil, domain.ErroValidacao("funil não informado")
	}

	funil.Nome = strings.TrimSpace(funil.Nome)

	if funil.Nome == "" {
		return uuid.Nil, domain.ErroValidacao("nome do funil é obrigatório")
	}
	if funil.FunilID == uuid.Nil {
		funil.FunilID = uuid.New()
	}

	funil.Ativo = true

	return f.repo.Criar(ctx, funil)
}

func (f *FunilUsecaseImpl) Atualizar(ctx context.Context, funil *domainlead.Funil) error {
	if funil == nil || funil.FunilID == uuid.Nil {
		return domain.ErroValidacao("funil inválido")
	}

	funil.Nome = strings.TrimSpace(funil.Nome)

	if funil.Nome == "" {
		return domain.ErroValidacao("nome do funil é obrigatório")
	}

	atual, err := f.repo.Obter(ctx, funil.FunilID)
	if err != nil {
		return err
	}
	if atual == nil {
		return domain.ErrNotFound
	}

	return f.repo.Atualizar(ctx, funil)
}

func (f *FunilUsecaseImpl) Obter(ctx context.Context, funilID uuid.UUID) (*domainlead.Funil, error) {
	if funilID == uuid.Nil {
		return nil, domain.ErroValidacao("id do funil não informado")
	}

	item, err := f.repo.Obter(ctx, funilID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, domain.ErrNotFound
	}

	return item, nil
}

func (f *FunilUsecaseImpl) Listar(ctx context.Context, filtro domain.PaginacaoFiltro) ([]domainlead.Funil, error) {
	return f.repo.Listar(ctx, filtro.Normalizada())
}

func (f *FunilUsecaseImpl) ListarAtivos(ctx context.Context, filtro domain.PaginacaoFiltro) ([]domainlead.Funil, error) {
	return f.repo.ListarAtivos(ctx, filtro.Normalizada())
}

func (f *FunilUsecaseImpl) Remover(ctx context.Context, funilID uuid.UUID) error {
	if funilID == uuid.Nil {
		return domain.ErroValidacao("id do funil não informado")
	}

	atual, err := f.repo.Obter(ctx, funilID)
	if err != nil {
		return err
	}
	if atual == nil {
		return domain.ErrNotFound
	}

	return f.repo.Remover(ctx, funilID)
}

func (f *FunilUsecaseImpl) Existe(ctx context.Context, funilID uuid.UUID) (bool, error) {
	if funilID == uuid.Nil {
		return false, domain.ErroValidacao("id do funil não informado")
	}

	return f.repo.Existe(ctx, funilID)
}
