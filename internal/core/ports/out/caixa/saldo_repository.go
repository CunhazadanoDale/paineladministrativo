package caixa

import (
	"context"

	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	"github.com/google/uuid"
)

type SaldoRepository interface {
	Saldos(ctx context.Context, empresaIDs []uuid.UUID) ([]*domaincaixa.SaldoConta, error)
}
