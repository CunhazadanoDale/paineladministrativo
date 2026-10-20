package importacao_test

import (
	"errors"
	"math"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

func TestParcelaGiganteNaoEstouraASoma(t *testing.T) {
	importacao := novaImportacaoDeTeste(t)
	if _, err := importacao.IncluirParcela(uuid.New(), parcela("Sinal", 100000), agora); err != nil {
		t.Fatalf("IncluirParcela: %v", err)
	}

	for _, valor := range []int64{math.MaxInt64 - 100, 1_000_000_000_000_001} {
		if _, err := importacao.IncluirParcela(uuid.New(), parcela("Gigante", valor), agora); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("parcela de %d: erro = %v", valor, err)
		}
	}
	if len(importacao.Parcelas) != 1 || importacao.SomaParcelasAtivas().Centesimos() != 100000 {
		t.Errorf("parcelas = %+v", importacao.Parcelas)
	}

	if _, err := importacao.LancarCusto(uuid.New(), custoDe(domainimportacao.TipoCustoFreteInternacional, 1_000_000_000_000_001, true), agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("custo acima do limite: erro = %v", err)
	}
	dados := dadosDeTeste()
	dados.Itens[0].ValorMoedaCentesimos = 1_000_000_000_000_001
	if _, err := domainimportacao.NovaImportacao(uuid.New(), uuid.New(), dados, agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("item acima do limite: erro = %v", err)
	}
}

func TestMoedaEFornecedorNaoMudamComParcelas(t *testing.T) {
	importacao := novaImportacaoDeTeste(t)
	sinal, _ := importacao.IncluirParcela(uuid.New(), parcela("Sinal", 100000), agora)

	outraMoeda := dadosDeTeste()
	outraMoeda.FornecedorID = importacao.FornecedorID
	outraMoeda.Moeda = domainfornecedor.MoedaEuro
	if err := importacao.Alterar(outraMoeda, agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("trocar moeda com parcela aberta: erro = %v", err)
	}

	outroFornecedor := dadosDeTeste()
	if err := importacao.Alterar(outroFornecedor, agora); err != nil {
		t.Errorf("trocar fornecedor com parcela só aberta: %v", err)
	}

	cambio := int64(5_000_000)
	if _, err := importacao.FecharCambio(sinal.ID, domainimportacao.FechamentoCambioInput{CambioMicro: &cambio, Vencimento: dia(2026, 10, 20)}, agora); err != nil {
		t.Fatalf("FecharCambio: %v", err)
	}
	maisUmFornecedor := dadosDeTeste()
	if err := importacao.Alterar(maisUmFornecedor, agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("trocar fornecedor com câmbio fechado: erro = %v", err)
	}
}

func TestReceberRecusaContaEncerradaNoCusto(t *testing.T) {
	importacao := desembaracada(t)
	sinal, _ := importacao.IncluirParcela(uuid.New(), parcela("Sinal", 100000), agora)
	cambio := int64(5_000_000)
	if _, err := importacao.FecharCambio(sinal.ID, domainimportacao.FechamentoCambioInput{CambioMicro: &cambio, Vencimento: dia(2026, 10, 20)}, agora); err != nil {
		t.Fatalf("FecharCambio: %v", err)
	}
	comConta(t, importacao, sinal.ID, domainsolicitacao.StatusRejeitado)

	entrada := domainimportacao.RecebimentoInput{RecebidaEm: dia(2026, 10, 13), LocalID: uuid.New(), CambioMicro: &cambio}
	if _, err := importacao.Receber(entrada, agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("receber com parcela de conta rejeitada: erro = %v", err)
	}

	comConta(t, importacao, sinal.ID, domainsolicitacao.StatusAprovado)
	frete, _ := importacao.LancarCusto(uuid.New(), custoDe(domainimportacao.TipoCustoFreteInternacional, 1000, true), agora)
	cancelado := domainsolicitacao.StatusCancelado
	frete.StatusContaAPagar = &cancelado
	if _, err := importacao.Receber(entrada, agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("receber com custo de conta cancelada: erro = %v", err)
	}

	frete.CompoeCusto = false
	if _, err := importacao.Receber(entrada, agora); err != nil {
		t.Errorf("custo fora do rateio não deveria bloquear: %v", err)
	}
}

func TestCambioInformadoSempreValidado(t *testing.T) {
	importacao := desembaracada(t)
	negativo := int64(-5)
	if _, err := importacao.CalcularCusto(&negativo, nil); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("câmbio negativo: erro = %v", err)
	}

	dados := dadosDeTeste()
	dados.Moeda = domainfornecedor.MoedaReal
	emReais, _ := domainimportacao.NovaImportacao(uuid.New(), uuid.New(), dados, agora)
	cinco := int64(5_000_000)
	if _, err := emReais.CalcularCusto(&cinco, nil); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("câmbio diferente de 1 em reais: erro = %v", err)
	}
	if calculo, err := emReais.CalcularCusto(nil, nil); err != nil || calculo.ValorMercadoriaCentavos != 450000 {
		t.Errorf("reais sem câmbio = %+v (%v)", calculo, err)
	}
}

func TestAlertaUsaAChegadaReal(t *testing.T) {
	prevista := dia(2026, 12, 20)
	chegou := dia(2026, 11, 1)
	prazo := dia(2026, 11, 20)
	alocacao := domainimportacao.AlocacaoAberta{PedidoItemID: uuid.New(), ChegadaPrevista: &prevista, ChegadaEm: &chegou, PrazoLimite: &prazo}

	if alertas := domainimportacao.ClassificarAlertas(10, []domainimportacao.AlocacaoAberta{alocacao}, nil); len(alertas.PrazoEmRisco) != 0 {
		t.Errorf("chegou antes do prazo e continua em risco: %+v", alertas.PrazoEmRisco)
	}
}

func TestAcoesQueExpoemValores(t *testing.T) {
	if !domainimportacao.AcaoParcelaCambioFechado.ExpoeValores() || !domainimportacao.AcaoRecebida.ExpoeValores() ||
		!domainimportacao.AcaoCustoLancado.ExpoeValores() {
		t.Error("ações com valores não marcadas")
	}
	if domainimportacao.AcaoEmbarcada.ExpoeValores() || domainimportacao.AcaoAlocacaoAlterada.ExpoeValores() {
		t.Error("ações sem valores marcadas")
	}
}

func TestAlocacaoRecusadaForaDeEstado(t *testing.T) {
	importacao := novaImportacaoDeTeste(t)
	if err := importacao.ExigirAlocacoesAbertas(); err != nil {
		t.Errorf("aberta: %v", err)
	}
	if err := importacao.Cancelar("x", agora); err != nil {
		t.Fatalf("Cancelar: %v", err)
	}
	if err := importacao.ExigirAlocacoesAbertas(); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("cancelada: erro = %v", err)
	}
}
