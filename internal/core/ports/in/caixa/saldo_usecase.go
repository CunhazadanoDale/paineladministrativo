package caixa

import (
	"context"

	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	"github.com/google/uuid"
)

type SaldoUseCase interface {
	Listar(ctx context.Context, usuarioID uuid.UUID, empresa *uuid.UUID) ([]*domaincaixa.SaldoConta, error)
}
