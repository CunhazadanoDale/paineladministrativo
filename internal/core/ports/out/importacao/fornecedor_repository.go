package importacao

import (
	"context"

	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	"github.com/google/uuid"
)

type FornecedorRepository interface {
	Obter(ctx context.Context, id uuid.UUID) (*domainfornecedor.Fornecedor, error)
}
