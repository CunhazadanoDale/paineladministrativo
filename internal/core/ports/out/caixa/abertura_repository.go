package caixa

import (
	"context"

	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	"github.com/google/uuid"
)

type AberturaRepository interface {
	Obter(ctx context.Context, contaBancariaID uuid.UUID) (*domaincaixa.Abertura, error)
	Criar(ctx context.Context, abertura *domaincaixa.Abertura) error
	Atualizar(ctx context.Context, abertura *domaincaixa.Abertura) error
}
