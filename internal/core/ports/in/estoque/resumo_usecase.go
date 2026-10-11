package estoque

import (
	"context"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type ResumoInput struct {
	UsuarioID uuid.UUID
	Empresa   *uuid.UUID
}

type ResumoUseCase interface {
	Resumo(ctx context.Context, input ResumoInput) (*domainestoque.ResumoEstoque, error)
}
