package pedido_test

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	"github.com/google/uuid"
)

var agora = time.Date(2026, 10, 14, 15, 0, 0, 0, time.UTC)

func dadosDeTeste(modelo domainpedido.Modelo) domainpedido.DadosPedido {
	return domainpedido.DadosPedido{
		ClienteID: uuid.New(),
		Modelo:    modelo,
		Itens: []domainpedido.ItemInput{
			{ProdutoID: uuid.New(), Quantidade: 10, PrecoUnitarioCentavos: 1500},
			{ProdutoID: uuid.New(), Quantidade: 4, PrecoUnitarioCentavos: 2500},
		},
	}
}

func novoPedidoDeTeste(t *testing.T, modelo domainpedido.Modelo) *domainpedido.Pedido {
	t.Helper()

	pedido, err := domainpedido.NovoPedido(uuid.New(), uuid.New(), dadosDeTeste(modelo))
	if err != nil {
		t.Fatalf("NovoPedido falhou: %v", err)
	}

	return pedido
}

func confirmadoDeTeste(t *testing.T, modelo domainpedido.Modelo) *domainpedido.Pedido {
	t.Helper()

	pedido := novoPedidoDeTeste(t, modelo)
	if err := pedido.Confirmar(agora); err != nil {
		t.Fatalf("Confirmar falhou: %v", err)
	}

	return pedido
}

func TestNovoPedidoNasceComoProposta(t *testing.T) {
	dados := dadosDeTeste(domainpedido.ModeloProntaEntrega)
	obraNula := uuid.Nil
	numero := "  OC-778  "
	prazo := time.Date(2026, 12, 20, 18, 0, 0, 0, time.UTC)
	dados.ObraID = &obraNula
	dados.NumeroPedidoCliente = &numero
	dados.PrazoLimite = &prazo
	dados.Observacoes = "  entregar pela manhã  "

	pedido, err := domainpedido.NovoPedido(uuid.New(), uuid.New(), dados)
	if err != nil {
		t.Fatalf("NovoPedido falhou: %v", err)
	}

	if pedido.Status != domainpedido.StatusProposta || pedido.Versao != 1 || pedido.Numero != 0 {
		t.Errorf("pedido nasceu com status %q, versão %d e número %d", pedido.Status, pedido.Versao, pedido.Numero)
	}
	if pedido.ValorTotalCentavos != 25000 {
		t.Errorf("valor total = %d, esperado 25000", pedido.ValorTotalCentavos)
	}
	if pedido.Condicao.PrazoBoletoDias() == nil || *pedido.Condicao.PrazoBoletoDias() != 30 {
		t.Errorf("condição da pronta entrega sem boleto de 30 dias")
	}
	if pedido.ObraID != nil {
		t.Errorf("obra nula virou %v", pedido.ObraID)
	}
	if pedido.NumeroPedidoCliente == nil || *pedido.NumeroPedidoCliente != "OC-778" || pedido.Observacoes != "entregar pela manhã" {
		t.Errorf("textos não foram limpos: %v / %q", pedido.NumeroPedidoCliente, pedido.Observacoes)
	}
	if pedido.PrazoLimite == nil || !pedido.PrazoLimite.Equal(time.Date(2026, 12, 20, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("prazo-limite = %v, esperado só a data", pedido.PrazoLimite)
	}
	for _, item := range pedido.Itens {
		if item.PedidoID != pedido.ID || item.QuantidadeEntregue != 0 {
			t.Errorf("item %+v não pertence ao pedido ou já nasceu entregue", item)
		}
	}
}

func TestNovoPedidoValida(t *testing.T) {
	produto := uuid.New()
	prazo := agora.AddDate(0, 1, 0)
	longo := strings.Repeat("x", 61)

	casos := []struct {
		nome    string
		alterar func(*domainpedido.DadosPedido)
	}{
		{"sem cliente", func(d *domainpedido.DadosPedido) { d.ClienteID = uuid.Nil }},
		{"modelo inválido", func(d *domainpedido.DadosPedido) { d.Modelo = domainpedido.ModeloDe("consignado") }},
		{"sem itens", func(d *domainpedido.DadosPedido) { d.Itens = nil }},
		{"produto repetido", func(d *domainpedido.DadosPedido) {
			d.Itens = []domainpedido.ItemInput{
				{ProdutoID: produto, Quantidade: 1, PrecoUnitarioCentavos: 1},
				{ProdutoID: produto, Quantidade: 2, PrecoUnitarioCentavos: 1},
			}
		}},
		{"item sem produto", func(d *domainpedido.DadosPedido) { d.Itens[0].ProdutoID = uuid.Nil }},
		{"quantidade zero", func(d *domainpedido.DadosPedido) { d.Itens[0].Quantidade = 0 }},
		{"quantidade acima do int32", func(d *domainpedido.DadosPedido) { d.Itens[0].Quantidade = math.MaxInt32 + 1 }},
		{"preço negativo", func(d *domainpedido.DadosPedido) { d.Itens[0].PrecoUnitarioCentavos = -1 }},
		{"total estourado", func(d *domainpedido.DadosPedido) {
			d.Itens[0].PrecoUnitarioCentavos = math.MaxInt64 / 10
			d.Itens[1].PrecoUnitarioCentavos = math.MaxInt64 / 4
		}},
		{"prazo na importação", func(d *domainpedido.DadosPedido) {
			d.Modelo = domainpedido.ModeloImportacao
			d.PrazoLimite = &prazo
		}},
		{"número do cliente longo", func(d *domainpedido.DadosPedido) { d.NumeroPedidoCliente = &longo }},
		{"observações longas", func(d *domainpedido.DadosPedido) { d.Observacoes = strings.Repeat("o", 2001) }},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			dados := dadosDeTeste(domainpedido.ModeloProntaEntrega)
			caso.alterar(&dados)

			if _, err := domainpedido.NovoPedido(uuid.New(), uuid.New(), dados); !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("erro %v, esperado erro de validação", err)
			}
		})
	}

	if _, err := domainpedido.NovoPedido(uuid.Nil, uuid.New(), dadosDeTeste(domainpedido.ModeloProntaEntrega)); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("sem empresa = %v, esperado erro de validação", err)
	}
	if _, err := domainpedido.NovoPedido(uuid.New(), uuid.Nil, dadosDeTeste(domainpedido.ModeloProntaEntrega)); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("sem usuário = %v, esperado erro de validação", err)
	}
}

