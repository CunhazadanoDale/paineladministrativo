package divida_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaindivida "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/divida"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

var agoraDoTeste = time.Date(2026, 10, 20, 15, 0, 0, 0, time.UTC)

func data(ano int, mes time.Month, dia int) time.Time {
	return time.Date(ano, mes, dia, 0, 0, 0, 0, time.UTC)
}

func dadosDeDivida() domaindivida.DadosDivida {
	return domaindivida.DadosDivida{
		DadosInformativos: domaindivida.DadosInformativos{
			CredorTipo:             domaindivida.CredorBanco,
			Descricao:              "Capital de giro Sicredi",
			CategoriaAmortizacaoID: uuid.New(),
			CategoriaJurosID:       uuid.New(),
		},
		FornecedorID:         uuid.New(),
		SaldoInicialCentavos: 300000,
		DataSaldo:            data(2026, 9, 30),
		Parcelas: []domaindivida.ParcelaInput{
			{Vencimento: data(2026, 12, 10), AmortizacaoCentavos: 100000, JurosCentavos: 3000},
			{Vencimento: data(2026, 10, 10), AmortizacaoCentavos: 100000, JurosCentavos: 5000},
			{Vencimento: data(2026, 11, 10), AmortizacaoCentavos: 100000, JurosCentavos: 4000},
		},
	}
}

func novaDivida(t *testing.T) *domaindivida.Divida {
	t.Helper()

	divida, err := domaindivida.NovaDivida(uuid.New(), uuid.New(), dadosDeDivida(), agoraDoTeste)
	if err != nil {
		t.Fatalf("NovaDivida falhou: %v", err)
	}

	return divida
}

func TestNovaDividaNumeraParcelasPeloVencimento(t *testing.T) {
	divida := novaDivida(t)

	if divida.Status != domaindivida.StatusAtiva || divida.Versao != 1 || divida.Indexador != domaindivida.IndexadorPrefixado {
		t.Fatalf("dívida = %+v", divida)
	}
	if len(divida.Parcelas) != 3 || divida.TotalDeParcelas() != 3 {
		t.Fatalf("parcelas = %d", len(divida.Parcelas))
	}
	for i, esperado := range []time.Time{data(2026, 10, 10), data(2026, 11, 10), data(2026, 12, 10)} {
		parcela := divida.Parcelas[i]
		if parcela.Numero != i+1 || !parcela.Vencimento.Equal(esperado) || parcela.Status != domaindivida.StatusParcelaPrevista {
			t.Errorf("parcela %d = %+v", i, parcela)
		}
	}
	if divida.SaldoDevedor() != 300000 || divida.Quitada() {
		t.Errorf("saldo = %d, quitada = %v", divida.SaldoDevedor(), divida.Quitada())
	}
}

func TestNovaDividaValida(t *testing.T) {
	casos := map[string]func(*domaindivida.DadosDivida){
		"sem credor":             func(d *domaindivida.DadosDivida) { d.FornecedorID = uuid.Nil },
		"sem tipo de credor":     func(d *domaindivida.DadosDivida) { d.CredorTipo = "" },
		"sem descrição":          func(d *domaindivida.DadosDivida) { d.Descricao = " " },
		"descrição longa":        func(d *domaindivida.DadosDivida) { d.Descricao = strings.Repeat("a", 121) },
		"saldo zero":             func(d *domaindivida.DadosDivida) { d.SaldoInicialCentavos = 0 },
		"sem data do saldo":      func(d *domaindivida.DadosDivida) { d.DataSaldo = time.Time{} },
		"data do saldo futura":   func(d *domaindivida.DadosDivida) { d.DataSaldo = data(2026, 10, 21) },
		"sem parcelas":           func(d *domaindivida.DadosDivida) { d.Parcelas = nil },
		"parcela antes do corte": func(d *domaindivida.DadosDivida) { d.Parcelas[0].Vencimento = data(2026, 9, 30) },
		"vencimento repetido":    func(d *domaindivida.DadosDivida) { d.Parcelas[0].Vencimento = data(2026, 10, 10) },
		"amortização negativa":   func(d *domaindivida.DadosDivida) { d.Parcelas[0].AmortizacaoCentavos = -1 },
		"parcela zerada": func(d *domaindivida.DadosDivida) {
			d.Parcelas[0].AmortizacaoCentavos, d.Parcelas[0].JurosCentavos = 0, 0
		},
		"soma diferente do saldo":       func(d *domaindivida.DadosDivida) { d.SaldoInicialCentavos = 300001 },
		"sem categoria de juros":        func(d *domaindivida.DadosDivida) { d.CategoriaJurosID = uuid.Nil },
		"sem categoria de amortização":  func(d *domaindivida.DadosDivida) { d.CategoriaAmortizacaoID = uuid.Nil },
		"conta sem data de liberação":   func(d *domaindivida.DadosDivida) { conta := uuid.New(); d.ContaLiberacaoID = &conta },
		"liberação futura":              func(d *domaindivida.DadosDivida) { futura := data(2026, 11, 1); d.LiberadaEm = &futura },
		"valor original zero":           func(d *domaindivida.DadosDivida) { zero := int64(0); d.ValorOriginalCentavos = &zero },
		"contratação no futuro":         func(d *domaindivida.DadosDivida) { futura := data(2027, 1, 1); d.ContratadaEm = &futura },
		"parcela acima do limite":       func(d *domaindivida.DadosDivida) { d.Parcelas[0].JurosCentavos = 1_000_000_000_000_001 },
		"vencimento da parcela ausente": func(d *domaindivida.DadosDivida) { d.Parcelas[0].Vencimento = time.Time{} },
	}

	for nome, alterar := range casos {
		t.Run(nome, func(t *testing.T) {
			dados := dadosDeDivida()
			alterar(&dados)
			if _, err := domaindivida.NovaDivida(uuid.New(), uuid.New(), dados, agoraDoTeste); !errors.Is(err, domain.ErrValidacao) {
				t.Fatalf("erro = %v", err)
			}
		})
	}
}

