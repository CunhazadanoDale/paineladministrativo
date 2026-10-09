package estoque

import (
	"context"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type ImagemRepository interface {
	Criar(ctx context.Context, imagem *domainestoque.Imagem) (uuid.UUID, error)
	Obter(ctx context.Context, id uuid.UUID) (*domainestoque.Imagem, error)
	ListarPorProduto(ctx context.Context, produtoID uuid.UUID) ([]*domainestoque.Imagem, error)
	ListarPorProdutos(ctx context.Context, produtoIDs []uuid.UUID) ([]*domainestoque.Imagem, error)
	Remover(ctx context.Context, id uuid.UUID) (bool, error)
}
