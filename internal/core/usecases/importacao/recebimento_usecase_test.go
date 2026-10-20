package importacao_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/importacao"
	"github.com/google/uuid"
)

func (c *cenario) alocar(t *testing.T, usuarioID uuid.UUID, importacao *domainimportacao.Importacao, alocacoes ...portsin.AlocacaoInput) *domainimportacao.Importacao {
	t.Helper()

	visao, err := c.recebimento.DefinirAlocacoes(c.ctx, portsin.DefinirAlocacoesInput{
		UsuarioID: usuarioID, ImportacaoID: importacao.ID, Versao: importacao.Versao, Alocacoes: alocacoes,
	})
	if err != nil {
		t.Fatalf("DefinirAlocacoes: %v", err)
	}

	return visao.Importacao
}

func TestComercialAlocaPedidosDaImportacao(t *testing.T) {
	c := novoCenario(t)
	importacao := c.criar(t, c.financeiro, c.mia, c.dados())
	pedido := c.novoItemDePedido(c.mia, c.porcelanato, 500, nil)

	alocada := c.alocar(t, c.comercial, importacao, portsin.AlocacaoInput{
		ImportacaoItemID: importacao.Itens[0].ID, PedidoItemID: pedido.PedidoItemID, Quantidade: 300,
	})
	if len(alocada.Alocacoes) != 1 || alocada.Alocacoes[0].Quantidade != 300 || alocada.Versao != 2 {
		t.Errorf("importação alocada = %+v", alocada)
	}

	outra := c.criar(t, c.financeiro, c.mia, c.dados())
	_, err := c.recebimento.DefinirAlocacoes(c.ctx, portsin.DefinirAlocacoesInput{
		UsuarioID: c.comercial, ImportacaoID: outra.ID, Versao: 1,
		Alocacoes: []portsin.AlocacaoInput{{ImportacaoItemID: outra.Itens[0].ID, PedidoItemID: pedido.PedidoItemID, Quantidade: 201}},
	})
	if !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("alocar além do restante somando outra importação: erro = %v", err)
	}

	daRed := c.novoItemDePedido(c.red, c.porcelanato, 10, nil)
	_, err = c.recebimento.DefinirAlocacoes(c.ctx, portsin.DefinirAlocacoesInput{
		UsuarioID: c.administrador, ImportacaoID: outra.ID, Versao: 1,
		Alocacoes: []portsin.AlocacaoInput{{ImportacaoItemID: outra.Itens[0].ID, PedidoItemID: daRed.PedidoItemID, Quantidade: 1}},
	})
	if !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("alocar pedido de outra empresa: erro = %v", err)
	}

	if _, err := c.recebimento.DefinirAlocacoes(c.ctx, portsin.DefinirAlocacoesInput{
		UsuarioID: c.operador, ImportacaoID: outra.ID, Versao: 1,
	}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("operador: erro = %v", err)
	}
}

func TestReceberDaEntradaComCustoEReservaParaOsPedidos(t *testing.T) {
	c := novoCenario(t)
	prazo := domainimportacao.Hoje(time.Now()).AddDate(0, 1, 0)
	pedido := c.novoItemDePedido(c.mia, c.porcelanato, 500, &prazo)
	importacao := c.criar(t, c.financeiro, c.mia, c.dados())
	importacao = c.alocar(t, c.comercial, importacao, portsin.AlocacaoInput{
		ImportacaoItemID: importacao.Itens[0].ID, PedidoItemID: pedido.PedidoItemID, Quantidade: 500,
	})
	importacao = c.desembaracar(t, importacao)

	cambio := int64(780_000)
	visao, err := c.recebimento.Receber(c.ctx, portsin.ReceberInput{
		UsuarioID: c.financeiro, ImportacaoID: importacao.ID, Versao: importacao.Versao,
		RecebidaEm: domainimportacao.Hoje(time.Now()), LocalID: c.galpao.ID, CambioMicro: &cambio,
		Recebidas: map[uuid.UUID]int{importacao.Itens[1].ID: 0},
	})
	if err != nil {
		t.Fatalf("Receber: %v", err)
	}
	recebida := visao.Importacao
	porcelanato, revestimento := recebida.Itens[0], recebida.Itens[1]
	if recebida.Status != domainimportacao.StatusRecebida || *porcelanato.QuantidadeRecebida != 800 || *porcelanato.CustoUnitarioCentavos != 2340 ||
		porcelanato.MovimentoID == nil || *revestimento.QuantidadeRecebida != 0 || revestimento.MovimentoID != nil {
		t.Fatalf("importação recebida = %+v; itens %+v e %+v", recebida, porcelanato, revestimento)
	}

	estoque := c.m.estoques[c.porcelanato.ID]
	saldo := estoque.SaldoNoLocal(c.galpao.ID)
	if saldo.Quantidade != 800 || saldo.Reservado != 500 || len(estoque.Lotes) != 1 || estoque.Lotes[0].Custo.Centavos() != 2340 {
		t.Errorf("estoque = saldo %+v, lotes %+v", saldo, estoque.Lotes)
	}
	if len(c.m.reservas) != 1 {
		t.Fatalf("reservas = %+v", c.m.reservas)
	}
	for _, reserva := range c.m.reservas {
		if reserva.Origem != domainestoque.OrigemReservaPedidoItem || *reserva.OrigemID != pedido.PedidoItemID || reserva.Quantidade != 500 {
			t.Errorf("reserva = %+v", reserva)
		}
		if *recebida.Alocacoes[0].ReservaID != reserva.ID || *recebida.Alocacoes[0].Reservada != 500 {
			t.Errorf("alocação = %+v", recebida.Alocacoes[0])
		}
	}
	if len(c.m.historicosDePedido) != 1 || c.m.historicosDePedido[0].PedidoID != pedido.PedidoID ||
		c.m.historicosDePedido[0].Acao != domainpedido.AcaoPendenciasReservadas {
		t.Errorf("histórico do pedido = %+v", c.m.historicosDePedido)
	}
	if _, err := c.recebimento.DefinirAlocacoes(c.ctx, portsin.DefinirAlocacoesInput{
		UsuarioID: c.financeiro, ImportacaoID: recebida.ID, Versao: recebida.Versao,
	}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("alocar depois do recebimento: erro = %v", err)
	}
}

