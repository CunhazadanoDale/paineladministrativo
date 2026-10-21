package dre

import (
	"context"
	"time"

	domaindre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/dre"
	"github.com/google/uuid"
)

type ConsultarSemanaInput struct {
	UsuarioID uuid.UUID
	Semana    *time.Time
	Empresa   *uuid.UUID
}

type RelatorioSemanalUseCase interface {
	Consultar(ctx context.Context, input ConsultarSemanaInput) (domaindre.RelatorioSemanal, error)
}
