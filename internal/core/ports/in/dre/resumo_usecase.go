package dre

import (
	"context"

	domaindre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/dre"
	"github.com/google/uuid"
)

type ConsultarPeriodoInput struct {
	UsuarioID uuid.UUID
	De        *domaindre.Mes
	Ate       *domaindre.Mes
	Empresa   *uuid.UUID
}

type ResumoDREUseCase interface {
	Consultar(ctx context.Context, input ConsultarPeriodoInput) (domaindre.ResumoDRE, error)
}
