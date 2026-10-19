package caixa

import (
	"context"
	"time"

	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	"github.com/google/uuid"
)

type DefinirAberturaInput struct {
	UsuarioID       uuid.UUID
	ContaBancariaID uuid.UUID
	DataCorte       time.Time
	SaldoCentavos   int64
	Versao          *int
}

type AberturaUseCase interface {
	Definir(ctx context.Context, input DefinirAberturaInput) (*domaincaixa.Abertura, error)
	Obter(ctx context.Context, contaBancariaID, usuarioID uuid.UUID) (*domaincaixa.Abertura, error)
}
