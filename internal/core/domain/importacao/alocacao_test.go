package importacao_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	"github.com/google/uuid"
)

func pedidoAlocavel(importacao *domainimportacao.Importacao, item *domainimportacao.Item, quantidade int) *domainimportacao.PedidoItemAlocavel {
	return &domainimportacao.PedidoItemAlocavel{
		PedidoItemID: uuid.New(), PedidoID: uuid.New(), PedidoNumero: 7, EmpresaID: importacao.EmpresaID,
		Modelo: domainpedido.ModeloImportacao, StatusPedido: domainpedido.StatusConfirmado,
		ProdutoID: item.ProdutoID, Quantidade: quantidade,
	}
}

func mapaDe(pedidos ...*domainimportacao.PedidoItemAlocavel) map[uuid.UUID]*domainimportacao.PedidoItemAlocavel {
	mapa := make(map[uuid.UUID]*domainimportacao.PedidoItemAlocavel, len(pedidos))
	for _, pedido := range pedidos {
		mapa[pedido.PedidoItemID] = pedido
	}

	return mapa
}

func TestDefinirAlocacoes(t *testing.T) {
	importacao := novaImportacaoDeTeste(t)
	porcelanato := importacao.Itens[0]
	pedido := pedidoAlocavel(importacao, porcelanato, 600)
	outro := pedidoAlocavel(importacao, porcelanato, 600)
	pedidos := mapaDe(pedido, outro)

	if err := importacao.DefinirAlocacoes(uuid.New(), []domainimportacao.AlocacaoInput{
		{ImportacaoItemID: porcelanato.ID, PedidoItemID: pedido.PedidoItemID, Quantidade: 600},
		{ImportacaoItemID: porcelanato.ID, PedidoItemID: outro.PedidoItemID, Quantidade: 400},
	}, pedidos, agora); err != nil {
		t.Fatalf("DefinirAlocacoes: %v", err)
	}
	if len(importacao.Alocacoes) != 2 || importacao.AlocadoNoItem(porcelanato.ID) != 1000 {
		t.Errorf("alocações = %+v", importacao.Alocacoes)
	}

	criada := importacao.Alocacoes[0].CriadoEm
	if err := importacao.DefinirAlocacoes(uuid.New(), []domainimportacao.AlocacaoInput{
		{ImportacaoItemID: porcelanato.ID, PedidoItemID: pedido.PedidoItemID, Quantidade: 500},
	}, pedidos, agora.Add(time.Hour)); err != nil {
		t.Fatalf("redefinir: %v", err)
	}
	if len(importacao.Alocacoes) != 1 || importacao.Alocacoes[0].Quantidade != 500 || !importacao.Alocacoes[0].CriadoEm.Equal(criada) {
		t.Errorf("alocações redefinidas = %+v", importacao.Alocacoes[0])
	}

	dados := dadosDeTeste()
	dados.FornecedorID = importacao.FornecedorID
	dados.Itens[0].ProdutoID = porcelanato.ProdutoID
	dados.Itens[0].Quantidade = 499
	if err := importacao.Alterar(dados, agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("reduzir o item abaixo do alocado: erro = %v", err)
	}
	dados.Itens[0].ProdutoID = uuid.New()
	if err := importacao.Alterar(dados, agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("remover item alocado: erro = %v", err)
	}
}

func TestDefinirAlocacoesValida(t *testing.T) {
	importacao := novaImportacaoDeTeste(t)
	porcelanato := importacao.Itens[0]

	deOutraEmpresa := pedidoAlocavel(importacao, porcelanato, 10)
	deOutraEmpresa.EmpresaID = uuid.New()
	deProntaEntrega := pedidoAlocavel(importacao, porcelanato, 10)
	deProntaEntrega.Modelo = domainpedido.ModeloProntaEntrega
	proposta := pedidoAlocavel(importacao, porcelanato, 10)
	proposta.StatusPedido = domainpedido.StatusProposta
	deOutroProduto := pedidoAlocavel(importacao, porcelanato, 10)
	deOutroProduto.ProdutoID = uuid.New()
	quaseEntregue := pedidoAlocavel(importacao, porcelanato, 10)
	quaseEntregue.QuantidadeEntregue = 6
	alocadoEmOutra := pedidoAlocavel(importacao, porcelanato, 10)
	alocadoEmOutra.AlocadoEmOutras = 8
	grande := pedidoAlocavel(importacao, porcelanato, 5000)
	pedidos := mapaDe(deOutraEmpresa, deProntaEntrega, proposta, deOutroProduto, quaseEntregue, alocadoEmOutra, grande)

	casos := []struct {
		nome    string
		entrada domainimportacao.AlocacaoInput
	}{
		{"outra empresa", domainimportacao.AlocacaoInput{ImportacaoItemID: porcelanato.ID, PedidoItemID: deOutraEmpresa.PedidoItemID, Quantidade: 1}},
		{"pronta entrega", domainimportacao.AlocacaoInput{ImportacaoItemID: porcelanato.ID, PedidoItemID: deProntaEntrega.PedidoItemID, Quantidade: 1}},
		{"proposta", domainimportacao.AlocacaoInput{ImportacaoItemID: porcelanato.ID, PedidoItemID: proposta.PedidoItemID, Quantidade: 1}},
		{"outro produto", domainimportacao.AlocacaoInput{ImportacaoItemID: porcelanato.ID, PedidoItemID: deOutroProduto.PedidoItemID, Quantidade: 1}},
		{"acima do restante", domainimportacao.AlocacaoInput{ImportacaoItemID: porcelanato.ID, PedidoItemID: quaseEntregue.PedidoItemID, Quantidade: 5}},
		{"acima com outras importações", domainimportacao.AlocacaoInput{ImportacaoItemID: porcelanato.ID, PedidoItemID: alocadoEmOutra.PedidoItemID, Quantidade: 3}},
		{"acima do item da importação", domainimportacao.AlocacaoInput{ImportacaoItemID: porcelanato.ID, PedidoItemID: grande.PedidoItemID, Quantidade: 1001}},
		{"quantidade zero", domainimportacao.AlocacaoInput{ImportacaoItemID: porcelanato.ID, PedidoItemID: grande.PedidoItemID, Quantidade: 0}},
		{"item de outra importação", domainimportacao.AlocacaoInput{ImportacaoItemID: uuid.New(), PedidoItemID: grande.PedidoItemID, Quantidade: 1}},
		{"pedido inexistente", domainimportacao.AlocacaoInput{ImportacaoItemID: porcelanato.ID, PedidoItemID: uuid.New(), Quantidade: 1}},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			err := importacao.DefinirAlocacoes(uuid.New(), []domainimportacao.AlocacaoInput{caso.entrada}, pedidos, agora)
			if !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("erro = %v", err)
			}
		})
	}

	repetida := domainimportacao.AlocacaoInput{ImportacaoItemID: porcelanato.ID, PedidoItemID: grande.PedidoItemID, Quantidade: 1}
	if err := importacao.DefinirAlocacoes(uuid.New(), []domainimportacao.AlocacaoInput{repetida, repetida}, pedidos, agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("repetida: erro = %v", err)
	}

	if err := importacao.Cancelar("desistência", agora); err != nil {
		t.Fatalf("Cancelar: %v", err)
	}
	if err := importacao.DefinirAlocacoes(uuid.New(), nil, pedidos, agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("alocar em importação cancelada: erro = %v", err)
	}
}

