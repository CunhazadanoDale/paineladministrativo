package estoque_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

func (c *cenarioEstoque) reservar(local *domainestoque.Local, quantidade int) (*domainestoque.OperacaoEstoque, error) {
	return c.estoque.Reservar(domainestoque.ReservaInput{
		Local:      local,
		Quantidade: domainestoque.QuantidadeDe(quantidade),
		UsuarioID:  c.usuario,
		Observacao: "  cliente Construtora Alfa  ",
		Agora:      c.inicio.AddDate(0, 1, 0),
	})
}

func (c *cenarioEstoque) aplicar(operacao *domainestoque.OperacaoEstoque) {
	for _, saldo := range operacao.Saldos {
		c.estoque.Saldos[saldo.LocalID] = saldo
	}
	c.estoque.Produto.Saldo = operacao.Saldo
	c.estoque.Produto.Reservado = operacao.Reservado
}

func TestReservarSeparaODisponivelSemMudarOSaldo(t *testing.T) {
	c := novoCenarioEstoque(t)
	c.comLote(c.galpao, 10, centavos(100), 0)

	operacao, err := c.reservar(c.galpao, 4)
	if err != nil {
		t.Fatalf("reservar falhou: %v", err)
	}

	if operacao.Movimento != nil || len(operacao.LotesCriados) != 0 || operacao.VersaoLida != 7 {
		t.Errorf("reserva gerou movimento %v, lotes %d e versão %d", operacao.Movimento, len(operacao.LotesCriados), operacao.VersaoLida)
	}
	if operacao.Saldo.Quantidade() != 10 || operacao.Reservado.Quantidade() != 4 {
		t.Errorf("produto com saldo %d e reservado %d, esperado 10 e 4", operacao.Saldo.Quantidade(), operacao.Reservado.Quantidade())
	}
	if saldo := saldoDaOperacao(t, operacao, c.galpao.ID); saldo.Quantidade != 10 || saldo.Reservado != 4 || saldo.Disponivel() != 6 {
		t.Errorf("saldo do galpão = %+v, esperado 10 com 4 reservados", saldo)
	}

	reserva := operacao.ReservaCriada
	if reserva == nil || reserva.Status != domainestoque.StatusReservaAtiva || reserva.Origem != domainestoque.OrigemReservaManual ||
		reserva.Quantidade != 4 || reserva.LocalID != c.galpao.ID || reserva.Observacao != "cliente Construtora Alfa" || reserva.EncerradaEm != nil {
		t.Errorf("reserva criada = %+v", reserva)
	}
}

func TestReservarValidaEntradaEDisponivel(t *testing.T) {
	c := novoCenarioEstoque(t)
	c.comLote(c.galpao, 5, centavos(100), 0)
	c.aplicar(mustReservar(t, c, c.galpao, 3))

	casos := []struct {
		nome  string
		input domainestoque.ReservaInput
	}{
		{"sem local", domainestoque.ReservaInput{Quantidade: domainestoque.QuantidadeDe(1), UsuarioID: c.usuario}},
		{"sem usuário", domainestoque.ReservaInput{Local: c.galpao, Quantidade: domainestoque.QuantidadeDe(1)}},
		{"quantidade negativa", domainestoque.ReservaInput{Local: c.galpao, Quantidade: domainestoque.QuantidadeDe(-1), UsuarioID: c.usuario}},
		{"acima do disponível", domainestoque.ReservaInput{Local: c.galpao, Quantidade: domainestoque.QuantidadeDe(3), UsuarioID: c.usuario}},
		{"local sem saldo", domainestoque.ReservaInput{Local: c.loja, Quantidade: domainestoque.QuantidadeDe(1), UsuarioID: c.usuario}},
		{"observação longa", domainestoque.ReservaInput{Local: c.galpao, Quantidade: domainestoque.QuantidadeDe(1), UsuarioID: c.usuario, Observacao: strings.Repeat("a", 1001)}},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			if _, err := c.estoque.Reservar(caso.input); !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("erro %v, esperado erro de validação", err)
			}
		})
	}
}

func mustReservar(t *testing.T, c *cenarioEstoque, local *domainestoque.Local, quantidade int) *domainestoque.OperacaoEstoque {
	t.Helper()

	operacao, err := c.reservar(local, quantidade)
	if err != nil {
		t.Fatalf("reservar falhou: %v", err)
	}

	return operacao
}

