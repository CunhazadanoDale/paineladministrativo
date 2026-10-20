package importacao

import (
	"fmt"
	"sort"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	"github.com/google/uuid"
)

type PedidoItemAlocavel struct {
	PedidoItemID       uuid.UUID
	PedidoID           uuid.UUID
	PedidoNumero       int
	EmpresaID          uuid.UUID
	Modelo             domainpedido.Modelo
	StatusPedido       domainpedido.Status
	PrazoLimite        *time.Time
	ProdutoID          uuid.UUID
	Quantidade         int
	QuantidadeEntregue int
	Reservado          int
	AlocadoEmOutras    int
}

func (p *PedidoItemAlocavel) Restante() int {
	return p.Quantidade - p.QuantidadeEntregue
}

func (p *PedidoItemAlocavel) FaltaCobrir() int {
	return max(p.Restante()-p.Reservado, 0)
}

type Alocacao struct {
	ImportacaoItemID uuid.UUID
	PedidoItemID     uuid.UUID
	Quantidade       int
	Reservada        *int
	ReservaID        *uuid.UUID
	UsuarioID        uuid.UUID
	CriadoEm         time.Time
	AtualizadoEm     time.Time
}

type AlocacaoInput struct {
	ImportacaoItemID uuid.UUID
	PedidoItemID     uuid.UUID
	Quantidade       int
}

type ReservaPlanejada struct {
	Alocacao   *Alocacao
	ProdutoID  uuid.UUID
	PedidoID   uuid.UUID
	Quantidade int
}

func (i *Importacao) DefinirAlocacoes(usuarioID uuid.UUID, entradas []AlocacaoInput, pedidos map[uuid.UUID]*PedidoItemAlocavel, agora time.Time) error {
	if err := i.ExigirAlocacoesAbertas(); err != nil {
		return err
	}

	atuais := make(map[[2]uuid.UUID]*Alocacao, len(i.Alocacoes))
	for _, alocacao := range i.Alocacoes {
		atuais[[2]uuid.UUID{alocacao.ImportacaoItemID, alocacao.PedidoItemID}] = alocacao
	}

	porItem := make(map[uuid.UUID]int)
	porPedidoItem := make(map[uuid.UUID]int)
	novas := make([]*Alocacao, 0, len(entradas))
	vistas := make(map[[2]uuid.UUID]bool, len(entradas))
	for _, entrada := range entradas {
		chave := [2]uuid.UUID{entrada.ImportacaoItemID, entrada.PedidoItemID}
		if vistas[chave] {
			return domain.ErroValidacao("alocação repetida; some as quantidades em uma linha")
		}
		vistas[chave] = true

		item := i.Item(entrada.ImportacaoItemID)
		if item == nil {
			return domain.ErroValidacao("item da alocação não pertence à importação")
		}
		pedido := pedidos[entrada.PedidoItemID]
		if err := conferirPedidoAlocavel(i, item, pedido); err != nil {
			return err
		}
		if entrada.Quantidade <= 0 || entrada.Quantidade > quantidadeMaxima {
			return domain.ErroValidacao("quantidade da alocação deve ser maior que zero")
		}

		porItem[item.ID] += entrada.Quantidade
		porPedidoItem[pedido.PedidoItemID] += entrada.Quantidade
		if porItem[item.ID] > item.Quantidade {
			return domain.ErroValidacao("as alocações passam da quantidade do item da importação")
		}
		if porPedidoItem[pedido.PedidoItemID]+pedido.AlocadoEmOutras > pedido.Restante() {
			return domain.ErroValidacao(fmt.Sprintf(
				"as alocações do item do pedido %d passam do que falta entregar (%d, já alocado em outras importações: %d)",
				pedido.PedidoNumero, pedido.Restante(), pedido.AlocadoEmOutras))
		}

		alocacao := &Alocacao{
			ImportacaoItemID: entrada.ImportacaoItemID,
			PedidoItemID:     entrada.PedidoItemID,
			Quantidade:       entrada.Quantidade,
			UsuarioID:        usuarioID,
			CriadoEm:         agora,
			AtualizadoEm:     agora,
		}
		if atual, ok := atuais[chave]; ok {
			alocacao.CriadoEm = atual.CriadoEm
		}
		novas = append(novas, alocacao)
	}

	i.Alocacoes = novas
	i.AtualizadoEm = agora

	return nil
}

