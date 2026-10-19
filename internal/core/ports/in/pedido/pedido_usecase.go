package pedido

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	"github.com/google/uuid"
)

type ItemInput struct {
	ProdutoID             uuid.UUID
	Quantidade            int
	PrecoUnitarioCentavos int64
}

type DadosPedidoInput struct {
	ClienteID             uuid.UUID
	ObraID                *uuid.UUID
	RepresentanteID       *uuid.UUID
	Modelo                string
	EntradaBP             *int
	PrazoBoletoDias       *int
	PrazoEntregaDias      *int
	Validade              *time.Time
	FreteTipo             string
	FreteEstimadoCentavos *int64
	PrazoLimite           *time.Time
	NumeroPedidoCliente   *string
	Observacoes           string
	Itens                 []ItemInput
}

type CriarPedidoInput struct {
	UsuarioID uuid.UUID
	EmpresaID uuid.UUID
	Dados     DadosPedidoInput
}

type AlterarPedidoInput struct {
	UsuarioID uuid.UUID
	PedidoID  uuid.UUID
	Versao    int
	Dados     DadosPedidoInput
}

type TransicaoPedidoInput struct {
	UsuarioID uuid.UUID
	PedidoID  uuid.UUID
	Versao    int
}

type RegistrarEntradaInput struct {
	UsuarioID     uuid.UUID
	PedidoID      uuid.UUID
	Versao        int
	PagaEm        time.Time
	ValorCentavos int64
}

type CancelarPedidoInput struct {
	UsuarioID              uuid.UUID
	PedidoID               uuid.UUID
	Versao                 int
	Motivo                 string
	ValorADevolverCentavos *int64
}

type ListarPedidosInput struct {
	UsuarioID       uuid.UUID
	Empresa         *uuid.UUID
	ClienteID       *uuid.UUID
	RepresentanteID *uuid.UUID
	Status          string
	Modelo          string
	Numero          *int
	Filtro          domain.PaginacaoFiltro
}

type PedidoUseCase interface {
	Criar(ctx context.Context, input CriarPedidoInput) (*domainpedido.Pedido, error)
	Obter(ctx context.Context, pedidoID, usuarioID uuid.UUID) (*domainpedido.Pedido, error)
	Listar(ctx context.Context, input ListarPedidosInput) ([]*domainpedido.Pedido, error)
	Alterar(ctx context.Context, input AlterarPedidoInput) (*domainpedido.Pedido, error)
	Confirmar(ctx context.Context, input TransicaoPedidoInput) (*domainpedido.Pedido, error)
	ReservarPendencias(ctx context.Context, input TransicaoPedidoInput) (*domainpedido.Pedido, error)
	RegistrarEntrada(ctx context.Context, input RegistrarEntradaInput) (*domainpedido.Pedido, error)
	Cancelar(ctx context.Context, input CancelarPedidoInput) (*domainpedido.Pedido, error)
	ObterComReservas(ctx context.Context, pedidoID, usuarioID uuid.UUID) (*domainpedido.Pedido, []*domainestoque.Reserva, error)
	Documento(ctx context.Context, pedidoID, usuarioID uuid.UUID) (*domainpedido.DocumentoProposta, error)
	ModelosDeVenda(ctx context.Context) []domainpedido.ModeloDeVenda
	ListarHistorico(ctx context.Context, pedidoID, usuarioID uuid.UUID) ([]*domainpedido.Historico, error)
}