func TestDataDoSaldoHojeEmBrasilia(t *testing.T) {
	dados := dadosDeDivida()
	dados.DataSaldo = data(2026, 10, 20)
	dados.Parcelas = []domaindivida.ParcelaInput{{Vencimento: data(2026, 10, 21), AmortizacaoCentavos: 300000}}
	madrugadaUTC := time.Date(2026, 10, 21, 2, 0, 0, 0, time.UTC)

	if _, err := domaindivida.NovaDivida(uuid.New(), uuid.New(), dados, madrugadaUTC); err != nil {
		t.Fatalf("data do saldo de hoje em Brasília recusada: %v", err)
	}
	dados.DataSaldo = data(2026, 10, 21)
	dados.Parcelas[0].Vencimento = data(2026, 10, 22)
	if _, err := domaindivida.NovaDivida(uuid.New(), uuid.New(), dados, madrugadaUTC); !errors.Is(err, domain.ErrValidacao) {
		t.Fatalf("21/10 ainda é amanhã em Brasília: %v", err)
	}
}

func TestGerarContasAteOFimDoMes(t *testing.T) {
	divida := novaDivida(t)
	usuario := uuid.New()

	geracao, err := divida.GerarContasAte(domaindivida.FimDoMes(domaindivida.Hoje(agoraDoTeste)), usuario, agoraDoTeste)
	if err != nil {
		t.Fatalf("GerarContasAte falhou: %v", err)
	}
	if len(geracao.Parcelas) != 1 || len(geracao.Contas) != 2 || geracao.Parcelas[0].Numero != 1 {
		t.Fatalf("geração = %+v", geracao)
	}
	parcela := divida.Parcelas[0]
	if parcela.Status != domaindivida.StatusParcelaGerada || parcela.GeradaEm == nil || parcela.ContaAmortizacao == nil || parcela.ContaJuros == nil {
		t.Fatalf("parcela gerada = %+v", parcela)
	}

	amortizacao, juros := geracao.Contas[0].Solicitacao, geracao.Contas[1].Solicitacao
	if amortizacao.Origem != domainsolicitacao.OrigemDividaAmortizacao || *amortizacao.OrigemID != parcela.ID ||
		amortizacao.Valor.Centavos() != 100000 || *amortizacao.CategoriaFinanceiraID != divida.CategoriaAmortizacaoID ||
		*amortizacao.FornecedorID != divida.FornecedorID || amortizacao.EmpresaID != divida.EmpresaID || amortizacao.SolicitanteID != usuario {
		t.Errorf("conta de amortização = %+v", amortizacao)
	}
	if juros.Origem != domainsolicitacao.OrigemDividaJuros || *juros.OrigemID != parcela.ID || juros.Valor.Centavos() != 5000 ||
		*juros.CategoriaFinanceiraID != divida.CategoriaJurosID || juros.Competencia.String() != "2026-10" ||
		juros.Prazo.String() != "2026-10-10" || juros.Status != domainsolicitacao.StatusPendenteAprovacao {
		t.Errorf("conta de juros = %+v", juros)
	}
	if !strings.Contains(juros.Observacao.Texto(), "parcela 1/3 — juros") {
		t.Errorf("observação = %q", juros.Observacao.Texto())
	}

	deNovo, err := divida.GerarContasAte(domaindivida.FimDoMes(data(2026, 10, 1)), usuario, agoraDoTeste)
	if err != nil || !deNovo.Vazia() {
		t.Fatalf("segunda geração do mês = %+v, %v", deNovo, err)
	}
	ateDezembro, err := divida.GerarContasAte(data(2026, 12, 31), usuario, agoraDoTeste)
	if err != nil || len(ateDezembro.Parcelas) != 2 || len(ateDezembro.Contas) != 4 {
		t.Fatalf("geração até dezembro = %+v, %v", ateDezembro, err)
	}
}

