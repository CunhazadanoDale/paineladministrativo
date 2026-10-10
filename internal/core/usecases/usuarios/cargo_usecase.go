package usuarios

import (
	"context"
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/usuarios"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/usuarios"
	"github.com/google/uuid"
)

var _ portsin.CargoUseCase = (*CargoUsecaseImpl)(nil)

type CargoUsecaseImpl struct {
	repo portsout.CargoRepository
}

func NewCargoUsecase(repo portsout.CargoRepository) *CargoUsecaseImpl {
	return &CargoUsecaseImpl{repo: repo}
}

func (c *CargoUsecaseImpl) Create(ctx context.Context, cargo *domainusuarios.Cargo) (uuid.UUID, error) {
	if cargo == nil {
		return uuid.Nil, domain.ErroValidacao("cargo não informado")
	}

	nome := strings.TrimSpace(cargo.Nome)
	if nome == "" {
		return uuid.Nil, domain.ErroValidacao("nome do cargo é obrigatório")
	}

	existente, err := c.repo.GetByNome(ctx, nome)
	if err != nil {
		return uuid.Nil, err
	}
	if existente != nil {
		return uuid.Nil, domain.ErroValidacao("já existe um cargo com esse nome")
	}

	novo := &domainusuarios.Cargo{
		ID:            uuid.New(),
		Nome:          nome,
		Descricao:     strings.TrimSpace(cargo.Descricao),
		Ativo:         true,
		Administrador: cargo.Administrador,
		Financeiro:    cargo.Financeiro,
		Comercial:     cargo.Comercial,
	}

	return c.repo.Create(ctx, novo)
}

func (c *CargoUsecaseImpl) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := c.buscar(ctx, id); err != nil {
		return err
	}

	return c.repo.Delete(ctx, id)
}

func (c *CargoUsecaseImpl) GetByID(ctx context.Context, id uuid.UUID) (*domainusuarios.Cargo, error) {
	return c.buscar(ctx, id)
}

func (c *CargoUsecaseImpl) GetByNome(ctx context.Context, nome string) (*domainusuarios.Cargo, error) {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return nil, domain.ErroValidacao("nome do cargo não informado")
	}

	cargo, err := c.repo.GetByNome(ctx, nome)
	if err != nil {
		return nil, err
	}
	if cargo == nil {
		return nil, domain.ErrNotFound
	}

	return cargo, nil
}

func (c *CargoUsecaseImpl) List(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*domainusuarios.Cargo, error) {
	return c.repo.List(ctx, filtro.Normalizada())
}

func (c *CargoUsecaseImpl) ListAtivos(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*domainusuarios.Cargo, error) {
	return c.repo.ListAtivos(ctx, filtro.Normalizada())
}

func (c *CargoUsecaseImpl) Update(ctx context.Context, cargo *domainusuarios.Cargo) error {
	if cargo == nil || cargo.ID == uuid.Nil {
		return domain.ErroValidacao("cargo inválido")
	}

	cargo.Nome = strings.TrimSpace(cargo.Nome)
	cargo.Descricao = strings.TrimSpace(cargo.Descricao)

	if cargo.Nome == "" {
		return domain.ErroValidacao("nome do cargo é obrigatório")
	}

	atual, err := c.buscar(ctx, cargo.ID)
	if err != nil {
		return err
	}

	existente, err := c.repo.GetByNome(ctx, cargo.Nome)
	if err != nil {
		return err
	}
	if existente != nil && existente.ID != atual.ID {
		return domain.ErroValidacao("já existe um cargo com esse nome")
	}

	return c.repo.Update(ctx, cargo)
}

func (c *CargoUsecaseImpl) buscar(ctx context.Context, id uuid.UUID) (*domainusuarios.Cargo, error) {
	if id == uuid.Nil {
		return nil, domain.ErroValidacao("id do cargo não informado")
	}

	cargo, err := c.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cargo == nil {
		return nil, domain.ErrNotFound
	}

	return cargo, nil
}
