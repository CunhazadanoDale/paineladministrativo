package estoque

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
)

type ListarProdutosPublicosInput struct {
	CategoriaSlug string
	Filtro        domain.PaginacaoFiltro
}

type PublicoUseCase interface {
	ListarCategorias(ctx context.Context) ([]*domainestoque.Categoria, error)
	ListarProdutos(ctx context.Context, input ListarProdutosPublicosInput) ([]*domainestoque.Produto, error)
	ObterProdutoPorSlug(ctx context.Context, slug string) (*domainestoque.Produto, error)
	ListarDestaques(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*domainestoque.Produto, error)
}
