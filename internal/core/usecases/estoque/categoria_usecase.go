package estoque

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/estoque"
	"github.com/google/uuid"
)

var _ portsin.CategoriaUseCase = (*CategoriaUsecaseImpl)(nil)

type CategoriaUsecaseImpl struct {
	repo portsout.CategoriaRepository
	permissoes
}

func NewCategoriaUsecase(
	repo portsout.CategoriaRepository,
	usuarios portsout.UsuarioRepository,
	cargos portsout.CargoRepository,
) *CategoriaUsecaseImpl {
	return &CategoriaUsecaseImpl{
		repo: repo,
		permissoes: permissoes{
			usuarios: usuarios,
			cargos:   cargos,
		},
	}
}

func (u *CategoriaUsecaseImpl) Criar(ctx context.Context, input portsin.CriarCategoriaInput) (uuid.UUID, error) {
	if err := u.exigeAdministrador(ctx, input.UsuarioID); err != nil {
		return uuid.Nil, err
	}

	var categoriaPai *domainestoque.Categoria
	if input.CategoriaPaiID != nil {
		pai, err := u.repo.Obter(ctx, *input.CategoriaPaiID)
		if err != nil {
			return uuid.Nil, err
		}
		if pai == nil {
			return uuid.Nil, domain.ErroNaoEncontrado("categoria pai não encontrada")
		}
		categoriaPai = pai
	}

	slug, err := domainestoque.NovoSlug(input.Nome)
	if err != nil {
		return uuid.Nil, err
	}

	existente, err := u.repo.ObterPorSlug(ctx, slug.Valor())
	if err != nil {
		return uuid.Nil, err
	}
	if existente != nil {
		return uuid.Nil, domain.ErroConflito("já existe uma categoria com este nome")
	}

	categoria, err := domainestoque.NovaCategoria(input.Nome, slug, categoriaPai, input.Ordem, input.Icone)
	if err != nil {
		return uuid.Nil, err
	}

	return u.repo.Criar(ctx, categoria)
}

func (u *CategoriaUsecaseImpl) Obter(ctx context.Context, id uuid.UUID) (*domainestoque.Categoria, error) {
	categoria, err := u.repo.Obter(ctx, id)
	if err != nil {
		return nil, err
	}
	if categoria == nil {
		return nil, domain.ErroNaoEncontrado("categoria não encontrada")
	}

	return categoria, nil
}

func (u *CategoriaUsecaseImpl) Listar(ctx context.Context, input portsin.ListarCategoriasInput) ([]*domainestoque.Categoria, error) {
	return u.repo.Listar(ctx, portsout.CategoriaFiltro{
		PaginacaoFiltro: input.Filtro.Normalizada(),
		Ativo:           input.Ativo,
	})
}

func (u *CategoriaUsecaseImpl) Alterar(ctx context.Context, input portsin.AlterarCategoriaInput) error {
	if err := u.exigeAdministrador(ctx, input.UsuarioID); err != nil {
		return err
	}

	categoria, err := u.Obter(ctx, input.CategoriaID)
	if err != nil {
		return err
	}

	if err := categoria.AlterarDados(input.Nome, input.Ordem, input.Icone, time.Now().UTC()); err != nil {
		return err
	}

	_, err = u.repo.Atualizar(ctx, categoria)

	return err
}

func (u *CategoriaUsecaseImpl) AlternarAtivo(ctx context.Context, input portsin.AlternarAtivoCategoriaInput) error {
	if err := u.exigeAdministrador(ctx, input.UsuarioID); err != nil {
		return err
	}

	categoria, err := u.Obter(ctx, input.CategoriaID)
	if err != nil {
		return err
	}

	possuiSubcategorias, err := u.repo.PossuiSubcategorias(ctx, categoria.ID)
	if err != nil {
		return err
	}

	if err := categoria.AlternarAtivo(input.Ativo, possuiSubcategorias, time.Now().UTC()); err != nil {
		return err
	}

	_, err = u.repo.Atualizar(ctx, categoria)

	return err
}
