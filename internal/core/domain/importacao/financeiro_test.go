package importacao_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

func parcela(descricao string, centesimos int64) domainimportacao.DadosParcela {
	return domainimportacao.DadosParcela{Descricao: descricao, ValorMoedaCentesimos: centesimos, VencimentoPrevisto: dia(2026, 11, 1)}
}

func custoDe(tipo domainimportacao.TipoCusto, centavos int64, compoe bool) domainimportacao.DadosCusto {
	return domainimportacao.DadosCusto{
		Tipo: tipo, FornecedorID: uuid.New(), ValorCentavos: centavos, Vencimento: dia(2026, 11, 10),
		CompoeCusto: compoe, CategoriaFinanceiraID: uuid.New(),
	}
}

func comConta(t *testing.T, importacao *domainimportacao.Importacao, parcelaID uuid.UUID, status domainsolicitacao.Status) {
	t.Helper()

	alvo := importacao.Parcela(parcelaID)
	id := uuid.New()
	alvo.SolicitacaoID = &id
	alvo.StatusContaAPagar = &status
}

func TestParcelasNaoPassamDoTotal(t *testing.T) {
	importacao := novaImportacaoDeTeste(t)

	sinal, err := importacao.IncluirParcela(uuid.New(), parcela(" Sinal 30% ", 135000), agora)
	if err != nil || sinal.Status != domainimportacao.StatusParcelaAberta || sinal.Descricao != "Sinal 30%" {
		t.Fatalf("sinal = %+v (%v)", sinal, err)
	}
	if _, err := importacao.IncluirParcela(uuid.New(), parcela("Saldo", 315001), agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("parcela acima do total: erro = %v", err)
	}
	saldo, err := importacao.IncluirParcela(uuid.New(), parcela("Saldo 70%", 315000), agora)
	if err != nil {
		t.Fatalf("saldo: %v", err)
	}
	if importacao.SomaParcelasAtivas().Centesimos() != 450000 {
		t.Errorf("soma das parcelas = %d", importacao.SomaParcelasAtivas().Centesimos())
	}

	if _, err := importacao.AlterarParcela(saldo.ID, parcela("Saldo", 315001), agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("alterar acima do total: erro = %v", err)
	}
	if importacao.Parcela(saldo.ID).Valor.Centesimos() != 315000 {
		t.Errorf("parcela mudou mesmo com erro: %+v", importacao.Parcela(saldo.ID))
	}

	dados := dadosDeTeste()
	dados.FornecedorID = importacao.FornecedorID
	dados.Itens[0].ValorMoedaCentesimos = 400000
	if err := importacao.Alterar(dados, agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("reduzir os itens abaixo das parcelas: erro = %v", err)
	}

	if _, err := importacao.RemoverParcela(saldo.ID, agora); err != nil {
		t.Fatalf("RemoverParcela: %v", err)
	}
	if len(importacao.Parcelas) != 1 || importacao.Parcela(saldo.ID) != nil {
		t.Errorf("parcelas depois de remover = %+v", importacao.Parcelas)
	}

	casos := []struct {
		nome  string
		dados domainimportacao.DadosParcela
	}{
		{"sem descrição", parcela(" ", 100)},
		{"valor zero", parcela("Sinal", 0)},
		{"sem vencimento", domainimportacao.DadosParcela{Descricao: "Sinal", ValorMoedaCentesimos: 100}},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			if _, err := importacao.IncluirParcela(uuid.New(), caso.dados, agora); !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("erro = %v", err)
			}
		})
	}
}

