package pedido

import (
	"context"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type EstoqueRepository interface {
	Carregar(ctx context.Context, produtoID uuid.UUID, localIDs []uuid.UUID) (*domainestoque.EstoqueProduto, error)
}
