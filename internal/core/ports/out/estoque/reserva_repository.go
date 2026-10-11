package estoque

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type ReservaFiltro struct {
	domain.PaginacaoFiltro
	ProdutoID  uuid.UUID
	LocalID    *uuid.UUID
	Status     domainestoque.StatusReserva
	EmpresaIDs []uuid.UUID
}

type ReservaRepository interface {
	Obter(ctx context.Context, id uuid.UUID) (*domainestoque.Reserva, error)
	Listar(ctx context.Context, filtro ReservaFiltro) ([]*domainestoque.Reserva, error)
}
