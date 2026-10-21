package divida_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaindivida "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/divida"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

func dadosDaRenegociacao() domaindivida.DadosDivida {
	dados := dadosDeDivida()
	dados.SaldoInicialCentavos = 250000
	dados.DataSaldo = data(2026, 10, 15)
	dados.Parcelas = []domaindivida.ParcelaInput{
		{Vencimento: data(2026, 11, 15), AmortizacaoCentavos: 125000, JurosCentavos: 4000},
		{Vencimento: data(2026, 12, 15), AmortizacaoCentavos: 125000, JurosCentavos: 2000},
	}

	return dados
}

func TestRenegociarCancelaOAbertoEMantemOPago(t *testing.T) {
	divida := novaDivida(t)
	if _, err := divida.GerarContasAte(data(2026, 11, 30), uuid.New(), agoraDoTeste); err != nil {
		t.Fatalf("GerarContasAte falhou: %v", err)
	}
	primeira, segunda, terceira := divida.Parcelas[0], divida.Parcelas[1], divida.Parcelas[2]
	primeira.ContaAmortizacao.Status = domainsolicitacao.StatusPago
	primeira.ContaJuros.Status = domainsolicitacao.StatusPago
	segunda.ContaJuros.Status = domainsolicitacao.StatusPago
	segunda.ContaAmortizacao.Status = domainsolicitacao.StatusAprovado

	renegociacao, err := divida.Renegociar("acordo com o banco", dadosDaRenegociacao(), uuid.New(), agoraDoTeste)
	if err != nil {
		t.Fatalf("Renegociar falhou: %v", err)
	}

	if divida.Status != domaindivida.StatusRenegociada || *divida.Motivo != "acordo com o banco" || divida.SaldoDevedor() != 0 {
		t.Fatalf("original = %+v", divida)
	}
	if renegociacao.SaldoEmAberto != 200000 || !strings.Contains(renegociacao.DetalheOriginal, "R$ 2000,00") ||
		!strings.Contains(renegociacao.DetalheSubstituta, "Capital de giro Sicredi") {
		t.Errorf("renegociação = %+v", renegociacao)
	}
	if len(renegociacao.Contas) != 1 || renegociacao.Contas[0].SolicitacaoID != segunda.ContaAmortizacao.SolicitacaoID ||
		renegociacao.Contas[0].StatusAnterior != domainsolicitacao.StatusAprovado {
		t.Fatalf("contas canceladas = %+v", renegociacao.Contas)
	}
	if primeira.Status != domaindivida.StatusParcelaGerada || segunda.Status != domaindivida.StatusParcelaGerada ||
		terceira.Status != domaindivida.StatusParcelaCancelada || len(renegociacao.Parcelas) != 3 {
		t.Errorf("parcelas: %s %s %s", primeira.Status, segunda.Status, terceira.Status)
	}

	substituta := renegociacao.Substituta
	if substituta.ID == divida.ID || substituta.DividaOrigemID == nil || *substituta.DividaOrigemID != divida.ID ||
		substituta.EmpresaID != divida.EmpresaID || substituta.Status != domaindivida.StatusAtiva || substituta.SaldoDevedor() != 250000 {
		t.Fatalf("substituta = %+v", substituta)
	}

	if _, err := divida.Renegociar("de novo", dadosDaRenegociacao(), uuid.New(), agoraDoTeste); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("renegociar duas vezes: %v", err)
	}
}

func TestRenegociarValida(t *testing.T) {
	divida := novaDivida(t)

	if _, err := divida.Renegociar(" ", dadosDaRenegociacao(), uuid.New(), agoraDoTeste); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("sem motivo: %v", err)
	}
	dados := dadosDaRenegociacao()
	dados.DataSaldo = data(2026, 9, 1)
	if _, err := divida.Renegociar("acordo", dados, uuid.New(), agoraDoTeste); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("data do saldo antes da original: %v", err)
	}
	dados = dadosDaRenegociacao()
	dados.SaldoInicialCentavos = 1
	if _, err := divida.Renegociar("acordo", dados, uuid.New(), agoraDoTeste); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("cronograma que não fecha: %v", err)
	}
	if divida.Status != domaindivida.StatusAtiva {
		t.Errorf("renegociação recusada mudou a original: %s", divida.Status)
	}
}

func TestReabrirParcelaComContasEncerradas(t *testing.T) {
	divida := novaDivida(t)
	if _, err := divida.GerarContasAte(data(2026, 10, 31), uuid.New(), agoraDoTeste); err != nil {
		t.Fatalf("GerarContasAte falhou: %v", err)
	}
	primeira := divida.Parcelas[0]

	if _, _, err := divida.ReabrirParcela(divida.Parcelas[1].ID, agoraDoTeste); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("reabrir parcela prevista: %v", err)
	}
	primeira.ContaJuros.Status = domainsolicitacao.StatusRejeitado
	if _, _, err := divida.ReabrirParcela(primeira.ID, agoraDoTeste); !errors.Is(err, domain.ErrConflito) {
		t.Fatalf("reabrir com conta aberta: %v", err)
	}
	primeira.ContaAmortizacao.Status = domainsolicitacao.StatusCancelado

	antiga, nova, err := divida.ReabrirParcela(primeira.ID, agoraDoTeste)
	if err != nil {
		t.Fatalf("ReabrirParcela falhou: %v", err)
	}
	if antiga.Status != domaindivida.StatusParcelaCancelada || antiga.SubstitutaID == nil || *antiga.SubstitutaID != nova.ID {
		t.Fatalf("antiga = %+v", antiga)
	}
	if nova.Numero != 1 || nova.Status != domaindivida.StatusParcelaPrevista || !nova.Vencimento.Equal(antiga.Vencimento) ||
		nova.AmortizacaoCentavos != 100000 || nova.JurosCentavos != 5000 || nova.ContaJuros != nil {
		t.Fatalf("nova = %+v", nova)
	}
	if len(divida.Parcelas) != 4 || divida.Parcelas[0] != antiga || divida.Parcelas[1] != nova || divida.SaldoDevedor() != 300000 {
		t.Fatalf("parcelas depois de reabrir = %d, saldo %d", len(divida.Parcelas), divida.SaldoDevedor())
	}

	geracao, err := divida.GerarContasAte(data(2026, 10, 31), uuid.New(), agoraDoTeste)
	if err != nil || len(geracao.Parcelas) != 1 || geracao.Parcelas[0] != nova {
		t.Fatalf("geração da reaberta = %+v, %v", geracao, err)
	}
}
