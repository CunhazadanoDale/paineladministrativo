package importacao_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	"github.com/google/uuid"
)

func importacaoParaRateio(t *testing.T) *domainimportacao.Importacao {
	t.Helper()

	importacao, err := domainimportacao.NovaImportacao(uuid.New(), uuid.New(), domainimportacao.DadosImportacao{
		FornecedorID: uuid.New(),
		Moeda:        domainfornecedor.MoedaDolar,
		Itens: []domainimportacao.ItemInput{
			{ProdutoID: uuid.New(), Quantidade: 1000, ValorMoedaCentesimos: 300000},
			{ProdutoID: uuid.New(), Quantidade: 300, ValorMoedaCentesimos: 100000},
		},
	}, agora)
	if err != nil {
		t.Fatalf("NovaImportacao: %v", err)
	}

	return importacao
}

func fechar(t *testing.T, importacao *domainimportacao.Importacao, centesimos, cambio int64) {
	t.Helper()

	parcelaNova, err := importacao.IncluirParcela(uuid.New(), parcela("Parcela", centesimos), agora)
	if err != nil {
		t.Fatalf("IncluirParcela: %v", err)
	}
	if _, err := importacao.FecharCambio(parcelaNova.ID, domainimportacao.FechamentoCambioInput{CambioMicro: &cambio, Vencimento: dia(2026, 10, 20)}, agora); err != nil {
		t.Fatalf("FecharCambio: %v", err)
	}
}

func TestCalcularCustoComParcelasFechadasECambioDoRestante(t *testing.T) {
	importacao := importacaoParaRateio(t)
	fechar(t, importacao, 120000, 5_000_000)
	if _, err := importacao.LancarCusto(uuid.New(), custoDe(domainimportacao.TipoCustoFreteInternacional, 400000, true), agora); err != nil {
		t.Fatalf("LancarCusto: %v", err)
	}
	if _, err := importacao.LancarCusto(uuid.New(), custoDe(domainimportacao.TipoCustoICMS, 999999, false), agora); err != nil {
		t.Fatalf("LancarCusto: %v", err)
	}

	if _, err := importacao.CalcularCusto(nil, nil); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("sem câmbio para o restante: erro = %v", err)
	}

	cambio := int64(5_500_000)
	calculo, err := importacao.CalcularCusto(&cambio, nil)
	if err != nil {
		t.Fatalf("CalcularCusto: %v", err)
	}

	if calculo.ParcelasFechadasCentavos != 600000 || calculo.SemCambioFechado.Centesimos() != 280000 || calculo.ValorMercadoriaCentavos != 2140000 {
		t.Errorf("mercadoria = %+v (esperado 6000,00 + 2800,00 × 5,5 = 21400,00)", calculo)
	}
	if calculo.CambioMedio.Micro() != 5_350_000 {
		t.Errorf("câmbio médio = %d, esperado 5,35", calculo.CambioMedio.Micro())
	}
	if calculo.CustosCentavos != 400000 || calculo.CustosForaDoRateioCentavos != 999999 {
		t.Errorf("custos = %d e fora do rateio %d", calculo.CustosCentavos, calculo.CustosForaDoRateioCentavos)
	}

	porcelanato, revestimento := calculo.Itens[0], calculo.Itens[1]
	if porcelanato.ValorMercadoriaCentavos != 1605000 || revestimento.ValorMercadoriaCentavos != 535000 {
		t.Errorf("mercadoria por item = %d e %d", porcelanato.ValorMercadoriaCentavos, revestimento.ValorMercadoriaCentavos)
	}
	if porcelanato.CustosCentavos != 300000 || revestimento.CustosCentavos != 100000 {
		t.Errorf("custos por item = %d e %d", porcelanato.CustosCentavos, revestimento.CustosCentavos)
	}
	if *porcelanato.CustoUnitarioCentavos != 1905 || *revestimento.CustoUnitarioCentavos != 2117 {
		t.Errorf("custo unitário = %d e %d (esperado 19050,00/1000 e 6350,00/300)", *porcelanato.CustoUnitarioCentavos, *revestimento.CustoUnitarioCentavos)
	}
}

func TestCalcularCustoSemRestanteDispensaCambio(t *testing.T) {
	importacao := importacaoParaRateio(t)
	fechar(t, importacao, 400000, 5_123_457)

	calculo, err := importacao.CalcularCusto(nil, nil)
	if err != nil {
		t.Fatalf("CalcularCusto: %v", err)
	}
	if calculo.CambioInformado != nil || calculo.ValorMercadoriaCentavos != 2049383 {
		t.Errorf("cálculo = %+v", calculo)
	}
	if soma := calculo.Itens[0].ValorMercadoriaCentavos + calculo.Itens[1].ValorMercadoriaCentavos; soma != calculo.ValorMercadoriaCentavos {
		t.Errorf("partes somam %d, total %d", soma, calculo.ValorMercadoriaCentavos)
	}
}

func TestCalcularCustoComRecebimentoParcial(t *testing.T) {
	importacao := importacaoParaRateio(t)
	fechar(t, importacao, 400000, 5_000_000)
	if _, err := importacao.LancarCusto(uuid.New(), custoDe(domainimportacao.TipoCustoDespachante, 100000, true), agora); err != nil {
		t.Fatalf("LancarCusto: %v", err)
	}

	recebidas := map[uuid.UUID]int{importacao.Itens[0].ID: 500}
	calculo, err := importacao.CalcularCusto(nil, recebidas)
	if err != nil {
		t.Fatalf("CalcularCusto: %v", err)
	}

	porcelanato, revestimento := calculo.Itens[0], calculo.Itens[1]
	if porcelanato.Quantidade != 500 || porcelanato.CustosCentavos != 100000 || *porcelanato.CustoUnitarioCentavos != 3200 {
		t.Errorf("porcelanato recebido pela metade = %+v", porcelanato)
	}
	if revestimento.Quantidade != 0 || revestimento.CustosCentavos != 0 || revestimento.CustoUnitarioCentavos != nil {
		t.Errorf("revestimento não recebido = %+v", revestimento)
	}
}

func TestCalcularCustoEmReaisEComBonificacao(t *testing.T) {
	importacao, err := domainimportacao.NovaImportacao(uuid.New(), uuid.New(), domainimportacao.DadosImportacao{
		FornecedorID: uuid.New(),
		Moeda:        domainfornecedor.MoedaReal,
		Itens: []domainimportacao.ItemInput{
			{ProdutoID: uuid.New(), Quantidade: 3, ValorMoedaCentesimos: 0},
			{ProdutoID: uuid.New(), Quantidade: 1, ValorMoedaCentesimos: 0},
		},
	}, agora)
	if err != nil {
		t.Fatalf("NovaImportacao: %v", err)
	}
	if _, err := importacao.LancarCusto(uuid.New(), custoDe(domainimportacao.TipoCustoFreteNacional, 1000, true), agora); err != nil {
		t.Fatalf("LancarCusto: %v", err)
	}

	calculo, err := importacao.CalcularCusto(nil, nil)
	if err != nil {
		t.Fatalf("CalcularCusto: %v", err)
	}
	if calculo.Itens[0].CustosCentavos != 750 || calculo.Itens[1].CustosCentavos != 250 || *calculo.Itens[0].CustoUnitarioCentavos != 250 {
		t.Errorf("bonificação rateada pela quantidade = %+v", calculo.Itens)
	}
	if calculo.CambioMedio.Micro() != 0 {
		t.Errorf("câmbio médio sem valor = %d", calculo.CambioMedio.Micro())
	}
}
