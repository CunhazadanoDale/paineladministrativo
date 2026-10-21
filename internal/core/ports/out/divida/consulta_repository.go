package divida

import (
	"context"

	domaindivida "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/divida"
	"github.com/google/uuid"
)

type ConsultaRepository interface {
	ParcelasEmAberto(ctx context.Context, empresaIDs []uuid.UUID) ([]domaindivida.ParcelaEmAberto, error)
}
