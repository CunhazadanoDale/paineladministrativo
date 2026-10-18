package receber

import (
	"context"
	"time"

	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	"github.com/google/uuid"
)

type FiltroResumo struct {
	De         time.Time
	Ate        time.Time
	Hoje       time.Time
	EmpresaIDs []uuid.UUID
}

type ResumoContasAReceberRepository interface {
	ResumoSemanal(ctx context.Context, filtro FiltroResumo) ([]domainreceber.LinhaResumoSemanal, error)
	Atrasados(ctx context.Context, filtro FiltroResumo) ([]domainreceber.AtrasoPorEmpresa, error)
}