func TestGerarContasSemComponenteZerado(t *testing.T) {
	dados := dadosDeDivida()
	dados.SaldoInicialCentavos = 200000
	dados.Parcelas = []domaindivida.ParcelaInput{
		{Vencimento: data(2026, 10, 10), JurosCentavos: 5000},
		{Vencimento: data(2026, 10, 25), AmortizacaoCentavos: 200000},
	}
	divida, err := domaindivida.NovaDivida(uuid.New(), uuid.New(), dados, agoraDoTeste)
	if err != nil {
		t.Fatalf("NovaDivida falhou: %v", err)
	}

	geracao, err := divida.GerarContasAte(data(2026, 10, 31), uuid.New(), agoraDoTeste)
	if err != nil || len(geracao.Contas) != 2 {
		t.Fatalf("geração = %+v, %v", geracao, err)
	}
	if divida.Parcelas[0].ContaAmortizacao != nil || divida.Parcelas[0].ContaJuros == nil ||
		divida.Parcelas[1].ContaJuros != nil || divida.Parcelas[1].ContaAmortizacao == nil {
		t.Errorf("contas por componente: %+v %+v", divida.Parcelas[0], divida.Parcelas[1])
	}
}

func TestSituacaoESaldoPelasContas(t *testing.T) {
	divida := novaDivida(t)
	if _, err := divida.GerarContasAte(data(2026, 11, 30), uuid.New(), agoraDoTeste); err != nil {
		t.Fatalf("GerarContasAte falhou: %v", err)
	}
	hoje := data(2026, 11, 15)
	primeira, segunda, terceira := divida.Parcelas[0], divida.Parcelas[1], divida.Parcelas[2]

	primeira.ContaAmortizacao.Status = domainsolicitacao.StatusPago
	if primeira.Situacao(hoje) != domaindivida.SituacaoVencida || primeira.JurosEmAberto() != 5000 || primeira.AmortizacaoEmAberto() != 0 {
		t.Errorf("parcela com só a amortização paga: %s", primeira.Situacao(hoje))
	}
	primeira.ContaJuros.Status = domainsolicitacao.StatusPago
	if primeira.Situacao(hoje) != domaindivida.SituacaoPaga {
		t.Errorf("parcela paga: %s", primeira.Situacao(hoje))
	}
	segunda.ContaJuros.Status = domainsolicitacao.StatusRejeitado
	if segunda.Situacao(hoje) != domaindivida.SituacaoContaCancelada {
		t.Errorf("parcela com conta rejeitada: %s", segunda.Situacao(hoje))
	}
	if terceira.Situacao(hoje) != domaindivida.SituacaoAVencer {
		t.Errorf("parcela prevista: %s", terceira.Situacao(hoje))
	}
	if divida.SaldoDevedor() != 200000 || divida.Quitada() {
		t.Errorf("saldo = %d", divida.SaldoDevedor())
	}
}

func TestAlterarParcelaPrevista(t *testing.T) {
	divida := novaDivida(t)
	segunda := divida.Parcelas[1]

	parcela, detalhe, err := divida.AlterarParcela(segunda.ID, data(2026, 11, 15), 4500, agoraDoTeste)
	if err != nil || parcela.JurosCentavos != 4500 || !parcela.Vencimento.Equal(data(2026, 11, 15)) || !strings.Contains(detalhe, "parcela 2") {
		t.Fatalf("parcela alterada = %+v, %q, %v", parcela, detalhe, err)
	}

	for nome, vencimento := range map[string]time.Time{
		"antes da anterior": data(2026, 10, 10),
		"depois da próxima": data(2026, 12, 10),
	} {
		if _, _, err := divida.AlterarParcela(segunda.ID, vencimento, 4500, agoraDoTeste); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("%s: erro = %v", nome, err)
		}
	}
	if _, _, err := divida.AlterarParcela(uuid.New(), data(2026, 11, 15), 1, agoraDoTeste); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("parcela inexistente: %v", err)
	}

	if _, err := divida.GerarContasAte(data(2026, 11, 30), uuid.New(), agoraDoTeste); err != nil {
		t.Fatalf("GerarContasAte falhou: %v", err)
	}
	if _, _, err := divida.AlterarParcela(segunda.ID, data(2026, 11, 16), 4500, agoraDoTeste); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("parcela gerada alterada: %v", err)
	}
}

