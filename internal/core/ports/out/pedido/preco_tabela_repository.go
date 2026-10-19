package pedido

import (
	"context"
	"time"

	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	"github.com/google/uuid"
)

type PrecoTabelaRepository interface {
	PrecoVigente(ctx context.Context, produtoID uuid.UUID, modelo domainpedido.Modelo, data time.Time) (*int64, error)
}
