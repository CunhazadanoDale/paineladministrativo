package pedido

import (
	"context"
	"strconv"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/pedido"
	"github.com/google/uuid"
)

type estoqueDoPedido struct {
	estoque  portsout.EstoqueRepository
	reservas portsout.ReservaRepository
	locais   portsout.LocalRepository
}

type encadeamento struct {
	estoque   *domainestoque.EstoqueProduto
	operacoes *[]*domainestoque.OperacaoEstoque
}

func (c encadeamento) aplicar(operacao *domainestoque.OperacaoEstoque, err error) (*domainestoque.OperacaoEstoque, error) {
	if err != nil {
		return nil, err
	}
	if err := c.estoque.Aplicar(operacao); err != nil {
		return nil, err
	}
	*c.operacoes = append(*c.operacoes, operacao)

	return operacao, nil
}

func (e *estoqueDoPedido) reservasPorItem(ctx context.Context, pedido *domainpedido.Pedido) (map[uuid.UUID][]*domainestoque.Reserva, error) {
	ids := make([]uuid.UUID, 0, len(pedido.Itens))
	for _, item := range pedido.Itens {
		ids = append(ids, item.ID)
	}

	reservas, err := e.reservas.AtivasPorOrigem(ctx, ids)
	if err != nil {
		return nil, err
	}

	porItem := make(map[uuid.UUID][]*domainestoque.Reserva, len(ids))
	for _, reserva := range reservas {
		if reserva.OrigemID != nil {
			porItem[*reserva.OrigemID] = append(porItem[*reserva.OrigemID], reserva)
		}
	}

	return porItem, nil
}

func (e *estoqueDoPedido) carregar(ctx context.Context, produtoID uuid.UUID, localIDs []uuid.UUID) (*domainestoque.EstoqueProduto, error) {
	estoque, err := e.estoque.Carregar(ctx, produtoID, localIDs)
	if err != nil {
		return nil, err
	}
	if estoque == nil {
		return nil, domain.ErroValidacao("produto do pedido não encontrado")
	}

	return estoque, nil
}

func (e *estoqueDoPedido) reservarFaltantes(
	ctx context.Context, pedido *domainpedido.Pedido, usuarioID uuid.UUID, agora time.Time,
) ([]*domainestoque.OperacaoEstoque, int, error) {
	locais, err := e.locais.AtivosDaEmpresa(ctx, pedido.EmpresaID)
	if err != nil {
		return nil, 0, err
	}
	if len(locais) == 0 {
		return nil, 0, nil
	}

	localIDs := make([]uuid.UUID, 0, len(locais))
	localPorID := make(map[uuid.UUID]*domainestoque.Local, len(locais))
	for _, local := range locais {
		localIDs = append(localIDs, local.ID)
		localPorID[local.ID] = local
	}

	reservasDoItem, err := e.reservasPorItem(ctx, pedido)
	if err != nil {
		return nil, 0, err
	}

	operacoes := make([]*domainestoque.OperacaoEstoque, 0)
	reservado := 0
	for _, item := range pedido.Itens {
		falta := item.FaltaCobrir(somaReservada(reservasDoItem[item.ID]))
		if falta == 0 {
			continue
		}

		estoque, err := e.carregar(ctx, item.ProdutoID, localIDs)
		if err != nil {
			return nil, 0, err
		}
		disponiveis := make([]domainpedido.DisponivelNoLocal, 0, len(localIDs))
		for _, localID := range localIDs {
			disponiveis = append(disponiveis, domainpedido.DisponivelNoLocal{
				LocalID:    localID,
				Disponivel: estoque.SaldoNoLocal(localID).Disponivel(),
			})
		}

		cadeia := encadeamento{estoque: estoque, operacoes: &operacoes}
		for _, planejada := range domainpedido.DistribuirReserva(falta, disponiveis) {
			itemID := item.ID
			if _, err := cadeia.aplicar(estoque.Reservar(domainestoque.ReservaInput{
				Local:      localPorID[planejada.LocalID],
				Quantidade: domainestoque.QuantidadeDe(planejada.Quantidade),
				Origem:     domainestoque.OrigemReservaPedidoItem,
				OrigemID:   &itemID,
				UsuarioID:  usuarioID,
				Observacao: referenciaDoPedido(pedido),
				Agora:      agora,
			})); err != nil {
				return nil, 0, err
			}
			reservado += planejada.Quantidade
		}
	}

	return operacoes, reservado, nil
}

