package solicitacao

import (
	"context"
	"time"

	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

type FiltroResumo struct {
	De         time.Time
	Ate        time.Time
	Hoje       time.Time
	EmpresaIDs []uuid.UUID
}

type ResumoContasAPagarRepository interface {
	ResumoSemanal(ctx context.Context, filtro FiltroResumo) ([]domainsolicitacao.LinhaResumoSemanal, error)
	Atrasadas(ctx context.Context, filtro FiltroResumo) ([]domainsolicitacao.AtrasoPorEmpresa, error)
}
