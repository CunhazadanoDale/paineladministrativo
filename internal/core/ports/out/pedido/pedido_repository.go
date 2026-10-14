package pedido

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	"github.com/google/uuid"
)

type EntregaAlterada struct {
	Entrega        *domainpedido.Entrega
	VersaoLida     int
	StatusAnterior domainpedido.StatusEntrega
}

type Gravacao struct {
	Pedido            *domainpedido.Pedido
	Novo              bool
	VersaoLida        int
	StatusAnterior    domainpedido.Status
	SubstituirItens   bool
	EntregasNovas     []*domainpedido.Entrega
	EntregasAlteradas []EntregaAlterada
	OperacoesEstoque  []*domainestoque.OperacaoEstoque
	Historicos        []*domainpedido.Historico
}

type PedidoFiltro struct {
	domain.PaginacaoFiltro
	Empresa         *uuid.UUID
	EmpresaIDs      []uuid.UUID
	ClienteID       *uuid.UUID
	RepresentanteID *uuid.UUID
	Status          domainpedido.Status
	Modelo          domainpedido.Modelo
	Numero          *int
}

type PedidoRepository interface {
	Gravar(ctx context.Context, gravacao *Gravacao) error
	Obter(ctx context.Context, id uuid.UUID) (*domainpedido.Pedido, error)
	Listar(ctx context.Context, filtro PedidoFiltro) ([]*domainpedido.Pedido, error)
	ListarHistorico(ctx context.Context, pedidoID uuid.UUID) ([]*domainpedido.Historico, error)
}
