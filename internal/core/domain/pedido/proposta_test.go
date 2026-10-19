package pedido_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	"github.com/google/uuid"
)

func esperarErro(t *testing.T, err error, sentinela error, trecho string) {
	t.Helper()

	if !errors.Is(err, sentinela) || !strings.Contains(err.Error(), trecho) {
		t.Fatalf("erro = %v; esperado %v com %q", err, sentinela, trecho)
	}
}

func inteiro(valor int) *int {
	return &valor
}

func centavos(valor int64) *int64 {
	return &valor
}

func TestPropostaNasceComValidadeDe15DiasERevisao1(t *testing.T) {
	pedido := novoPedidoDeTeste(t, domainpedido.ModeloProntaEntrega)
	hoje := domainpedido.Hoje(time.Now().UTC())

	if pedido.Validade == nil || !pedido.Validade.Equal(hoje.AddDate(0, 0, 15)) || pedido.Revisao != 1 || pedido.Frete != nil {
		t.Fatalf("proposta = validade %v, revisão %d, frete %+v", pedido.Validade, pedido.Revisao, pedido.Frete)
	}
	if pedido.Vencida(time.Now().UTC()) {
		t.Fatalf("proposta nova não está vencida")
	}
}

func TestPropostaVencidaNaoConfirmaAteRenovar(t *testing.T) {
	pedido := novoPedidoDeTeste(t, domainpedido.ModeloProntaEntrega)
	depoisDaValidade := pedido.Validade.AddDate(0, 0, 1).Add(15 * time.Hour)

	if !pedido.Vencida(depoisDaValidade) {
		t.Fatalf("a proposta deveria estar vencida no dia seguinte à validade")
	}
	esperarErro(t, pedido.Confirmar(depoisDaValidade), domain.ErrConflito, "proposta vencida em")

	noUltimoDia := pedido.Validade.Add(23 * time.Hour)
	if pedido.Vencida(noUltimoDia) {
		t.Fatalf("no último dia da validade, em Brasília, a proposta ainda vale")
	}

	dados := dadosDeTeste(domainpedido.ModeloProntaEntrega)
	renovada := domainpedido.Hoje(depoisDaValidade).AddDate(0, 0, 10)
	dados.Validade = &renovada
	if err := pedido.Alterar(dados, depoisDaValidade); err != nil {
		t.Fatalf("Alterar: %v", err)
	}
	if pedido.Revisao != 2 || !pedido.Validade.Equal(renovada) {
		t.Fatalf("renovada = revisão %d, validade %v", pedido.Revisao, pedido.Validade)
	}
	if err := pedido.Confirmar(depoisDaValidade); err != nil {
		t.Fatalf("Confirmar depois de renovar: %v", err)
	}
	if pedido.Vencida(renovada.AddDate(0, 1, 0)) {
		t.Fatalf("pedido confirmado nunca fica vencido")
	}
}

func TestValidadeDaPropostaNaoPodeSerAnteriorAHoje(t *testing.T) {
	dados := dadosDeTeste(domainpedido.ModeloProntaEntrega)
	ontem := domainpedido.Hoje(time.Now().UTC()).AddDate(0, 0, -1)
	dados.Validade = &ontem

	_, err := domainpedido.NovoPedido(uuid.New(), uuid.New(), dados)
	esperarErro(t, err, domain.ErrValidacao, "validade da proposta não pode ser anterior a hoje")
}

func TestCondicoesEditaveisNaProposta(t *testing.T) {
	dados := dadosDeTeste(domainpedido.ModeloImportacao)
	dados.Condicao = &domainpedido.CondicaoInput{EntradaBP: inteiro(5000), PrazoEntregaDias: inteiro(90), PrazoBoletoDias: inteiro(28)}

	pedido, err := domainpedido.NovoPedido(uuid.New(), uuid.New(), dados)
	if err != nil {
		t.Fatalf("NovoPedido: %v", err)
	}
	if pedido.Condicao.Entrada().PontosBase() != 5000 || *pedido.Condicao.PrazoEntregaDias() != 90 || *pedido.Condicao.PrazoBoletoDias() != 28 {
		t.Fatalf("condição = %+v", pedido.Condicao)
	}

	parcial := dadosDeTeste(domainpedido.ModeloImportacao)
	parcial.Condicao = &domainpedido.CondicaoInput{EntradaBP: inteiro(0)}
	semEntrada, err := domainpedido.NovoPedido(uuid.New(), uuid.New(), parcial)
	if err != nil || semEntrada.Condicao.Entrada().PontosBase() != 0 || *semEntrada.Condicao.PrazoEntregaDias() != 120 {
		t.Fatalf("condição parcial = %+v, %v", semEntrada.Condicao, err)
	}

	casos := []struct {
		nome     string
		modelo   domainpedido.Modelo
		condicao domainpedido.CondicaoInput
		trecho   string
	}{
		{"entrada acima de 100%", domainpedido.ModeloImportacao, domainpedido.CondicaoInput{EntradaBP: inteiro(10001)}, "entrada deve estar entre"},
		{"prazo do boleto zero", domainpedido.ModeloProntaEntrega, domainpedido.CondicaoInput{PrazoBoletoDias: inteiro(0)}, "prazo do boleto"},
		{"prazo de entrega longo", domainpedido.ModeloImportacao, domainpedido.CondicaoInput{PrazoEntregaDias: inteiro(731)}, "prazo de entrega deve estar"},
		{"prazo de entrega na pronta entrega", domainpedido.ModeloProntaEntrega, domainpedido.CondicaoInput{PrazoEntregaDias: inteiro(10)}, "só vale na importação"},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			dados := dadosDeTeste(caso.modelo)
			condicao := caso.condicao
			dados.Condicao = &condicao

			_, err := domainpedido.NovoPedido(uuid.New(), uuid.New(), dados)
			esperarErro(t, err, domain.ErrValidacao, caso.trecho)
		})
	}
}

func TestFreteEstimadoEPrecoDeTabelaDoItem(t *testing.T) {
	dados := dadosDeTeste(domainpedido.ModeloProntaEntrega)
	frete, err := domainpedido.NovoFrete(domainpedido.TipoFreteCIF, centavos(18000))
	if err != nil {
		t.Fatalf("NovoFrete: %v", err)
	}
	dados.Frete = &frete
	dados.Itens[0].PrecoTabelaCentavos = centavos(1800)

	pedido, err := domainpedido.NovoPedido(uuid.New(), uuid.New(), dados)
	if err != nil {
		t.Fatalf("NovoPedido: %v", err)
	}
	if pedido.Frete == nil || pedido.Frete.Tipo() != domainpedido.TipoFreteCIF || *pedido.Frete.Centavos() != 18000 {
		t.Fatalf("frete = %+v", pedido.Frete)
	}
	if pedido.ValorTotalCentavos != 25000 {
		t.Fatalf("o frete estimado não soma ao total: %d", pedido.ValorTotalCentavos)
	}
	if desconto := pedido.Itens[0].DescontoUnitarioCentavos(); desconto == nil || *desconto != 300 {
		t.Fatalf("desconto = %v", desconto)
	}
	if pedido.Itens[1].DescontoUnitarioCentavos() != nil {
		t.Fatalf("item sem preço de tabela não tem desconto")
	}

	dados.Itens[0].PrecoTabelaCentavos = centavos(0)
	_, err = domainpedido.NovoPedido(uuid.New(), uuid.New(), dados)
	esperarErro(t, err, domain.ErrValidacao, "preço de tabela do item")
}
