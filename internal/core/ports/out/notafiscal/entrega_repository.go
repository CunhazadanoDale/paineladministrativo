package notafiscal

import (
	"context"

	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	"github.com/google/uuid"
)

type EntregaRepository interface {
	Obter(ctx context.Context, id uuid.UUID) (*domainpedido.Entrega, error)
}
