package dre

import (
	"context"

	domaindre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/dre"
	"github.com/google/uuid"
)

type FiltroResumo struct {
	Intervalo  domaindre.Intervalo
	EmpresaIDs []uuid.UUID
}

type ResumoRepository interface {
	Lancamentos(ctx context.Context, filtro FiltroResumo) ([]domaindre.Lancamento, error)
	ConsumosSemCusto(ctx context.Context, filtro FiltroResumo) ([]domaindre.ConsumoSemCusto, error)
	Faturamento(ctx context.Context, filtro FiltroResumo) ([]domaindre.Faturamento, error)
}