func TestFecharCambioConverteEmReais(t *testing.T) {
	importacao := novaImportacaoDeTeste(t)
	sinal, _ := importacao.IncluirParcela(uuid.New(), parcela("Sinal", 135001), agora)

	semCambio := domainimportacao.FechamentoCambioInput{Vencimento: dia(2026, 10, 20)}
	if _, err := importacao.FecharCambio(sinal.ID, semCambio, agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("sem câmbio: erro = %v", err)
	}
	cambio := int64(5_432_150)
	if _, err := importacao.FecharCambio(sinal.ID, domainimportacao.FechamentoCambioInput{CambioMicro: &cambio}, agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("sem vencimento: erro = %v", err)
	}

	fechada, err := importacao.FecharCambio(sinal.ID, domainimportacao.FechamentoCambioInput{CambioMicro: &cambio, Vencimento: dia(2026, 10, 20)}, agora)
	if err != nil {
		t.Fatalf("FecharCambio: %v", err)
	}
	if fechada.Status != domainimportacao.StatusParcelaCambioFechado || *fechada.ValorCentavos != 733346 || fechada.Cambio.Micro() != cambio {
		t.Errorf("parcela fechada = %+v, valor %d (esperado 1350,01 × 5,43215 = 7333,46)", fechada, *fechada.ValorCentavos)
	}
	if _, err := importacao.FecharCambio(sinal.ID, domainimportacao.FechamentoCambioInput{CambioMicro: &cambio, Vencimento: dia(2026, 10, 20)}, agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("fechar de novo: erro = %v", err)
	}
	if _, err := importacao.AlterarParcela(sinal.ID, parcela("Sinal", 100), agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("alterar com câmbio fechado: erro = %v", err)
	}
	if _, err := importacao.RemoverParcela(sinal.ID, agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("remover com câmbio fechado: erro = %v", err)
	}
	if _, err := importacao.FecharCambio(uuid.New(), semCambio, agora); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("parcela inexistente: erro = %v", err)
	}
}

func TestCambioDaImportacaoEmReais(t *testing.T) {
	dados := dadosDeTeste()
	dados.Moeda = domainfornecedor.MoedaReal
	importacao, err := domainimportacao.NovaImportacao(uuid.New(), uuid.New(), dados, agora)
	if err != nil {
		t.Fatalf("NovaImportacao: %v", err)
	}
	sinal, _ := importacao.IncluirParcela(uuid.New(), parcela("Sinal", 100000), agora)

	outro := int64(5_000_000)
	if _, err := importacao.FecharCambio(sinal.ID, domainimportacao.FechamentoCambioInput{CambioMicro: &outro, Vencimento: dia(2026, 10, 20)}, agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("câmbio diferente de 1 em reais: erro = %v", err)
	}
	fechada, err := importacao.FecharCambio(sinal.ID, domainimportacao.FechamentoCambioInput{Vencimento: dia(2026, 10, 20)}, agora)
	if err != nil || *fechada.ValorCentavos != 100000 {
		t.Errorf("parcela em reais = %+v (%v)", fechada, err)
	}
}

func TestReabrirParcelaExigeContaEncerrada(t *testing.T) {
	importacao := novaImportacaoDeTeste(t)
	sinal, _ := importacao.IncluirParcela(uuid.New(), parcela("Sinal", 135000), agora)

	if _, _, err := importacao.ReabrirParcela(sinal.ID, uuid.New(), agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("reabrir aberta: erro = %v", err)
	}
	cambio := int64(5_400_000)
	if _, err := importacao.FecharCambio(sinal.ID, domainimportacao.FechamentoCambioInput{CambioMicro: &cambio, Vencimento: dia(2026, 10, 20)}, agora); err != nil {
		t.Fatalf("FecharCambio: %v", err)
	}
	comConta(t, importacao, sinal.ID, domainsolicitacao.StatusAprovado)
	if _, _, err := importacao.ReabrirParcela(sinal.ID, uuid.New(), agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("reabrir com conta aprovada: erro = %v", err)
	}

	comConta(t, importacao, sinal.ID, domainsolicitacao.StatusRejeitado)
	antiga, nova, err := importacao.ReabrirParcela(sinal.ID, uuid.New(), agora)
	if err != nil {
		t.Fatalf("ReabrirParcela: %v", err)
	}
	if antiga.Status != domainimportacao.StatusParcelaCancelada || nova.Status != domainimportacao.StatusParcelaAberta ||
		nova.ID == antiga.ID || nova.Valor != antiga.Valor || len(importacao.Parcelas) != 2 {
		t.Errorf("antiga %+v, nova %+v", antiga, nova)
	}
	if importacao.SomaParcelasAtivas().Centesimos() != 135000 {
		t.Errorf("soma das ativas = %d", importacao.SomaParcelasAtivas().Centesimos())
	}
}

