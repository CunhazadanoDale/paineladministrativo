package dre

import (
	"context"

	domaindre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/dre"
)

type FaturamentoUseCase interface {
	Consultar(ctx context.Context, input ConsultarPeriodoInput) (domaindre.ResumoFaturamento, error)
}
