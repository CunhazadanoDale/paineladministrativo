package caixa

import (
	"context"

	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	"github.com/google/uuid"
)

type CategoriaFinanceiraRepository interface {
	Obter(ctx context.Context, id uuid.UUID) (*domainfinanceiro.CategoriaFinanceira, error)
}