func TestCustos(t *testing.T) {
	importacao := novaImportacaoDeTeste(t)

	frete, err := importacao.LancarCusto(uuid.New(), custoDe(domainimportacao.TipoCustoFreteInternacional, 800000, true), agora)
	if err != nil {
		t.Fatalf("LancarCusto: %v", err)
	}
	if frete.Descricao != "Frete internacional" || !frete.EntraNoRateio() || frete.Posterior {
		t.Errorf("frete = %+v", frete)
	}

	casos := []struct {
		nome  string
		dados domainimportacao.DadosCusto
	}{
		{"tipo vazio", custoDe("", 100, true)},
		{"valor zero", custoDe(domainimportacao.TipoCustoICMS, 0, true)},
		{"sem fornecedor", domainimportacao.DadosCusto{Tipo: domainimportacao.TipoCustoICMS, ValorCentavos: 100, Vencimento: dia(2026, 11, 1), CategoriaFinanceiraID: uuid.New()}},
		{"sem categoria", domainimportacao.DadosCusto{Tipo: domainimportacao.TipoCustoICMS, ValorCentavos: 100, Vencimento: dia(2026, 11, 1), FornecedorID: uuid.New()}},
		{"sem vencimento", domainimportacao.DadosCusto{Tipo: domainimportacao.TipoCustoICMS, ValorCentavos: 100, FornecedorID: uuid.New(), CategoriaFinanceiraID: uuid.New()}},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			if _, err := importacao.LancarCusto(uuid.New(), caso.dados, agora); !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("erro = %v", err)
			}
		})
	}

	if _, err := importacao.CancelarCusto(frete.ID, uuid.New(), agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("cancelar custo sem conta encerrada: erro = %v", err)
	}
	cancelado := domainsolicitacao.StatusCancelado
	importacao.Custo(frete.ID).StatusContaAPagar = &cancelado
	if _, err := importacao.CancelarCusto(frete.ID, uuid.New(), agora); err != nil {
		t.Fatalf("CancelarCusto: %v", err)
	}
	if frete.Ativo() || frete.EntraNoRateio() {
		t.Errorf("custo cancelado continua ativo: %+v", frete)
	}
	if _, err := importacao.CancelarCusto(frete.ID, uuid.New(), agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("cancelar de novo: erro = %v", err)
	}

	if tipo, err := domainimportacao.NovoTipoCusto(" ICMS "); err != nil || tipo != domainimportacao.TipoCustoICMS {
		t.Errorf("NovoTipoCusto = %q (%v)", tipo, err)
	}
	if _, err := domainimportacao.NovoTipoCusto("multa"); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("tipo inválido: erro = %v", err)
	}
}

func TestCancelarImportacaoComContasAPagar(t *testing.T) {
	importacao := novaImportacaoDeTeste(t)
	sinal, _ := importacao.IncluirParcela(uuid.New(), parcela("Sinal", 135000), agora)
	cambio := int64(5_400_000)
	if _, err := importacao.FecharCambio(sinal.ID, domainimportacao.FechamentoCambioInput{CambioMicro: &cambio, Vencimento: dia(2026, 10, 20)}, agora); err != nil {
		t.Fatalf("FecharCambio: %v", err)
	}
	comConta(t, importacao, sinal.ID, domainsolicitacao.StatusPago)

	if err := importacao.Cancelar("desistência", agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("cancelar com conta paga: erro = %v", err)
	}
	if contas := importacao.ContasAPagarAbertas(); len(contas) != 1 || contas[0] != *sinal.SolicitacaoID {
		t.Errorf("contas abertas = %v", contas)
	}

	comConta(t, importacao, sinal.ID, domainsolicitacao.StatusCancelado)
	if err := importacao.Cancelar("desistência", agora); err != nil {
		t.Fatalf("Cancelar: %v", err)
	}
	if _, err := importacao.IncluirParcela(uuid.New(), parcela("Outra", 100), agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("parcela em importação cancelada: erro = %v", err)
	}
	if _, err := importacao.LancarCusto(uuid.New(), custoDe(domainimportacao.TipoCustoSeguro, 100, true), agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("custo em importação cancelada: erro = %v", err)
	}
}
