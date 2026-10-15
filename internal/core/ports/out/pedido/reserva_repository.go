package pedido

import (
	"context"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type ReservaRepository interface {
	AtivasPorOrigem(ctx context.Context, origemIDs []uuid.UUID) ([]*domainestoque.Reserva, error)
}