func TestAlterarSoNaProposta(t *testing.T) {
	pedido := novoPedidoDeTeste(t, domainpedido.ModeloProntaEntrega)
	dados := dadosDeTeste(domainpedido.ModeloImportacao)
	dados.Itens = dados.Itens[:1]

	if err := pedido.Alterar(dados, agora); err != nil {
		t.Fatalf("Alterar falhou: %v", err)
	}
	if pedido.Modelo != domainpedido.ModeloImportacao || pedido.Condicao.Entrada().PontosBase() != 3000 {
		t.Errorf("troca de modelo não refez a condição: %q, entrada %d", pedido.Modelo, pedido.Condicao.Entrada().PontosBase())
	}
	if len(pedido.Itens) != 1 || pedido.ValorTotalCentavos != 15000 {
		t.Errorf("itens %d e total %d, esperado 1 e 15000", len(pedido.Itens), pedido.ValorTotalCentavos)
	}

	invalido := dadosDeTeste(domainpedido.ModeloProntaEntrega)
	invalido.Itens = nil
	if err := pedido.Alterar(invalido, agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("alteração inválida = %v, esperado erro de validação", err)
	}
	if len(pedido.Itens) != 1 {
		t.Errorf("alteração inválida mudou os itens para %d", len(pedido.Itens))
	}

	if err := pedido.Confirmar(agora); err != nil {
		t.Fatalf("Confirmar falhou: %v", err)
	}
	if err := pedido.Alterar(dadosDeTeste(domainpedido.ModeloProntaEntrega), agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("alterar confirmado = %v, esperado conflito", err)
	}
}

func TestConfirmarSoAPartirDaProposta(t *testing.T) {
	pedido := confirmadoDeTeste(t, domainpedido.ModeloProntaEntrega)

	if pedido.Status != domainpedido.StatusConfirmado {
		t.Errorf("status = %q, esperado confirmado", pedido.Status)
	}
	if err := pedido.Confirmar(agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("confirmar duas vezes = %v, esperado conflito", err)
	}
}

func TestConferirVersao(t *testing.T) {
	pedido := novoPedidoDeTeste(t, domainpedido.ModeloProntaEntrega)

	if err := pedido.ConferirVersao(1); err != nil {
		t.Errorf("versão atual = %v, esperado nil", err)
	}
	if err := pedido.ConferirVersao(0); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("versão ausente = %v, esperado erro de validação", err)
	}
	if err := pedido.ConferirVersao(2); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("versão divergente = %v, esperado conflito", err)
	}
}

func TestRegistrarEntradaDefineOPrazoDaImportacao(t *testing.T) {
	pedido := confirmadoDeTeste(t, domainpedido.ModeloImportacao)
	pagaEm := time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)

	if err := pedido.RegistrarEntrada(pagaEm, 7500, agora); err != nil {
		t.Fatalf("RegistrarEntrada falhou: %v", err)
	}

	if pedido.EntradaPagaEm == nil || !pedido.EntradaPagaEm.Equal(time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("entrada paga em %v", pedido.EntradaPagaEm)
	}
	if pedido.EntradaPagaCentavos == nil || *pedido.EntradaPagaCentavos != 7500 {
		t.Errorf("valor da entrada = %v", pedido.EntradaPagaCentavos)
	}
	if pedido.PrazoLimite == nil || !pedido.PrazoLimite.Equal(time.Date(2027, 2, 7, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("prazo-limite = %v, esperado 2027-02-07", pedido.PrazoLimite)
	}
	if err := pedido.RegistrarEntrada(pagaEm, 7500, agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("segunda entrada = %v, esperado conflito", err)
	}
}

