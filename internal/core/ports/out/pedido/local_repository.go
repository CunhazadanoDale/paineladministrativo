package pedido

import (
	"context"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type LocalRepository interface {
	ObterVarios(ctx context.Context, ids []uuid.UUID) ([]*domainestoque.Local, error)
	AtivosDaEmpresa(ctx context.Context, empresaID uuid.UUID) ([]*domainestoque.Local, error)
}
