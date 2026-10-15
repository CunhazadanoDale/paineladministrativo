package pedido

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	"github.com/google/uuid"
)

type ConsultarPendenciasInput struct {
	UsuarioID uuid.UUID
	Empresa   *uuid.UUID
	ClienteID *uuid.UUID
	ProdutoID *uuid.UUID
	Filtro    domain.PaginacaoFiltro
}

type ConsultarPrazosInput struct {
	UsuarioID uuid.UUID
	Empresa   *uuid.UUID
	Situacao  string
	Dias      *int
	Filtro    domain.PaginacaoFiltro
}

type ConsultarCancelamentosInput struct {
	UsuarioID    uuid.UUID
	Empresa      *uuid.UUID
	ComDevolucao bool
	De           *time.Time
	Ate          *time.Time
	Filtro       domain.PaginacaoFiltro
}

type ConsultaUseCase interface {
	Pendencias(ctx context.Context, input ConsultarPendenciasInput) ([]*domainpedido.Pendencia, error)
	Prazos(ctx context.Context, input ConsultarPrazosInput) ([]*domainpedido.Pedido, error)
	Cancelamentos(ctx context.Context, input ConsultarCancelamentosInput) ([]*domainpedido.Pedido, error)
}