func TestRegistrarEntradaValida(t *testing.T) {
	ontem := agora.AddDate(0, 0, -1)
	amanha := agora.AddDate(0, 0, 1)

	casos := []struct {
		nome      string
		pedido    func(*testing.T) *domainpedido.Pedido
		pagaEm    time.Time
		centavos  int64
		sentinela error
	}{
		{"pronta entrega", func(t *testing.T) *domainpedido.Pedido {
			return confirmadoDeTeste(t, domainpedido.ModeloProntaEntrega)
		}, ontem, 100, domain.ErrValidacao},
		{"proposta", func(t *testing.T) *domainpedido.Pedido {
			return novoPedidoDeTeste(t, domainpedido.ModeloImportacao)
		}, ontem, 100, domain.ErrConflito},
		{"valor zero", func(t *testing.T) *domainpedido.Pedido {
			return confirmadoDeTeste(t, domainpedido.ModeloImportacao)
		}, ontem, 0, domain.ErrValidacao},
		{"valor acima do total", func(t *testing.T) *domainpedido.Pedido {
			return confirmadoDeTeste(t, domainpedido.ModeloImportacao)
		}, ontem, 25001, domain.ErrValidacao},
		{"data futura", func(t *testing.T) *domainpedido.Pedido {
			return confirmadoDeTeste(t, domainpedido.ModeloImportacao)
		}, amanha, 100, domain.ErrValidacao},
		{"sem data", func(t *testing.T) *domainpedido.Pedido {
			return confirmadoDeTeste(t, domainpedido.ModeloImportacao)
		}, time.Time{}, 100, domain.ErrValidacao},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			pedido := caso.pedido(t)

			if err := pedido.RegistrarEntrada(caso.pagaEm, caso.centavos, agora); !errors.Is(err, caso.sentinela) {
				t.Errorf("erro %v, esperado %v", err, caso.sentinela)
			}
			if pedido.EntradaPagaEm != nil {
				t.Error("entrada inválida foi gravada")
			}
		})
	}
}

func TestCancelarSemEntradaNaoTemDevolucao(t *testing.T) {
	pedido := novoPedidoDeTeste(t, domainpedido.ModeloProntaEntrega)
	valor := int64(100)

	if _, err := pedido.Cancelar(domainpedido.CancelamentoInput{Motivo: "cliente desistiu", ValorADevolverCentavos: &valor}, agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("devolução sem entrada = %v, esperado erro de validação", err)
	}
	if _, err := pedido.Cancelar(domainpedido.CancelamentoInput{Motivo: "   "}, agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("sem motivo = %v, esperado erro de validação", err)
	}
	if pedido.Status != domainpedido.StatusProposta {
		t.Fatalf("cancelamento inválido mudou o status para %q", pedido.Status)
	}

	if _, err := pedido.Cancelar(domainpedido.CancelamentoInput{Motivo: "  cliente desistiu  "}, agora); err != nil {
		t.Fatalf("Cancelar falhou: %v", err)
	}
	if pedido.Status != domainpedido.StatusCancelado || pedido.CanceladoMotivo == nil || *pedido.CanceladoMotivo != "cliente desistiu" || pedido.ValorADevolverCentavos != nil {
		t.Errorf("pedido cancelado = %q, motivo %v, devolução %v", pedido.Status, pedido.CanceladoMotivo, pedido.ValorADevolverCentavos)
	}
	if _, err := pedido.Cancelar(domainpedido.CancelamentoInput{Motivo: "de novo"}, agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("cancelar cancelado = %v, esperado conflito", err)
	}
}

func TestCancelarComEntradaDevolveAteAEntrada(t *testing.T) {
	novoComEntrada := func(t *testing.T) *domainpedido.Pedido {
		t.Helper()

		pedido := confirmadoDeTeste(t, domainpedido.ModeloImportacao)
		if err := pedido.RegistrarEntrada(agora, 7500, agora); err != nil {
			t.Fatalf("RegistrarEntrada falhou: %v", err)
		}

		return pedido
	}
	parcial := int64(5000)
	acima := int64(7501)
	negativo := int64(-1)

	casos := []struct {
		nome      string
		informado *int64
		esperado  int64
		sentinela error
	}{
		{"padrão é a entrada", nil, 7500, nil},
		{"valor parcial", &parcial, 5000, nil},
		{"acima da entrada", &acima, 0, domain.ErrValidacao},
		{"negativo", &negativo, 0, domain.ErrValidacao},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			pedido := novoComEntrada(t)

			_, err := pedido.Cancelar(domainpedido.CancelamentoInput{Motivo: "atraso", ValorADevolverCentavos: caso.informado}, agora)
			if !errors.Is(err, caso.sentinela) {
				t.Fatalf("erro %v, esperado %v", err, caso.sentinela)
			}
			if caso.sentinela != nil {
				return
			}
			if pedido.ValorADevolverCentavos == nil || *pedido.ValorADevolverCentavos != caso.esperado {
				t.Errorf("valor a devolver = %v, esperado %d", pedido.ValorADevolverCentavos, caso.esperado)
			}
		})
	}
}
