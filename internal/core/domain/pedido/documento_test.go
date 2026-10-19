package pedido_test

import (
	"reflect"
	"testing"
	"time"

	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	"github.com/google/uuid"
)

func TestCondicoesPorExtensoDaImportacaoComFreteEValidade(t *testing.T) {
	dados := dadosDeTeste(domainpedido.ModeloImportacao)
	dados.Condicao = &domainpedido.CondicaoInput{EntradaBP: inteiro(2550), PrazoBoletoDias: inteiro(28)}
	frete, err := domainpedido.NovoFrete(domainpedido.TipoFreteCIF, centavos(123456789))
	if err != nil {
		t.Fatalf("NovoFrete: %v", err)
	}
	dados.Frete = &frete
	pedido, err := domainpedido.NovoPedido(uuid.New(), uuid.New(), dados)
	if err != nil {
		t.Fatalf("NovoPedido: %v", err)
	}

	esperadas := []string{
		"Entrada de 25,5% na confirmação do pedido.",
		"Saldo de 74,5% em boleto com vencimento em 28 dias após a chegada da mercadoria.",
		"Prazo de entrega de 120 dias contados do pagamento da entrada.",
		"Frete CIF (por conta do vendedor), estimado em R$ 1.234.567,89.",
		"Proposta válida até " + pedido.Validade.Format("02/01/2006") + ".",
	}
	if obtidas := pedido.CondicoesPorExtenso(); !reflect.DeepEqual(obtidas, esperadas) {
		t.Fatalf("condições = %#v", obtidas)
	}
}

func TestCondicoesPorExtensoDaProntaEntregaConfirmada(t *testing.T) {
	dados := dadosDeTeste(domainpedido.ModeloProntaEntrega)
	prazo := time.Date(2026, 12, 20, 0, 0, 0, 0, time.UTC)
	dados.PrazoLimite = &prazo
	frete, err := domainpedido.NovoFrete(domainpedido.TipoFreteFOB, nil)
	if err != nil {
		t.Fatalf("NovoFrete: %v", err)
	}
	dados.Frete = &frete
	pedido, err := domainpedido.NovoPedido(uuid.New(), uuid.New(), dados)
	if err != nil {
		t.Fatalf("NovoPedido: %v", err)
	}
	if err := pedido.Confirmar(time.Now().UTC()); err != nil {
		t.Fatalf("Confirmar: %v", err)
	}

	esperadas := []string{
		"Pagamento em boleto com vencimento em 30 dias após a entrega.",
		"Entrega até 20/12/2026.",
		"Frete FOB (por conta do cliente).",
	}
	if obtidas := pedido.CondicoesPorExtenso(); !reflect.DeepEqual(obtidas, esperadas) {
		t.Fatalf("condições = %#v", obtidas)
	}
}

func TestModelosDeVendaTrazemACondicaoPadrao(t *testing.T) {
	modelos := domainpedido.ModelosDeVenda()
	if len(modelos) != 2 || modelos[0].Modelo != domainpedido.ModeloProntaEntrega || modelos[1].Modelo != domainpedido.ModeloImportacao {
		t.Fatalf("modelos = %+v", modelos)
	}
	importacao := []string{
		"Entrada de 30% na confirmação do pedido.",
		"Saldo de 70% na chegada da mercadoria.",
		"Prazo de entrega de 120 dias contados do pagamento da entrada.",
	}
	if !reflect.DeepEqual(modelos[1].Condicoes, importacao) || modelos[1].Condicao.Entrada().PontosBase() != 3000 {
		t.Fatalf("importação = %#v", modelos[1])
	}
}

func TestDocumentoDaPropostaMontaItensEDesconto(t *testing.T) {
	dados := dadosDeTeste(domainpedido.ModeloProntaEntrega)
	dados.Itens[0].PrecoTabelaCentavos = centavos(1800)
	dados.Itens[1].PrecoTabelaCentavos = centavos(2000)
	pedido, err := domainpedido.NovoPedido(uuid.New(), uuid.New(), dados)
	if err != nil {
		t.Fatalf("NovoPedido: %v", err)
	}
	codigo := "CIM-50"
	partes := domainpedido.PartesDoDocumento{
		Empresa: domainpedido.EmpresaDoDocumento{RazaoSocial: "MIA Materiais Ltda"},
		Cliente: domainpedido.ClienteDoDocumento{Nome: "Construtora Alfa"},
		Produtos: map[uuid.UUID]domainpedido.ProdutoDoDocumento{
			pedido.Itens[0].ProdutoID: {ID: pedido.Itens[0].ProdutoID, Codigo: &codigo, Nome: "Cimento", Unidade: "sc"},
		},
	}

	documento, err := domainpedido.NovoDocumentoProposta(pedido, partes, time.Now().UTC())
	if err != nil {
		t.Fatalf("NovoDocumentoProposta: %v", err)
	}
	if documento.Revisao != 1 || documento.Vencida || documento.TotalCentavos != 25000 || documento.Empresa.RazaoSocial != "MIA Materiais Ltda" {
		t.Fatalf("documento = %+v", documento)
	}
	primeiro := documento.Itens[0]
	if primeiro.Nome != "Cimento" || *primeiro.Codigo != "CIM-50" || *primeiro.DescontoUnitarioCentavos != 300 || primeiro.SubtotalCentavos != 15000 {
		t.Fatalf("primeiro item = %+v", primeiro)
	}
	if *documento.Itens[1].DescontoUnitarioCentavos != -500 || documento.DescontoCentavos != 3000 {
		t.Fatalf("desconto = item %d, total %d", *documento.Itens[1].DescontoUnitarioCentavos, documento.DescontoCentavos)
	}
}
