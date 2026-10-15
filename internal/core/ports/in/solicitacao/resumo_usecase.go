package solicitacao

import (
	"context"
	"time"

	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

type ConsultarResumoInput struct {
	UsuarioID uuid.UUID
	De        *time.Time
	Ate       *time.Time
	Empresa   *uuid.UUID
}

type ResumoContasAPagarUseCase interface {
	Consultar(ctx context.Context, input ConsultarResumoInput) (domainsolicitacao.ResumoContasAPagar, error)
}
