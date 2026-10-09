package estoque

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type CategoriaFiltro struct {
	domain.PaginacaoFiltro
	Ativo *bool
}

type CategoriaRepository interface {
	Criar(ctx context.Context, categoria *domainestoque.Categoria) (uuid.UUID, error)
	Obter(ctx context.Context, id uuid.UUID) (*domainestoque.Categoria, error)
	ObterPorSlug(ctx context.Context, slug string) (*domainestoque.Categoria, error)
	Listar(ctx context.Context, filtro CategoriaFiltro) ([]*domainestoque.Categoria, error)
	Atualizar(ctx context.Context, categoria *domainestoque.Categoria) (bool, error)
	PossuiSubcategorias(ctx context.Context, id uuid.UUID) (bool, error)
	PossuiProdutos(ctx context.Context, id uuid.UUID) (bool, error)
}
