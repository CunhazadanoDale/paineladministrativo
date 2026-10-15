package pedido_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	"github.com/google/uuid"
)

func TestDistribuirReservaComecaPeloMaiorDisponivel(t *testing.T) {
	galpao := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	loja := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	deposito := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	vazio := uuid.MustParse("00000000-0000-0000-0000-000000000004")
	locais := []domainpedido.DisponivelNoLocal{
		{LocalID: loja, Disponivel: 5},
		{LocalID: vazio, Disponivel: 0},
		{LocalID: deposito, Disponivel: 8},
		{LocalID: galpao, Disponivel: 5},
	}

	planejadas := domainpedido.DistribuirReserva(15, locais)

	esperado := []domainpedido.ReservaPlanejada{
		{LocalID: deposito, Quantidade: 8},
		{LocalID: galpao, Quantidade: 5},
		{LocalID: loja, Quantidade: 2},
	}
	if len(planejadas) != len(esperado) {
		t.Fatalf("planejadas = %+v, esperado %+v", planejadas, esperado)
	}
	for indice := range esperado {
		if planejadas[indice] != esperado[indice] {
			t.Errorf("reserva %d = %+v, esperado %+v", indice, planejadas[indice], esperado[indice])
		}
	}
	if locais[0].LocalID != loja {
		t.Error("a distribuição reordenou a lista recebida")
	}
}

func TestDistribuirReservaSemFaltaOuSemDisponivel(t *testing.T) {
	locais := []domainpedido.DisponivelNoLocal{{LocalID: uuid.New(), Disponivel: 3}}

	if planejadas := domainpedido.DistribuirReserva(0, locais); len(planejadas) != 0 {
		t.Errorf("sem falta planejou %+v", planejadas)
	}
	if planejadas := domainpedido.DistribuirReserva(5, nil); len(planejadas) != 0 {
		t.Errorf("sem locais planejou %+v", planejadas)
	}
	if planejadas := domainpedido.DistribuirReserva(5, locais); len(planejadas) != 1 || planejadas[0].Quantidade != 3 {
		t.Errorf("cobertura parcial = %+v, esperado 3", planejadas)
	}
}

func TestFaltaCobrir(t *testing.T) {
	item := &domainpedido.Item{Quantidade: 10, QuantidadeEntregue: 3}

	if falta := item.FaltaCobrir(4); falta != 3 {
		t.Errorf("falta = %d, esperado 3", falta)
	}
	if falta := item.FaltaCobrir(9); falta != 0 {
		t.Errorf("falta com reserva sobrando = %d, esperado 0", falta)
	}
}

func reservaDoItem(item *domainpedido.EntregaItem, quantidade int) *domainestoque.Reserva {
	origemID := item.PedidoItemID

	return &domainestoque.Reserva{
		ID:         uuid.New(),
		LocalID:    item.LocalID,
		Quantidade: quantidade,
		Origem:     domainestoque.OrigemReservaPedidoItem,
		OrigemID:   &origemID,
		Status:     domainestoque.StatusReservaAtiva,
	}
}

func reservaNoLocal(item *domainpedido.EntregaItem, localID uuid.UUID, quantidade int) *domainestoque.Reserva {
	reserva := reservaDoItem(item, quantidade)
	reserva.LocalID = localID

	return reserva
}

func reservado(planejadas []domainpedido.ReservaPlanejada) map[uuid.UUID]int {
	porLocal := map[uuid.UUID]int{}
	for _, planejada := range planejadas {
		porLocal[planejada.LocalID] += planejada.Quantidade
	}

	return porLocal
}

func TestPlanejarBaixaNoLocalDeSaida(t *testing.T) {
	item := &domainpedido.EntregaItem{PedidoItemID: uuid.New(), LocalID: uuid.New(), Quantidade: 6}

	exata := reservaDoItem(item, 6)
	plano, err := domainpedido.PlanejarBaixa(item, 10, []*domainestoque.Reserva{exata})
	if err != nil || plano.BaixarReserva != exata || len(plano.Liberar) != 0 || plano.Sair != 0 || len(plano.Reservar) != 0 {
		t.Errorf("reserva exata = %+v, %v; esperado baixar a reserva", plano, err)
	}

	maior := reservaDoItem(item, 10)
	plano, err = domainpedido.PlanejarBaixa(item, 10, []*domainestoque.Reserva{maior})
	if err != nil || plano.BaixarReserva != nil || len(plano.Liberar) != 1 || plano.Sair != 6 || reservado(plano.Reservar)[item.LocalID] != 4 {
		t.Errorf("reserva maior = %+v, %v; esperado liberar, sair 6 e reservar 4", plano, err)
	}

	duas := []*domainestoque.Reserva{reservaDoItem(item, 4), reservaDoItem(item, 2)}
	plano, err = domainpedido.PlanejarBaixa(item, 10, duas)
	if err != nil || plano.BaixarReserva != nil || len(plano.Liberar) != 2 || plano.Sair != 6 || len(plano.Reservar) != 0 {
		t.Errorf("duas reservas = %+v, %v; esperado liberar as duas e sair 6", plano, err)
	}

	menor := reservaDoItem(item, 2)
	plano, err = domainpedido.PlanejarBaixa(item, 10, []*domainestoque.Reserva{menor})
	if err != nil || len(plano.Liberar) != 1 || plano.Sair != 6 || len(plano.Reservar) != 0 {
		t.Errorf("reserva menor = %+v, %v; esperado liberar e sair 6 do disponível", plano, err)
	}

	plano, err = domainpedido.PlanejarBaixa(item, 10, nil)
	if err != nil || len(plano.Liberar) != 0 || plano.Sair != 6 {
		t.Errorf("sem reserva = %+v, %v; esperado sair 6 do disponível", plano, err)
	}
}

