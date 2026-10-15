package pedido_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/pedido"
	"github.com/google/uuid"
)

func (c *cenario) sair(t *testing.T, entrega *domainpedido.Entrega) *domainpedido.Entrega {
	t.Helper()

	saiu, err := c.entregas.RegistrarSaida(c.ctx, portsin.TransicaoEntregaInput{UsuarioID: c.comercial, EntregaID: entrega.ID, Versao: entrega.Versao})
	if err != nil {
		t.Fatalf("RegistrarSaida falhou: %v", err)
	}

	return saiu
}

func TestCriarEntregaValida(t *testing.T) {
	c := novoCenario(t)
	c.entrada(t, c.cimento, c.galpao, 10)
	pedido := c.confirmado(t, c.dados())
	c.novaEntrega(t, pedido, c.galpao, 6)

	casos := []struct {
		nome      string
		usuario   uuid.UUID
		local     uuid.UUID
		quant     int
		frete     string
		sentinela error
	}{
		{"acima do ainda não programado", c.comercial, c.galpao.ID, 5, "fob", domain.ErrValidacao},
		{"local de outra empresa", c.comercial, c.galpaoRed.ID, 1, "fob", domain.ErrValidacao},
		{"local inexistente", c.comercial, uuid.New(), 1, "fob", domain.ErrValidacao},
		{"cif sem valor", c.comercial, c.galpao.ID, 1, "cif", domain.ErrValidacao},
		{"frete inválido", c.comercial, c.galpao.ID, 1, "cpt", domain.ErrValidacao},
		{"sem perfil comercial", c.operador, c.galpao.ID, 1, "fob", domain.ErrPermissao},
		{"outra empresa", c.comercialRed, c.galpao.ID, 1, "fob", domain.ErrPermissao},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			_, err := c.entregas.Criar(c.ctx, portsin.CriarEntregaInput{
				UsuarioID: caso.usuario,
				PedidoID:  pedido.ID,
				FreteTipo: caso.frete,
				Itens:     []portsin.EntregaItemInput{{PedidoItemID: pedido.Itens[0].ID, LocalID: caso.local, Quantidade: caso.quant}},
			})
			if !errors.Is(err, caso.sentinela) {
				t.Errorf("erro %v, esperado %v", err, caso.sentinela)
			}
		})
	}

	proposta := c.criar(t, c.comercial, c.mia, c.dados())
	if _, err := c.entregas.Criar(c.ctx, portsin.CriarEntregaInput{
		UsuarioID: c.comercial, PedidoID: proposta.ID, FreteTipo: "fob",
		Itens: []portsin.EntregaItemInput{{PedidoItemID: proposta.Itens[0].ID, LocalID: c.galpao.ID, Quantidade: 1}},
	}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("entrega de proposta = %v, esperado conflito", err)
	}
}