func TestSaidaComReservaBaixaReservaELotes(t *testing.T) {
	c := novoCenarioEstoque(t)
	c.comLote(c.galpao, 10, centavos(100), 0)
	reservou := mustReservar(t, c, c.galpao, 8)
	c.aplicar(reservou)
	reserva := reservou.ReservaCriada

	if _, err := c.movimentar(domainestoque.TipoMovimentoSaida, c.galpao, 3, nil); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("saída sem reserva acima do disponível = %v, esperado erro de validação", err)
	}

	baixar := func(tipo domainestoque.TipoMovimento, local *domainestoque.Local, quantidade int, alvo *domainestoque.Reserva) (*domainestoque.OperacaoEstoque, error) {
		return c.estoque.Movimentar(domainestoque.MovimentoLocalInput{
			Tipo:       tipo,
			Local:      local,
			Quantidade: domainestoque.QuantidadeDe(quantidade),
			Reserva:    alvo,
			UsuarioID:  c.usuario,
			Agora:      c.inicio.AddDate(0, 2, 0),
		})
	}

	outra := *reserva
	outra.ProdutoID = uuid.New()
	encerrada := *reserva
	encerrada.Status = domainestoque.StatusReservaLiberada
	casos := []struct {
		nome       string
		tipo       domainestoque.TipoMovimento
		local      *domainestoque.Local
		quantidade int
		alvo       *domainestoque.Reserva
		sentinela  error
	}{
		{"perda com reserva", domainestoque.TipoMovimentoPerda, c.galpao, 8, reserva, domain.ErrValidacao},
		{"quantidade diferente", domainestoque.TipoMovimentoSaida, c.galpao, 5, reserva, domain.ErrValidacao},
		{"outro local", domainestoque.TipoMovimentoSaida, c.loja, 8, reserva, domain.ErrValidacao},
		{"outro produto", domainestoque.TipoMovimentoSaida, c.galpao, 8, &outra, domain.ErrValidacao},
		{"reserva encerrada", domainestoque.TipoMovimentoSaida, c.galpao, 8, &encerrada, domain.ErrConflito},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			if _, err := baixar(caso.tipo, caso.local, caso.quantidade, caso.alvo); !errors.Is(err, caso.sentinela) {
				t.Errorf("erro %v, esperado %v", err, caso.sentinela)
			}
		})
	}

	operacao, err := baixar(domainestoque.TipoMovimentoSaida, c.galpao, 8, reserva)
	if err != nil {
		t.Fatalf("saída com reserva falhou: %v", err)
	}
	if operacao.Saldo.Quantidade() != 2 || operacao.Reservado.Quantidade() != 0 {
		t.Errorf("produto com saldo %d e reservado %d, esperado 2 e 0", operacao.Saldo.Quantidade(), operacao.Reservado.Quantidade())
	}
	if saldo := saldoDaOperacao(t, operacao, c.galpao.ID); saldo.Quantidade != 2 || saldo.Reservado != 0 {
		t.Errorf("saldo do galpão = %+v, esperado 2 sem reserva", saldo)
	}
	baixada := operacao.ReservaEncerrada
	if baixada == nil || baixada.Status != domainestoque.StatusReservaBaixada || baixada.EncerradaEm == nil || baixada.ID != reserva.ID {
		t.Errorf("reserva baixada = %+v", baixada)
	}
	if operacao.Movimento.ReservaID == nil || *operacao.Movimento.ReservaID != reserva.ID || len(operacao.Movimento.Consumos) != 1 {
		t.Errorf("movimento da baixa = %+v", operacao.Movimento)
	}
	if !reserva.Ativa() {
		t.Error("o cálculo não deve alterar a reserva carregada")
	}
}

func TestLiberarReservaDevolveODisponivel(t *testing.T) {
	c := novoCenarioEstoque(t)
	c.comLote(c.galpao, 10, centavos(100), 0)
	reservou := mustReservar(t, c, c.galpao, 6)
	c.aplicar(reservou)

	operacao, err := c.estoque.LiberarReserva(reservou.ReservaCriada, c.inicio.AddDate(0, 2, 0))
	if err != nil {
		t.Fatalf("liberar falhou: %v", err)
	}
	if operacao.Reservado.Quantidade() != 0 || operacao.Saldo.Quantidade() != 10 || operacao.Movimento != nil {
		t.Errorf("liberação: reservado %d, saldo %d, movimento %v", operacao.Reservado.Quantidade(), operacao.Saldo.Quantidade(), operacao.Movimento)
	}
	if saldo := saldoDaOperacao(t, operacao, c.galpao.ID); saldo.Reservado != 0 || saldo.Disponivel() != 10 {
		t.Errorf("saldo do galpão = %+v, esperado tudo disponível", saldo)
	}
	if liberada := operacao.ReservaEncerrada; liberada.Status != domainestoque.StatusReservaLiberada || liberada.EncerradaEm == nil {
		t.Errorf("reserva liberada = %+v", liberada)
	}

	if _, err := c.estoque.LiberarReserva(operacao.ReservaEncerrada, c.inicio.AddDate(0, 3, 0)); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("liberar reserva encerrada = %v, esperado conflito", err)
	}
	if _, err := c.estoque.LiberarReserva(nil, c.inicio); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("liberar sem reserva = %v, esperado erro de validação", err)
	}
}

func TestStatusDeReservaAceitaSoOsConhecidos(t *testing.T) {
	if status, err := domainestoque.NovoStatusReserva(" ATIVA "); err != nil || status != domainestoque.StatusReservaAtiva {
		t.Errorf("NovoStatusReserva(ATIVA) = %q, %v", status, err)
	}
	if _, err := domainestoque.NovoStatusReserva("cancelada"); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("status desconhecido = %v, esperado erro de validação", err)
	}
}
