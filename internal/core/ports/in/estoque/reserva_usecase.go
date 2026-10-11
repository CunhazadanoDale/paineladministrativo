package estoque

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type ReservarEstoqueInput struct {
	UsuarioID  uuid.UUID
	ProdutoID  uuid.UUID
	LocalID    uuid.UUID
	Quantidade int
	Observacao string
}

type ListarReservasInput struct {
	UsuarioID uuid.UUID
	ProdutoID uuid.UUID
	LocalID   *uuid.UUID
	Status    string
	Filtro    domain.PaginacaoFiltro
}

type ReservaUseCase interface {
	Reservar(ctx context.Context, input ReservarEstoqueInput) (*domainestoque.Reserva, error)
	Liberar(ctx context.Context, reservaID, usuarioID uuid.UUID) (*domainestoque.Reserva, error)
	Listar(ctx context.Context, input ListarReservasInput) ([]*domainestoque.Reserva, error)
}
