package caixa

import (
	"context"
	"time"

	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	"github.com/google/uuid"
)

type ProjetarInput struct {
	UsuarioID uuid.UUID
	Empresa   *uuid.UUID
	Ate       *time.Time
}

type MovimentoDiarioInput struct {
	UsuarioID       uuid.UUID
	Empresa         *uuid.UUID
	ContaBancariaID *uuid.UUID
	De              *time.Time
	Ate             *time.Time
}

type ProjecaoUseCase interface {
	Projetar(ctx context.Context, input ProjetarInput) (*domaincaixa.Projecao, error)
	MovimentoDiario(ctx context.Context, input MovimentoDiarioInput) ([]*domaincaixa.MovimentoDiario, error)
}