func TestPlanejarReservasRespeitaPrazoFaltaERecebido(t *testing.T) {
	importacao := desembaracada(t)
	porcelanato := importacao.Itens[0]

	cedo := dia(2026, 11, 1)
	tarde := dia(2026, 12, 1)
	urgente := pedidoAlocavel(importacao, porcelanato, 500)
	urgente.PrazoLimite = &cedo
	urgente.Reservado = 100
	depois := pedidoAlocavel(importacao, porcelanato, 500)
	depois.PrazoLimite = &tarde
	semPrazo := pedidoAlocavel(importacao, porcelanato, 500)
	pedidos := mapaDe(urgente, depois, semPrazo)

	if err := importacao.DefinirAlocacoes(uuid.New(), []domainimportacao.AlocacaoInput{
		{ImportacaoItemID: porcelanato.ID, PedidoItemID: semPrazo.PedidoItemID, Quantidade: 300},
		{ImportacaoItemID: porcelanato.ID, PedidoItemID: depois.PedidoItemID, Quantidade: 300},
		{ImportacaoItemID: porcelanato.ID, PedidoItemID: urgente.PedidoItemID, Quantidade: 400},
	}, pedidos, agora); err != nil {
		t.Fatalf("DefinirAlocacoes: %v", err)
	}

	cambio := int64(780_000)
	if _, err := importacao.Receber(domainimportacao.RecebimentoInput{
		RecebidaEm: agora, LocalID: uuid.New(), CambioMicro: &cambio,
		Recebidas: map[uuid.UUID]int{porcelanato.ID: 700},
	}, agora); err != nil {
		t.Fatalf("Receber: %v", err)
	}

	planejadas := importacao.PlanejarReservas(pedidos)
	if len(planejadas) != 2 {
		t.Fatalf("reservas planejadas = %+v", planejadas)
	}
	if planejadas[0].Alocacao.PedidoItemID != urgente.PedidoItemID || planejadas[0].Quantidade != 400 {
		t.Errorf("primeira reserva = %+v, esperado 400 do pedido mais urgente", planejadas[0])
	}
	if planejadas[1].Alocacao.PedidoItemID != depois.PedidoItemID || planejadas[1].Quantidade != 300 {
		t.Errorf("segunda reserva = %+v", planejadas[1])
	}
	for _, alocacao := range importacao.Alocacoes {
		if alocacao.PedidoItemID == semPrazo.PedidoItemID && (alocacao.Reservada == nil || *alocacao.Reservada != 0) {
			t.Errorf("pedido sem prazo ficou com %v reservado; o recebido acabou", alocacao.Reservada)
		}
	}
}

func TestPlanejarReservasIgnoraPedidoCancelado(t *testing.T) {
	importacao := desembaracada(t)
	porcelanato := importacao.Itens[0]
	pedido := pedidoAlocavel(importacao, porcelanato, 500)
	if err := importacao.DefinirAlocacoes(uuid.New(), []domainimportacao.AlocacaoInput{
		{ImportacaoItemID: porcelanato.ID, PedidoItemID: pedido.PedidoItemID, Quantidade: 300},
	}, mapaDe(pedido), agora); err != nil {
		t.Fatalf("DefinirAlocacoes: %v", err)
	}
	cambio := int64(780_000)
	if _, err := importacao.Receber(domainimportacao.RecebimentoInput{RecebidaEm: agora, LocalID: uuid.New(), CambioMicro: &cambio}, agora); err != nil {
		t.Fatalf("Receber: %v", err)
	}

	pedido.StatusPedido = domainpedido.StatusCancelado
	if planejadas := importacao.PlanejarReservas(mapaDe(pedido)); len(planejadas) != 0 {
		t.Errorf("reserva para pedido cancelado: %+v", planejadas)
	}
}
