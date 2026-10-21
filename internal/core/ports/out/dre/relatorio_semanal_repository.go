package dre

import (
	"context"
	"time"

	domaindre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/dre"
	"github.com/google/uuid"
)

type FiltroSemanal struct {
	Intervalo  domaindre.Intervalo
	Hoje       time.Time
	EmpresaIDs []uuid.UUID
	Limite     int
}

type RelatorioSemanalRepository interface {
	NotasEmitidas(ctx context.Context, filtro FiltroSemanal) ([]domaindre.NotasEmitidas, error)
	PagamentosFeitos(ctx context.Context, filtro FiltroSemanal) ([]domaindre.PagamentosFeitos, error)
	RecebimentosFeitos(ctx context.Context, filtro FiltroSemanal) ([]domaindre.RecebimentosFeitos, error)
	TitulosVencidos(ctx context.Context, filtro FiltroSemanal) (domaindre.Vencidos[domaindre.TituloVencido], error)
	ContasVencidas(ctx context.Context, filtro FiltroSemanal) (domaindre.Vencidos[domaindre.ContaVencida], error)
	EstoqueBaixo(ctx context.Context, limite int) (domaindre.EstoqueBaixo, error)
}
