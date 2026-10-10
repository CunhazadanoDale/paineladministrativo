package estoque

import (
	"context"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/estoque"
	"github.com/google/uuid"
)

func produtoVisivelNaVitrine(ctx context.Context, categorias portsout.CategoriaRepository, produto *domainestoque.Produto) (bool, error) {
	if produto == nil || !produto.Ativo || produto.Saldo.Vazio() {
		return false, nil
	}

	return categoriaVisivel(ctx, categorias, produto.CategoriaID)
}

func categoriaVisivel(ctx context.Context, categorias portsout.CategoriaRepository, categoriaID uuid.UUID) (bool, error) {
	categoria, err := categorias.Obter(ctx, categoriaID)
	if err != nil || categoria == nil || !categoria.Ativo {
		return false, err
	}
	if categoria.CategoriaPaiID == nil {
		return true, nil
	}

	pai, err := categorias.Obter(ctx, *categoria.CategoriaPaiID)
	if err != nil {
		return false, err
	}

	return pai != nil && pai.Ativo, nil
}