func TestSaidaParcialLiberaSaiEReservaDeNovo(t *testing.T) {
	c := novoCenario(t)
	c.entrada(t, c.cimento, c.galpao, 10)
	pedido := c.confirmado(t, c.dados())
	entrega := c.novaEntrega(t, pedido, c.galpao, 4)

	saiu := c.sair(t, entrega)

	if saiu.Status != domainpedido.StatusEntregaSaiu || saiu.Versao != 2 || saiu.SaiuEm == nil || saiu.Itens[0].MovimentoID == nil {
		t.Errorf("entrega = %q, versão %d, saiu em %v, movimento %v", saiu.Status, saiu.Versao, saiu.SaiuEm, saiu.Itens[0].MovimentoID)
	}
	salvo := c.pedido(pedido.ID)
	if salvo.Status != domainpedido.StatusParcialmenteEntregue || salvo.Itens[0].QuantidadeEntregue != 4 {
		t.Errorf("pedido = %q, entregue %d", salvo.Status, salvo.Itens[0].QuantidadeEntregue)
	}
	if reservas := c.reservasDoItem(pedido.Itens[0].ID); reservas[c.galpao.ID] != 6 || len(reservas) != 1 {
		t.Errorf("reservas depois da saída = %v, esperado 6 no galpão", reservas)
	}
	if saldo := c.saldo(c.cimento, c.galpao); saldo.Quantidade != 6 || saldo.Reservado != 6 {
		t.Errorf("galpão = %+v, esperado 6 com 6 reservados", saldo)
	}

	historico, _ := c.pedidos.ListarHistorico(c.ctx, pedido.ID, c.comercial)
	ultimo := historico[len(historico)-1]
	if ultimo.Acao != domainpedido.AcaoEntregaSaiu || ultimo.StatusAnterior == nil || *ultimo.StatusAnterior != domainpedido.StatusConfirmado ||
		ultimo.StatusNovo != domainpedido.StatusParcialmenteEntregue {
		t.Errorf("último histórico = %+v", ultimo)
	}

	if _, err := c.entregas.RegistrarSaida(c.ctx, portsin.TransicaoEntregaInput{UsuarioID: c.comercial, EntregaID: entrega.ID, Versao: 2}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("sair duas vezes = %v, esperado conflito", err)
	}
}

func TestSaidaComAReservaExataBaixaAReserva(t *testing.T) {
	c := novoCenario(t)
	c.entrada(t, c.cimento, c.galpao, 10)
	c.entrada(t, c.porcelanato, c.galpao, 4)
	pedido := c.confirmado(t, c.dados())

	entrega, err := c.entregas.Criar(c.ctx, portsin.CriarEntregaInput{
		UsuarioID: c.comercial,
		PedidoID:  pedido.ID,
		FreteTipo: "fob",
		Itens: []portsin.EntregaItemInput{
			{PedidoItemID: pedido.Itens[0].ID, LocalID: c.galpao.ID, Quantidade: 10},
			{PedidoItemID: pedido.Itens[1].ID, LocalID: c.galpao.ID, Quantidade: 4},
		},
	})
	if err != nil {
		t.Fatalf("Criar entrega falhou: %v", err)
	}

	c.sair(t, entrega)

	if salvo := c.pedido(pedido.ID); salvo.Status != domainpedido.StatusEntregue {
		t.Errorf("pedido = %q, esperado entregue", salvo.Status)
	}
	baixadas := 0
	for _, reserva := range c.m.reservas {
		if reserva.Status == domainestoque.StatusReservaBaixada {
			baixadas++
		}
		if reserva.Ativa() {
			t.Errorf("reserva ainda ativa: %+v", reserva)
		}
	}
	if baixadas != 2 {
		t.Errorf("reservas baixadas = %d, esperado 2", baixadas)
	}
	if saldo := c.saldo(c.cimento, c.galpao); saldo.Quantidade != 0 || saldo.Reservado != 0 {
		t.Errorf("galpão do cimento = %+v, esperado zerado", saldo)
	}
}

func TestSaidaSemReservaUsaODisponivelDoLocal(t *testing.T) {
	c := novoCenario(t)
	pedido := c.confirmado(t, c.dados())
	c.entrada(t, c.cimento, c.loja, 3)
	entrega := c.novaEntrega(t, pedido, c.loja, 2)

	c.sair(t, entrega)

	if saldo := c.saldo(c.cimento, c.loja); saldo.Quantidade != 1 || saldo.Reservado != 0 {
		t.Errorf("loja = %+v, esperado 1 livre", saldo)
	}

	semEstoque := c.novaEntrega(t, pedido, c.galpao, 2)
	if _, err := c.entregas.RegistrarSaida(c.ctx, portsin.TransicaoEntregaInput{UsuarioID: c.comercial, EntregaID: semEstoque.ID, Versao: 1}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("saída sem estoque = %v, esperado erro de validação", err)
	}
	if c.m.entregas[semEstoque.ID].Status != domainpedido.StatusEntregaPendente {
		t.Error("saída recusada mudou a entrega")
	}
}

