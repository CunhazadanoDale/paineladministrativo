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
	repo     portsout.CargoRepository
	usuarios portsout.UsuarioRepository
}

func NewCargoUsecase(repo portsout.CargoRepository, usuarios portsout.UsuarioRepository) *CargoUsecaseImpl {
	return &CargoUsecaseImpl{repo: repo, usuarios: usuarios}
}

func (c *CargoUsecaseImpl) Criar(ctx context.Context, cargo *domainusuarios.Cargo) (uuid.UUID, error) {
	if cargo == nil {
		return uuid.Nil, domain.ErroValidacao("cargo não informado")
	}

	nome := strings.TrimSpace(cargo.Nome)
	if nome == "" {
		return uuid.Nil, domain.ErroValidacao("nome do cargo é obrigatório")
	}

	existente, err := c.repo.ObterPorNome(ctx, nome)
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

	return c.repo.Criar(ctx, novo)
}

func (c *CargoUsecaseImpl) Remover(ctx context.Context, id uuid.UUID) error {
	if _, err := c.buscar(ctx, id); err != nil {
		return err
	}

	return c.repo.Remover(ctx, id)
}

func (c *CargoUsecaseImpl) Obter(ctx context.Context, id uuid.UUID) (*domainusuarios.Cargo, error) {
	return c.buscar(ctx, id)
}

func (c *CargoUsecaseImpl) ObterPorNome(ctx context.Context, nome string) (*domainusuarios.Cargo, error) {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return nil, domain.ErroValidacao("nome do cargo não informado")
	}

	cargo, err := c.repo.ObterPorNome(ctx, nome)
	if err != nil {
		return nil, err
	}
	if cargo == nil {
		return nil, domain.ErrNotFound
	}

	return cargo, nil
}

func (c *CargoUsecaseImpl) Listar(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*domainusuarios.Cargo, error) {
	return c.repo.Listar(ctx, filtro.Normalizada())
}

func (c *CargoUsecaseImpl) ListarAtivos(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*domainusuarios.Cargo, error) {
	return c.repo.ListarAtivos(ctx, filtro.Normalizada())
}

func (c *CargoUsecaseImpl) Atualizar(ctx context.Context, cargo *domainusuarios.Cargo) error {
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

	existente, err := c.repo.ObterPorNome(ctx, cargo.Nome)
	if err != nil {
		return err
	}
	if existente != nil && existente.ID != atual.ID {
		return domain.ErroValidacao("já existe um cargo com esse nome")
	}
	if atual.Administrador && !cargo.Administrador {
		if err := c.garantirAdministradorForaDoCargo(ctx, atual.ID); err != nil {
			return err
		}
	}

	return c.repo.Atualizar(ctx, cargo)
}

func (c *CargoUsecaseImpl) garantirAdministradorForaDoCargo(ctx context.Context, cargoID uuid.UUID) error {
	noCargo, err := c.usuarios.ContarAtivosPorCargo(ctx, cargoID)
	if err != nil || noCargo == 0 {
		return err
	}

	total, err := c.usuarios.ContarAdministradoresAtivos(ctx)
	if err != nil {
		return err
	}
	if total-noCargo < 1 {
		return domain.ErroConflito(mensagemUltimoAdministrador)
	}

	return nil
}

func (c *CargoUsecaseImpl) buscar(ctx context.Context, id uuid.UUID) (*domainusuarios.Cargo, error) {
	if id == uuid.Nil {
		return nil, domain.ErroValidacao("id do cargo não informado")
	}

	cargo, err := c.repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	if cargo == nil {
		return nil, domain.ErrNotFound
	}

	return cargo, nil
}
