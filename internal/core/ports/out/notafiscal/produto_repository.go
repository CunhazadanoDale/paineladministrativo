package notafiscal

import (
	"context"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type ProdutoRepository interface {
	Obter(ctx context.Context, id uuid.UUID) (*domainestoque.Produto, error)
	IDsPorCodigos(ctx context.Context, codigos []string) (map[string]uuid.UUID, error)
}
