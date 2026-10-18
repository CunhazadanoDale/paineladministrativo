package receber

import (
	"context"
	"time"

	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	"github.com/google/uuid"
)

type ConsultarResumoInput struct {
	UsuarioID uuid.UUID
	Empresa   *uuid.UUID
	De        *time.Time
	Ate       *time.Time
}

type ResumoContasAReceberUseCase interface {
	Consultar(ctx context.Context, input ConsultarResumoInput) (domainreceber.ResumoContasAReceber, error)
}