func TestConcluirECancelarEntrega(t *testing.T) {
	c := novoCenario(t)
	c.entrada(t, c.cimento, c.galpao, 10)
	pedido := c.confirmado(t, c.dados())
	entrega := c.novaEntrega(t, pedido, c.galpao, 2)
	pendente := c.novaEntrega(t, pedido, c.galpao, 1)

	saiu := c.sair(t, entrega)
	if _, err := c.entregas.Concluir(c.ctx, portsin.TransicaoEntregaInput{UsuarioID: c.comercial, EntregaID: entrega.ID, Versao: 1}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("concluir com versão velha = %v, esperado conflito", err)
	}
	concluida, err := c.entregas.Concluir(c.ctx, portsin.TransicaoEntregaInput{UsuarioID: c.comercial, EntregaID: entrega.ID, Versao: saiu.Versao, Data: domainpedido.Hoje(time.Now())})
	if err != nil || concluida.Status != domainpedido.StatusEntregaEntregue || concluida.EntregueEm == nil {
		t.Fatalf("concluída = %+v (%v)", concluida, err)
	}

	cancelada, err := c.entregas.Cancelar(c.ctx, portsin.TransicaoEntregaInput{UsuarioID: c.comercial, EntregaID: pendente.ID, Versao: 1})
	if err != nil || cancelada.Status != domainpedido.StatusEntregaCancelada {
		t.Fatalf("cancelada = %+v (%v)", cancelada, err)
	}
	if _, err := c.entregas.Cancelar(c.ctx, portsin.TransicaoEntregaInput{UsuarioID: c.comercial, EntregaID: entrega.ID, Versao: concluida.Versao}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("cancelar entrega concluída = %v, esperado conflito", err)
	}
	if reservas := c.reservasDoItem(pedido.Itens[0].ID); reservas[c.galpao.ID] != 8 {
		t.Errorf("cancelar a entrega pendente mexeu nas reservas: %v", reservas)
	}
	if _, err := c.entregas.Obter(c.ctx, uuid.New(), c.comercial); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("entrega inexistente = %v, esperado não encontrado", err)
	}
}

func TestEntregasRespeitamEmpresaERepresentante(t *testing.T) {
	c := novoCenario(t)
	c.entrada(t, c.cimento, c.galpao, 20)
	doVendedor := c.criar(t, c.vendedor, c.mia, c.dadosDoVendedor())
	if _, err := c.pedidos.Confirmar(c.ctx, portsin.TransicaoPedidoInput{UsuarioID: c.vendedor, PedidoID: doVendedor.ID, Versao: 1}); err != nil {
		t.Fatalf("Confirmar falhou: %v", err)
	}
	entregaDoVendedor := c.novaEntrega(t, c.pedido(doVendedor.ID), c.galpao, 1)
	alheio := c.confirmado(t, c.dados())
	entregaAlheia := c.novaEntrega(t, alheio, c.galpao, 1)

	daRed := c.criar(t, c.comercialRed, c.red, c.dados())
	if _, err := c.pedidos.Confirmar(c.ctx, portsin.TransicaoPedidoInput{UsuarioID: c.comercialRed, PedidoID: daRed.ID, Versao: 1}); err != nil {
		t.Fatalf("Confirmar falhou: %v", err)
	}
	if _, err := c.entregas.Criar(c.ctx, portsin.CriarEntregaInput{
		UsuarioID: c.comercialRed, PedidoID: daRed.ID, FreteTipo: "fob",
		Itens: []portsin.EntregaItemInput{{PedidoItemID: daRed.Itens[0].ID, LocalID: c.galpaoRed.ID, Quantidade: 1}},
	}); err != nil {
		t.Fatalf("entrega da Red falhou: %v", err)
	}

	contar := func(usuario uuid.UUID) int {
		t.Helper()
		entregas, err := c.entregas.Listar(c.ctx, portsin.ListarEntregasInput{UsuarioID: usuario})
		if err != nil {
			t.Fatalf("Listar falhou: %v", err)
		}
		return len(entregas)
	}
	if total := contar(c.comercial); total != 2 {
		t.Errorf("entregas da MIA = %d, esperado 2", total)
	}
	if total := contar(c.comercialRed); total != 1 {
		t.Errorf("entregas da Red = %d, esperado 1", total)
	}
	if total := contar(c.vendedor); total != 1 {
		t.Errorf("entregas do vendedor = %d, esperado 1", total)
	}
	if total := contar(c.administrador); total != 3 {
		t.Errorf("entregas do administrador = %d, esperado 3", total)
	}

	if _, err := c.entregas.Obter(c.ctx, entregaAlheia.ID, c.vendedor); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("vendedor vendo entrega alheia = %v, esperado permissão", err)
	}
	if _, err := c.entregas.Obter(c.ctx, entregaDoVendedor.ID, c.comercialRed); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("Red vendo entrega da MIA = %v, esperado permissão", err)
	}
	if _, err := c.entregas.RegistrarSaida(c.ctx, portsin.TransicaoEntregaInput{UsuarioID: c.comercialRed, EntregaID: entregaAlheia.ID, Versao: 1}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("Red dando saída na MIA = %v, esperado permissão", err)
	}
	if _, err := c.entregas.ListarDoPedido(c.ctx, alheio.ID, c.comercialRed); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("Red listando entregas da MIA = %v, esperado permissão", err)
	}
	if _, err := c.entregas.Listar(c.ctx, portsin.ListarEntregasInput{UsuarioID: c.comercial, Status: "voando"}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("status inválido = %v, esperado erro de validação", err)
	}
}