func TestReceberSoReservaOQueFaltaCobrir(t *testing.T) {
	c := novoCenario(t)
	pedido := c.novoItemDePedido(c.mia, c.porcelanato, 500, nil)
	importacao := c.criar(t, c.financeiro, c.mia, c.dados())
	importacao = c.alocar(t, c.financeiro, importacao, portsin.AlocacaoInput{
		ImportacaoItemID: importacao.Itens[0].ID, PedidoItemID: pedido.PedidoItemID, Quantidade: 500,
	})
	importacao = c.desembaracar(t, importacao)
	c.m.reservas[uuid.New()] = &domainestoque.Reserva{
		ID: uuid.New(), Quantidade: 450, Origem: domainestoque.OrigemReservaPedidoItem, OrigemID: &pedido.PedidoItemID,
		Status: domainestoque.StatusReservaAtiva,
	}

	cambio := int64(780_000)
	visao, err := c.recebimento.Receber(c.ctx, portsin.ReceberInput{
		UsuarioID: c.financeiro, ImportacaoID: importacao.ID, Versao: importacao.Versao,
		RecebidaEm: domainimportacao.Hoje(time.Now()), LocalID: c.galpao.ID, CambioMicro: &cambio,
	})
	if err != nil {
		t.Fatalf("Receber: %v", err)
	}
	if reservada := visao.Importacao.Alocacoes[0].Reservada; reservada == nil || *reservada != 50 {
		t.Errorf("reservado pela importação = %v, esperado 50", reservada)
	}
}

func TestReceberValidaLocalEPerfil(t *testing.T) {
	c := novoCenario(t)
	importacao := c.desembaracar(t, c.criar(t, c.financeiro, c.mia, c.dados()))
	cambio := int64(780_000)

	casos := []struct {
		nome      string
		usuario   uuid.UUID
		local     uuid.UUID
		sentinela error
	}{
		{"comercial", c.comercial, c.galpao.ID, domain.ErrPermissao},
		{"local de outra empresa", c.financeiro, c.galpaoRed.ID, domain.ErrValidacao},
		{"local inativo", c.financeiro, c.fechado.ID, domain.ErrValidacao},
		{"local inexistente", c.financeiro, uuid.New(), domain.ErrValidacao},
		{"sem local", c.financeiro, uuid.Nil, domain.ErrValidacao},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			_, err := c.recebimento.Receber(c.ctx, portsin.ReceberInput{
				UsuarioID: caso.usuario, ImportacaoID: importacao.ID, Versao: importacao.Versao,
				RecebidaEm: domainimportacao.Hoje(time.Now()), LocalID: caso.local, CambioMicro: &cambio,
			})
			if !errors.Is(err, caso.sentinela) {
				t.Errorf("erro = %v, esperado %v", err, caso.sentinela)
			}
		})
	}
	if len(c.m.reservas) != 0 || c.m.importacoes[importacao.ID].Status != domainimportacao.StatusDesembaracada {
		t.Error("recebimento recusado gravou alguma coisa")
	}
}

func TestReceberRecusaQuandoOPedidoSaiDeAtendimento(t *testing.T) {
	c := novoCenario(t)
	pedido := c.novoItemDePedido(c.mia, c.porcelanato, 500, nil)
	importacao := c.criar(t, c.financeiro, c.mia, c.dados())
	importacao = c.alocar(t, c.financeiro, importacao, portsin.AlocacaoInput{
		ImportacaoItemID: importacao.Itens[0].ID, PedidoItemID: pedido.PedidoItemID, Quantidade: 100,
	})
	importacao = c.desembaracar(t, importacao)
	c.m.pedidoItens[pedido.PedidoItemID].StatusPedido = domainpedido.StatusCancelado

	cambio := int64(780_000)
	visao, err := c.recebimento.Receber(c.ctx, portsin.ReceberInput{
		UsuarioID: c.financeiro, ImportacaoID: importacao.ID, Versao: importacao.Versao,
		RecebidaEm: domainimportacao.Hoje(time.Now()), LocalID: c.galpao.ID, CambioMicro: &cambio,
	})
	if err != nil {
		t.Fatalf("Receber com pedido cancelado: %v", err)
	}
	if len(c.m.reservas) != 0 || *visao.Importacao.Alocacoes[0].Reservada != 0 {
		t.Errorf("reservou para pedido cancelado: %+v", c.m.reservas)
	}
}
