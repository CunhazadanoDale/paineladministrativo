package dre_test

import (
	"testing"
	"time"

	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	domaindivida "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/divida"
	domaindre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/dre"
	"github.com/google/uuid"
)

func centavos(valor int64) *int64 {
	return &valor
}

func TestRelatorioSemanalSomaOsBlocosESaldos(t *testing.T) {
	hoje := time.Date(2026, time.October, 21, 0, 0, 0, 0, time.UTC)
	semana := domaindre.SemanaDe(time.Date(2026, time.October, 14, 0, 0, 0, 0, time.UTC))
	corte := time.Date(2026, time.September, 30, 0, 0, 0, 0, time.UTC)

	relatorio := domaindre.NovoRelatorioSemanal(semana, hoje, domaindre.ConteudoDoRelatorio{
		Notas: []domaindre.NotasEmitidas{
			{EmpresaID: empresaA, Natureza: "venda", Total: domaindre.TotalSemanal{Quantidade: 2, ValorCentavos: 1000}},
			{EmpresaID: empresaA, Natureza: "remessa", Total: domaindre.TotalSemanal{Quantidade: 1, ValorCentavos: 400}},
			{EmpresaID: empresaB, Natureza: "venda", Total: domaindre.TotalSemanal{Quantidade: 1, ValorCentavos: 500}},
		},
		Pagamentos: []domaindre.PagamentosFeitos{
			{EmpresaID: empresaA, Total: domaindre.TotalSemanal{Quantidade: 1, ValorCentavos: 300}},
			{EmpresaID: empresaB, Total: domaindre.TotalSemanal{Quantidade: 2, ValorCentavos: 200}},
		},
		Recebimentos: []domaindre.RecebimentosFeitos{
			{EmpresaID: empresaA, Tipo: "conta", Total: domaindre.TotalSemanal{Quantidade: 3, ValorCentavos: 900}, JurosCentavos: 10},
		},
		TitulosVencidos: domaindre.NovosVencidos([]domaindre.VencidosPorEmpresa{
			{EmpresaID: empresaA, Total: domaindre.TotalSemanal{Quantidade: 2, ValorCentavos: 700}},
			{EmpresaID: empresaB, Total: domaindre.TotalSemanal{Quantidade: 1, ValorCentavos: 100}},
		}, []domaindre.TituloVencido{{Numero: "NF-1"}}),
		Saldos: []*domaincaixa.SaldoConta{
			{DataCorte: &corte, SaldoInicialCentavos: centavos(1000), MovimentoCentavos: 250},
			{DataCorte: &corte, SaldoInicialCentavos: centavos(-100)},
			{},
		},
	})

	if !relatorio.Inicio.Equal(time.Date(2026, time.October, 12, 0, 0, 0, 0, time.UTC)) ||
		!relatorio.Fim.Equal(time.Date(2026, time.October, 18, 0, 0, 0, 0, time.UTC)) || !relatorio.Hoje.Equal(hoje) {
		t.Errorf("semana = %s a %s, hoje %s", relatorio.Inicio, relatorio.Fim, relatorio.Hoje)
	}
	if relatorio.TotalNotasVenda != (domaindre.TotalSemanal{Quantidade: 3, ValorCentavos: 1500}) {
		t.Errorf("notas de venda = %+v, esperado só as vendas", relatorio.TotalNotasVenda)
	}
	if relatorio.TotalPagamentos != (domaindre.TotalSemanal{Quantidade: 3, ValorCentavos: 500}) {
		t.Errorf("pagamentos = %+v", relatorio.TotalPagamentos)
	}
	if relatorio.TotalRecebido != (domaindre.TotalSemanal{Quantidade: 3, ValorCentavos: 900}) {
		t.Errorf("recebido = %+v", relatorio.TotalRecebido)
	}
	if relatorio.TitulosVencidos.Total != (domaindre.TotalSemanal{Quantidade: 3, ValorCentavos: 800}) || len(relatorio.TitulosVencidos.Itens) != 1 {
		t.Errorf("títulos vencidos = %+v", relatorio.TitulosVencidos)
	}
	if relatorio.Saldos.SaldoTotalCentavos != 1150 || relatorio.Saldos.ContasSemAbertura != 1 || len(relatorio.Saldos.Contas) != 3 {
		t.Errorf("saldos = %+v", relatorio.Saldos)
	}
}

func TestRelatorioSemanalVazioDevolveListasVazias(t *testing.T) {
	relatorio := domaindre.NovoRelatorioSemanal(domaindre.SemanaDe(time.Now()), time.Now(), domaindre.ConteudoDoRelatorio{})

	if relatorio.Notas == nil || relatorio.Pagamentos == nil || relatorio.Recebimentos == nil ||
		relatorio.TitulosVencidos.PorEmpresa == nil || relatorio.TitulosVencidos.Itens == nil ||
		relatorio.ContasVencidas.Itens == nil || relatorio.EstoqueBaixo.Itens == nil || relatorio.Saldos.Contas == nil {
		t.Errorf("relatório vazio com listas nulas: %+v", relatorio)
	}
}

func TestRelatorioSemanalResumeOEndividamentoPeloHojeDoRelatorio(t *testing.T) {
	hoje := time.Date(2026, time.October, 21, 0, 0, 0, 0, time.UTC)
	divida := uuid.New()

	relatorio := domaindre.NovoRelatorioSemanal(domaindre.SemanaDe(hoje), hoje, domaindre.ConteudoDoRelatorio{
		Parcelas: []domaindivida.ParcelaEmAberto{
			{DividaID: divida, EmpresaID: empresaA, CredorTipo: domaindivida.CredorBanco, Vencimento: hoje.AddDate(0, 0, -1), AmortizacaoCentavos: 500, JurosCentavos: 50},
			{DividaID: divida, EmpresaID: empresaA, CredorTipo: domaindivida.CredorBanco, Vencimento: hoje.AddDate(0, 0, 20), AmortizacaoCentavos: 700, JurosCentavos: 40},
		},
	})

	endividamento := relatorio.Endividamento
	if endividamento.SaldoDevedorCentavos != 1200 || endividamento.Dividas != 1 {
		t.Errorf("saldo devedor = %d em %d dívidas", endividamento.SaldoDevedorCentavos, endividamento.Dividas)
	}
	if endividamento.Vencido.AmortizacaoCentavos != 500 || endividamento.AVencer30.AmortizacaoCentavos != 700 {
		t.Errorf("vencido = %+v, a vencer em 30 dias = %+v", endividamento.Vencido, endividamento.AVencer30)
	}
	if len(endividamento.PorEmpresa) != 1 || endividamento.PorEmpresa[0].EmpresaID != empresaA {
		t.Errorf("por empresa = %+v", endividamento.PorEmpresa)
	}
}