func TestSaidaLiberaReservaQueSobraEmOutroLocal(t *testing.T) {
	c := novoCenario(t)
	c.entrada(t, c.cimento, c.galpao, 6)
	c.entrada(t, c.cimento, c.loja, 4)
	pedido := c.confirmado(t, c.dados())
	if reservas := c.reservasDoItem(pedido.Itens[0].ID); reservas[c.galpao.ID] != 6 || reservas[c.loja.ID] != 4 {
		t.Fatalf("reservas depois de confirmar = %v, esperado 6 no galpão e 4 na loja", reservas)
	}
	c.entrada(t, c.cimento, c.galpao, 4)

	c.sair(t, c.novaEntrega(t, c.pedido(pedido.ID), c.galpao, 10))

	if reservas := c.reservasDoItem(pedido.Itens[0].ID); len(reservas) != 0 {
		t.Errorf("reservas do item entregue por completo = %v, esperado nenhuma", reservas)
	}
	if loja := c.saldo(c.cimento, c.loja); loja.Quantidade != 4 || loja.Reservado != 0 {
		t.Errorf("loja = %+v, esperado 4 livres", loja)
	}
	if galpao := c.saldo(c.cimento, c.galpao); galpao.Quantidade != 0 || galpao.Reservado != 0 {
		t.Errorf("galpão = %+v, esperado zerado", galpao)
	}
	if item := c.pedido(pedido.ID).Itens[0]; item.QuantidadeEntregue != 10 {
		t.Errorf("entregue = %d, esperado 10", item.QuantidadeEntregue)
	}
}

func TestSaidaParcialReduzReservaDeOutroLocalAoRestante(t *testing.T) {
	c := novoCenario(t)
	c.entrada(t, c.cimento, c.galpao, 6)
	c.entrada(t, c.cimento, c.loja, 4)
	pedido := c.confirmado(t, c.dados())
	c.entrada(t, c.cimento, c.galpao, 2)

	c.sair(t, c.novaEntrega(t, c.pedido(pedido.ID), c.galpao, 8))

	reservas := c.reservasDoItem(pedido.Itens[0].ID)
	if reservas[c.loja.ID] != 2 || reservas[c.galpao.ID] != 0 {
		t.Errorf("reservas com 2 a entregar = %v, esperado só 2 na loja", reservas)
	}
}
