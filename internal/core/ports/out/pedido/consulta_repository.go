package pedido

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	"github.com/google/uuid"
)

type EscopoFiltro struct {
	Empresa         *uuid.UUID
	EmpresaIDs      []uuid.UUID
	RepresentanteID *uuid.UUID
}

type PendenciaFiltro struct {
	domain.PaginacaoFiltro
	EscopoFiltro
	ClienteID *uuid.UUID
	ProdutoID *uuid.UUID
}

type PrazoFiltro struct {
	domain.PaginacaoFiltro
	EscopoFiltro
	Janela domainpedido.JanelaDePrazo
}

type CancelamentoFiltro struct {
	domain.PaginacaoFiltro
	EscopoFiltro
	ComDevolucao bool
	De           *time.Time
	Ate          *time.Time
}

type ConsultaRepository interface {
	Pendencias(ctx context.Context, filtro PendenciaFiltro) ([]*domainpedido.Pendencia, error)
	PorPrazo(ctx context.Context, filtro PrazoFiltro) ([]*domainpedido.Pedido, error)
	Cancelamentos(ctx context.Context, filtro CancelamentoFiltro) ([]*domainpedido.Pedido, error)
}
