package pedido_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
)

func TestPendenciaCalculaFaltaEOQueDaParaCobrir(t *testing.T) {
	pendencia := domainpedido.Pendencia{Vendido: 10, Entregue: 3, Reservado: 4, DisponivelNaEmpresa: 2}

	if pendencia.Restante() != 7 || pendencia.FaltaCobrir() != 3 || pendencia.PodeCobrirAgora() != 2 {
		t.Errorf("restante %d, falta %d, pode cobrir %d; esperado 7, 3 e 2",
			pendencia.Restante(), pendencia.FaltaCobrir(), pendencia.PodeCobrirAgora())
	}

	coberta := domainpedido.Pendencia{Vendido: 5, Entregue: 1, Reservado: 6, DisponivelNaEmpresa: -2}
	if coberta.FaltaCobrir() != 0 || coberta.PodeCobrirAgora() != 0 {
		t.Errorf("pendência coberta: falta %d, pode cobrir %d", coberta.FaltaCobrir(), coberta.PodeCobrirAgora())
	}
}

func TestJanelaDePrazo(t *testing.T) {
	hoje := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	dez := 10

	vencido, err := domainpedido.NovaJanelaDePrazo(domainpedido.SituacaoPrazoVencido, nil, agora)
	if err != nil || vencido.De != nil || !vencido.Ate.Equal(hoje.AddDate(0, 0, -1)) {
		t.Errorf("janela de vencidos = %+v (%v)", vencido, err)
	}

	aVencer, err := domainpedido.NovaJanelaDePrazo(domainpedido.SituacaoPrazoAVencer, &dez, agora)
	if err != nil || aVencer.De == nil || !aVencer.De.Equal(hoje) || !aVencer.Ate.Equal(hoje.AddDate(0, 0, 10)) {
		t.Errorf("janela a vencer = %+v (%v)", aVencer, err)
	}

	todos, err := domainpedido.NovaJanelaDePrazo(domainpedido.SituacaoPrazoTodos, nil, agora)
	if err != nil || todos.De != nil || !todos.Ate.Equal(hoje.AddDate(0, 0, 30)) {
		t.Errorf("janela padrão = %+v (%v)", todos, err)
	}

	for _, dias := range []int{-1, 366} {
		if _, err := domainpedido.NovaJanelaDePrazo(domainpedido.SituacaoPrazoAVencer, &dias, agora); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("dias %d = %v, esperado erro de validação", dias, err)
		}
	}
	if _, err := domainpedido.NovaSituacaoPrazo("atrasado"); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("situação desconhecida = %v, esperado erro de validação", err)
	}
	if situacao, err := domainpedido.NovaSituacaoPrazo(" Vencido "); err != nil || situacao != domainpedido.SituacaoPrazoVencido {
		t.Errorf("situação = %q (%v), esperado vencido", situacao, err)
	}
}

func TestDiasParaOPrazo(t *testing.T) {
	pedido := novoPedidoDeTeste(t, domainpedido.ModeloProntaEntrega)
	if pedido.DiasParaOPrazo(agora) != nil {
		t.Error("pedido sem prazo devolveu dias")
	}

	prazo := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	pedido.PrazoLimite = &prazo
	if dias := pedido.DiasParaOPrazo(agora); dias == nil || *dias != -4 {
		t.Errorf("dias para o prazo = %v, esperado -4", dias)
	}
}
