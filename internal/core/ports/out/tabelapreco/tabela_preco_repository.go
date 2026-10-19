package tabelapreco

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	domaintabelapreco "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/tabelapreco"
	"github.com/google/uuid"
)

type ItemFiltro struct {
	domain.PaginacaoFiltro
	ProdutoID *uuid.UUID
	Modelo    domainpedido.Modelo
}

type VigenteFiltro struct {
	domain.PaginacaoFiltro
	Modelo    domainpedido.Modelo
	Data      time.Time
	ProdutoID *uuid.UUID
	Busca     string
}

type TabelaPrecoRepository interface {
	Criar(ctx context.Context, item *domaintabelapreco.Item) error
	Listar(ctx context.Context, filtro ItemFiltro) ([]*domaintabelapreco.Item, error)
	Vigentes(ctx context.Context, filtro VigenteFiltro) ([]*domaintabelapreco.PrecoVigente, error)
	RemoverFuturo(ctx context.Context, id uuid.UUID, hoje time.Time) error
}