func (i *Importacao) AlocadoNoItem(itemID uuid.UUID) int {
	total := 0
	for _, alocacao := range i.Alocacoes {
		if alocacao.ImportacaoItemID == itemID {
			total += alocacao.Quantidade
		}
	}

	return total
}

func (i *Importacao) PlanejarReservas(pedidos map[uuid.UUID]*PedidoItemAlocavel) []ReservaPlanejada {
	alocacoes := append([]*Alocacao(nil), i.Alocacoes...)
	sort.SliceStable(alocacoes, func(a, b int) bool {
		prazoA, prazoB := pedidos[alocacoes[a].PedidoItemID].PrazoLimite, pedidos[alocacoes[b].PedidoItemID].PrazoLimite
		switch {
		case prazoA != nil && prazoB == nil:
			return true
		case prazoA == nil && prazoB != nil:
			return false
		case prazoA != nil && !prazoA.Equal(*prazoB):
			return prazoA.Before(*prazoB)
		}
		return alocacoes[a].CriadoEm.Before(alocacoes[b].CriadoEm)
	})

	disponivel := make(map[uuid.UUID]int, len(i.Itens))
	for _, item := range i.Itens {
		if item.QuantidadeRecebida != nil {
			disponivel[item.ID] = *item.QuantidadeRecebida
		}
	}
	faltaCobrir := make(map[uuid.UUID]int, len(pedidos))
	for id, pedido := range pedidos {
		faltaCobrir[id] = pedido.FaltaCobrir()
		if !pedido.StatusPedido.EmAtendimento() {
			faltaCobrir[id] = 0
		}
	}

	planejadas := make([]ReservaPlanejada, 0, len(alocacoes))
	for _, alocacao := range alocacoes {
		quantidade := min(alocacao.Quantidade, faltaCobrir[alocacao.PedidoItemID], disponivel[alocacao.ImportacaoItemID])
		quantidade = max(quantidade, 0)
		reservada := quantidade
		alocacao.Reservada = &reservada
		if quantidade == 0 {
			continue
		}

		faltaCobrir[alocacao.PedidoItemID] -= quantidade
		disponivel[alocacao.ImportacaoItemID] -= quantidade
		pedido := pedidos[alocacao.PedidoItemID]
		planejadas = append(planejadas, ReservaPlanejada{
			Alocacao:   alocacao,
			ProdutoID:  pedido.ProdutoID,
			PedidoID:   pedido.PedidoID,
			Quantidade: quantidade,
		})
	}

	return planejadas
}

func conferirPedidoAlocavel(importacao *Importacao, item *Item, pedido *PedidoItemAlocavel) error {
	switch {
	case pedido == nil:
		return domain.ErroValidacao("item de pedido da alocação não encontrado")
	case pedido.EmpresaID != importacao.EmpresaID:
		return domain.ErroValidacao(fmt.Sprintf("o pedido %d é de outra empresa; estoque entre empresas não é transferido", pedido.PedidoNumero))
	case pedido.Modelo != domainpedido.ModeloImportacao:
		return domain.ErroValidacao(fmt.Sprintf("o pedido %d não é de importação", pedido.PedidoNumero))
	case !pedido.StatusPedido.EmAtendimento():
		return domain.ErroValidacao(fmt.Sprintf("o pedido %d não está confirmado nem em entrega", pedido.PedidoNumero))
	case pedido.ProdutoID != item.ProdutoID:
		return domain.ErroValidacao(fmt.Sprintf("o item do pedido %d é de outro produto", pedido.PedidoNumero))
	}

	return nil
}

func (i *Importacao) ExigirAlocacoesAbertas() error {
	if !i.Status.Editavel() && i.Status != StatusDesembaracada {
		return domain.ErroConflito("as alocações só mudam antes do recebimento e com a importação ativa")
	}

	return nil
}
