package estoque

import (
	"context"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type ProdutoImportacaoRepository interface {
	Criar(ctx context.Context, produto *domainestoque.Produto) (uuid.UUID, error)
	ObterPorSlug(ctx context.Context, slug string) (*domainestoque.Produto, error)
	IDsPorCodigos(ctx context.Context, codigos []string) (map[string]uuid.UUID, error)
}

type CategoriaPorSlugRepository interface {
	ObterPorSlug(ctx context.Context, slug string) (*domainestoque.Categoria, error)
}
