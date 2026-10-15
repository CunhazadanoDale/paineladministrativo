package pedido

import (
	"context"

	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	"github.com/google/uuid"
)

type ObraRepository interface {
	Obter(ctx context.Context, id uuid.UUID) (*domaincliente.Obra, error)
}
