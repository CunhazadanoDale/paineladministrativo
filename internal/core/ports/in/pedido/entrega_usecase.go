package pedido

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	"github.com/google/uuid"
)

type EntregaItemInput struct {
	PedidoItemID uuid.UUID
	LocalID      uuid.UUID
	Quantidade   int
}

type CriarEntregaInput struct {
	UsuarioID      uuid.UUID
	PedidoID       uuid.UUID
	Transportadora *string
	FreteTipo      string
	FreteCentavos  *int64
	Previsao       *time.Time
	Itens          []EntregaItemInput
}

type TransicaoEntregaInput struct {
	UsuarioID uuid.UUID
	EntregaID uuid.UUID
	Versao    int
	Data      time.Time
}

type ListarEntregasInput struct {
	UsuarioID   uuid.UUID
	Empresa     *uuid.UUID
	Status      string
	PrevisaoAte *time.Time
	Filtro      domain.PaginacaoFiltro
}

type EntregaUseCase interface {
	Criar(ctx context.Context, input CriarEntregaInput) (*domainpedido.Entrega, error)
	Obter(ctx context.Context, entregaID, usuarioID uuid.UUID) (*domainpedido.Entrega, error)
	ListarDoPedido(ctx context.Context, pedidoID, usuarioID uuid.UUID) ([]*domainpedido.Entrega, error)
	Listar(ctx context.Context, input ListarEntregasInput) ([]*domainpedido.Entrega, error)
	RegistrarSaida(ctx context.Context, input TransicaoEntregaInput) (*domainpedido.Entrega, error)
	Concluir(ctx context.Context, input TransicaoEntregaInput) (*domainpedido.Entrega, error)
	Cancelar(ctx context.Context, input TransicaoEntregaInput) (*domainpedido.Entrega, error)
}