func TestPlanejarBaixaNaoDeixaReservaSobrandoEmOutroLocal(t *testing.T) {
	saida, loja, deposito := uuid.New(), uuid.New(), uuid.New()
	item := &domainpedido.EntregaItem{PedidoItemID: uuid.New(), LocalID: saida, Quantidade: 5}

	naLoja := reservaNoLocal(item, loja, 2)
	plano, err := domainpedido.PlanejarBaixa(item, 5, []*domainestoque.Reserva{reservaDoItem(item, 3), naLoja})
	if err != nil || plano.Sair != 5 || len(plano.Liberar) != 2 || len(plano.Reservar) != 0 {
		t.Errorf("item entregue por completo = %+v, %v; esperado liberar as duas reservas e não reservar nada", plano, err)
	}

	exata := reservaDoItem(item, 5)
	plano, err = domainpedido.PlanejarBaixa(item, 6, []*domainestoque.Reserva{exata, reservaNoLocal(item, loja, 4)})
	if err != nil || plano.BaixarReserva != exata || len(plano.Liberar) != 1 || reservado(plano.Reservar)[loja] != 1 {
		t.Errorf("reserva exata com sobra na loja = %+v, %v; esperado baixar e reduzir a loja para 1", plano, err)
	}

	plano, err = domainpedido.PlanejarBaixa(item, 10, []*domainestoque.Reserva{
		reservaDoItem(item, 7), reservaNoLocal(item, loja, 2), reservaNoLocal(item, deposito, 4),
	})
	porLocal := reservado(plano.Reservar)
	if err != nil || plano.Sair != 5 || len(plano.Liberar) != 2 || porLocal[saida] != 0 || porLocal[deposito] != 3 || len(plano.Reservar) != 1 {
		t.Errorf("reservas além do restante = %+v, %v; esperado manter loja, reduzir depósito para 3 e não reservar no local de saída", plano, err)
	}

	plano, err = domainpedido.PlanejarBaixa(item, 10, []*domainestoque.Reserva{reservaDoItem(item, 8), reservaNoLocal(item, loja, 1)})
	if err != nil || reservado(plano.Reservar)[saida] != 3 || len(plano.Liberar) != 1 {
		t.Errorf("sobra cabendo no restante = %+v, %v; esperado reservar 3 de novo no local de saída", plano, err)
	}
	if locais := plano.Locais(); len(locais) != 1 || locais[0] != saida {
		t.Errorf("locais do plano = %v, esperado só o de saída", locais)
	}
}

func TestPlanejarBaixaValidaAsReservas(t *testing.T) {
	item := &domainpedido.EntregaItem{PedidoItemID: uuid.New(), LocalID: uuid.New(), Quantidade: 2}

	encerrada := reservaDoItem(item, 2)
	encerrada.Status = domainestoque.StatusReservaBaixada
	deOutroItem := reservaDoItem(item, 2)
	outro := uuid.New()
	deOutroItem.OrigemID = &outro
	manual := reservaDoItem(item, 2)
	manual.Origem = domainestoque.OrigemReservaManual

	casos := []struct {
		nome      string
		reserva   *domainestoque.Reserva
		sentinela error
	}{
		{"encerrada", encerrada, domain.ErrConflito},
		{"de outro item", deOutroItem, domain.ErrValidacao},
		{"manual", manual, domain.ErrValidacao},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			if _, err := domainpedido.PlanejarBaixa(item, 10, []*domainestoque.Reserva{caso.reserva}); !errors.Is(err, caso.sentinela) {
				t.Errorf("erro %v, esperado %v", err, caso.sentinela)
			}
		})
	}

	if _, err := domainpedido.PlanejarBaixa(nil, 10, nil); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("item nulo = %v, esperado erro de validação", err)
	}
	if _, err := domainpedido.PlanejarBaixa(item, 1, nil); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("quantidade acima do restante = %v, esperado erro de validação", err)
	}
}

func TestNovoHistoricoLimitaODetalhe(t *testing.T) {
	anterior := domainpedido.StatusProposta
	detalhe := ""
	for range 1100 {
		detalhe += "é"
	}

	historico := domainpedido.NovoHistorico(uuid.New(), uuid.New(), domainpedido.AcaoConfirmado, &anterior, domainpedido.StatusConfirmado, detalhe, agora)

	if len([]rune(historico.Detalhe)) != 1000 || historico.Acao != domainpedido.AcaoConfirmado || historico.CriadoEm != agora {
		t.Errorf("histórico com %d caracteres, ação %q", len([]rune(historico.Detalhe)), historico.Acao)
	}
}
