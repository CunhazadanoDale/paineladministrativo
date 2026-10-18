package receber

import (
	"context"
	"time"

	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	"github.com/google/uuid"
)

type GerarTituloDeEntradaInput struct {
	UsuarioID       uuid.UUID
	PedidoID        uuid.UUID
	Vencimento      time.Time
	ContaBancariaID *uuid.UUID
}

type EntradaDoPedidoUseCase interface {
	GerarTituloDeEntrada(ctx context.Context, input GerarTituloDeEntradaInput) (*domainreceber.Titulo, bool, error)
}
