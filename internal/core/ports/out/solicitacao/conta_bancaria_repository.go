package solicitacao

import (
	"context"

	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	"github.com/google/uuid"
)

type ContaBancariaRepository interface {
	Obter(ctx context.Context, id uuid.UUID) (*domainfinanceiro.ContaBancaria, error)
}
