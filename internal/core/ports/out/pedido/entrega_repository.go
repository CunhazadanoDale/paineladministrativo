package pedido

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	"github.com/google/uuid"
)

type EntregaFiltro struct {
	domain.PaginacaoFiltro
	Empresa     *uuid.UUID
	EmpresaIDs  []uuid.UUID
	Status      domainpedido.StatusEntrega
	PrevisaoAte *time.Time
}

type EntregaRepository interface {
	Obter(ctx context.Context, id uuid.UUID) (*domainpedido.Entrega, error)
	ListarDoPedido(ctx context.Context, pedidoID uuid.UUID) ([]*domainpedido.Entrega, error)
	Listar(ctx context.Context, filtro EntregaFiltro) ([]*domainpedido.Entrega, error)
}