func TestCancelarDivida(t *testing.T) {
	divida := novaDivida(t)
	if _, err := divida.GerarContasAte(data(2026, 10, 31), uuid.New(), agoraDoTeste); err != nil {
		t.Fatalf("GerarContasAte falhou: %v", err)
	}

	if _, _, err := divida.Cancelar(" ", agoraDoTeste); !errors.Is(err, domain.ErrValidacao) {
		t.Fatalf("cancelar sem motivo: %v", err)
	}
	divida.Parcelas[0].ContaJuros.Status = domainsolicitacao.StatusPago
	if _, _, err := divida.Cancelar("lançada em duplicidade", agoraDoTeste); !errors.Is(err, domain.ErrConflito) {
		t.Fatalf("cancelar com conta paga: %v", err)
	}
	divida.Parcelas[0].ContaJuros.Status = domainsolicitacao.StatusAprovado

	parcelas, contas, err := divida.Cancelar("lançada em duplicidade", agoraDoTeste)
	if err != nil {
		t.Fatalf("Cancelar falhou: %v", err)
	}
	if divida.Status != domaindivida.StatusCancelada || *divida.Motivo != "lançada em duplicidade" || divida.SaldoDevedor() != 0 {
		t.Fatalf("dívida cancelada = %+v", divida)
	}
	if len(parcelas) != 3 || len(contas) != 2 || contas[1].StatusAnterior != domainsolicitacao.StatusAprovado {
		t.Fatalf("parcelas = %d, contas = %+v", len(parcelas), contas)
	}
	for _, parcela := range divida.Parcelas {
		if parcela.Status != domaindivida.StatusParcelaCancelada {
			t.Errorf("parcela %d = %s", parcela.Numero, parcela.Status)
		}
	}
	if divida.Parcelas[0].ContaAmortizacao.Status != domainsolicitacao.StatusCancelado {
		t.Errorf("conta cancelada em memória = %s", divida.Parcelas[0].ContaAmortizacao.Status)
	}

	if err := divida.Alterar(divida.DadosInformativosAtuais(), agoraDoTeste); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("alterar dívida cancelada: %v", err)
	}
	if _, _, err := divida.Cancelar("de novo", agoraDoTeste); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("cancelar duas vezes: %v", err)
	}
	if geracao, err := divida.GerarContasAte(data(2027, 12, 31), uuid.New(), agoraDoTeste); err != nil || !geracao.Vazia() {
		t.Errorf("gerar em dívida cancelada = %+v, %v", geracao, err)
	}
}

func TestConferirVersao(t *testing.T) {
	divida := novaDivida(t)

	if err := divida.ConferirVersao(0); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("versão ausente: %v", err)
	}
	if err := divida.ConferirVersao(2); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("versão divergente: %v", err)
	}
	if err := divida.ConferirVersao(1); err != nil {
		t.Errorf("versão certa: %v", err)
	}
}

func TestTaxaEIndexador(t *testing.T) {
	bp := 185
	taxa, err := domaindivida.NovaTaxa(&bp, " Mensal ")
	if err != nil || taxa.PontosBase() != 185 || taxa.Periodo() != domaindivida.PeriodoMensal {
		t.Fatalf("taxa = %+v, %v", taxa, err)
	}
	if semTaxa, err := domaindivida.NovaTaxa(nil, ""); semTaxa != nil || err != nil {
		t.Errorf("sem taxa = %+v, %v", semTaxa, err)
	}
	negativa, acima := -1, 100001
	for nome, caso := range map[string]struct {
		bp      *int
		periodo string
	}{
		"sem período": {&bp, ""}, "sem taxa": {nil, "anual"}, "negativa": {&negativa, "anual"},
		"acima do limite": {&acima, "anual"}, "período inválido": {&bp, "diario"},
	} {
		if _, err := domaindivida.NovaTaxa(caso.bp, caso.periodo); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("%s: %v", nome, err)
		}
	}

	if indexador, err := domaindivida.NovoIndexador(""); err != nil || indexador != domaindivida.IndexadorPrefixado {
		t.Errorf("indexador padrão = %s, %v", indexador, err)
	}
	if _, err := domaindivida.NovoIndexador("igpm"); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("indexador inválido: %v", err)
	}
	if _, err := domaindivida.NovoTipoCredor("agiota"); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("credor inválido: %v", err)
	}
	if tipo, err := domaindivida.NovoTipoCredor(" FISCO "); err != nil || tipo != domaindivida.CredorFisco {
		t.Errorf("credor = %s, %v", tipo, err)
	}
}