func (e *estoqueDoPedido) liberarReservas(ctx context.Context, pedido *domainpedido.Pedido, agora time.Time) ([]*domainestoque.OperacaoEstoque, error) {
	reservasDoItem, err := e.reservasPorItem(ctx, pedido)
	if err != nil {
		return nil, err
	}

	operacoes := make([]*domainestoque.OperacaoEstoque, 0)
	for _, item := range pedido.Itens {
		reservas := reservasDoItem[item.ID]
		if len(reservas) == 0 {
			continue
		}

		estoque, err := e.carregar(ctx, item.ProdutoID, locaisDasReservas(reservas))
		if err != nil {
			return nil, err
		}
		cadeia := encadeamento{estoque: estoque, operacoes: &operacoes}
		for _, reserva := range reservas {
			if _, err := cadeia.aplicar(estoque.LiberarReserva(reserva, agora)); err != nil {
				return nil, err
			}
		}
	}

	return operacoes, nil
}

func (e *estoqueDoPedido) baixarEntrega(
	ctx context.Context, pedido *domainpedido.Pedido, entrega *domainpedido.Entrega, usuarioID uuid.UUID, agora time.Time,
) ([]*domainestoque.OperacaoEstoque, error) {
	localIDs := make([]uuid.UUID, 0, len(entrega.Itens))
	for _, item := range entrega.Itens {
		localIDs = append(localIDs, item.LocalID)
	}
	locais, err := e.locais.ObterVarios(ctx, localIDs)
	if err != nil {
		return nil, err
	}
	localPorID := make(map[uuid.UUID]*domainestoque.Local, len(locais))
	for _, local := range locais {
		localPorID[local.ID] = local
	}

	reservasDoItem, err := e.reservasPorItem(ctx, pedido)
	if err != nil {
		return nil, err
	}

	referencia := referenciaDoPedido(pedido)
	operacoes := make([]*domainestoque.OperacaoEstoque, 0, len(entrega.Itens))
	for _, itemEntrega := range entrega.Itens {
		item := pedido.Item(itemEntrega.PedidoItemID)
		local := localPorID[itemEntrega.LocalID]
		if item == nil {
			return nil, domain.ErroValidacao("item da entrega não pertence ao pedido")
		}
		if local == nil {
			return nil, domain.ErroValidacao("local de saída não encontrado")
		}

		doLocal := make([]*domainestoque.Reserva, 0)
		for _, reserva := range reservasDoItem[item.ID] {
			if reserva.LocalID == local.ID {
				doLocal = append(doLocal, reserva)
			}
		}
		plano, err := domainpedido.PlanejarBaixa(itemEntrega, doLocal)
		if err != nil {
			return nil, err
		}

		estoque, err := e.carregar(ctx, item.ProdutoID, []uuid.UUID{local.ID})
		if err != nil {
			return nil, err
		}
		cadeia := encadeamento{estoque: estoque, operacoes: &operacoes}

		saida := domainestoque.MovimentoLocalInput{
			Tipo:         domainestoque.TipoMovimentoSaida,
			Local:        local,
			Quantidade:   domainestoque.QuantidadeDe(itemEntrega.Quantidade),
			Reserva:      plano.BaixarReserva,
			UsuarioID:    usuarioID,
			DocumentoRef: &referencia,
			Observacao:   "entrega do " + referencia,
			Agora:        agora,
		}
		for _, reserva := range plano.Liberar {
			if _, err := cadeia.aplicar(estoque.LiberarReserva(reserva, agora)); err != nil {
				return nil, err
			}
		}
		movimento, err := cadeia.aplicar(estoque.Movimentar(saida))
		if err != nil {
			return nil, err
		}
		if plano.ReservarDeNovo > 0 {
			itemID := item.ID
			if _, err := cadeia.aplicar(estoque.Reservar(domainestoque.ReservaInput{
				Local:      local,
				Quantidade: domainestoque.QuantidadeDe(plano.ReservarDeNovo),
				Origem:     domainestoque.OrigemReservaPedidoItem,
				OrigemID:   &itemID,
				UsuarioID:  usuarioID,
				Observacao: referencia,
				Agora:      agora,
			})); err != nil {
				return nil, err
			}
		}

		movimentoID := movimento.Movimento.ID
		itemEntrega.MovimentoID = &movimentoID
	}

	return operacoes, nil
}

func somaReservada(reservas []*domainestoque.Reserva) int {
	total := 0
	for _, reserva := range reservas {
		total += reserva.Quantidade
	}

	return total
}

func locaisDasReservas(reservas []*domainestoque.Reserva) []uuid.UUID {
	vistos := make(map[uuid.UUID]bool, len(reservas))
	ids := make([]uuid.UUID, 0, len(reservas))
	for _, reserva := range reservas {
		if !vistos[reserva.LocalID] {
			vistos[reserva.LocalID] = true
			ids = append(ids, reserva.LocalID)
		}
	}

	return ids
}

func referenciaDoPedido(pedido *domainpedido.Pedido) string {
	return "pedido " + strconv.Itoa(pedido.Numero)
}
