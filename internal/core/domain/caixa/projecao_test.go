package caixa_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
)

func TestProjetarSaldoDiario(t *testing.T) {
	hoje := data(2026, 10, 19)
	projecao := domaincaixa.Projetar(domaincaixa.DadosProjecao{
		Hoje:    hoje,
		Ate:     data(2026, 10, 22),
		Partida: domaincaixa.SaldoDePartida{ExtratoCentavos: 100000, EmTransitoCentavos: -5000},
		Entradas: []domaincaixa.Previsto{
			{Data: data(2026, 10, 10), ValorCentavos: 2000},
			{Data: data(2026, 10, 20), ValorCentavos: 30000},
			{Data: data(2026, 11, 5), ValorCentavos: 99999},
		},
		Saidas: []domaincaixa.Previsto{
			{Data: data(2026, 10, 1), ValorCentavos: 1000},
			{Data: data(2026, 10, 19), ValorCentavos: 4000},
			{Data: data(2026, 10, 21), ValorCentavos: 7000, Estimado: true},
		},
		Recorrentes: []domaincaixa.Previsto{
			{Data: data(2026, 10, 15), ValorCentavos: 300},
			{Data: data(2026, 10, 22), ValorCentavos: 800},
		},
	})

	if len(projecao.Dias) != 4 || !projecao.De.Equal(hoje) || !projecao.Ate.Equal(data(2026, 10, 22)) {
		t.Fatalf("projeção = %+v", projecao)
	}

	primeiro := projecao.Dias[0]
	if primeiro.EntradasVencidasCentavos != 2000 || primeiro.SaidasVencidasCentavos != 1000 || primeiro.SaidasCentavos != 4000 ||
		primeiro.RecorrentesCentavos != 300 || primeiro.SaldoCentavos != 91700 {
		t.Errorf("hoje = %+v", primeiro)
	}
	esperados := []int64{91700, 121700, 114700, 113900}
	for i, esperado := range esperados {
		if projecao.Dias[i].SaldoCentavos != esperado {
			t.Errorf("dia %d: saldo %d, esperado %d", i, projecao.Dias[i].SaldoCentavos, esperado)
		}
	}
	if projecao.Dias[2].SaidasEstimadasCentavos != 7000 || projecao.SaldoFinalCentavos() != 113900 {
		t.Errorf("estimadas e saldo final = %+v, %d", projecao.Dias[2], projecao.SaldoFinalCentavos())
	}
}

func TestHorizonteDaProjecao(t *testing.T) {
	hoje := data(2026, 10, 19)

	if padrao, err := domaincaixa.HorizonteDaProjecao(hoje, nil); err != nil || !padrao.Equal(data(2026, 11, 18)) {
		t.Errorf("padrão = %v, %v", padrao, err)
	}
	limite := data(2027, 1, 17)
	if fim, err := domaincaixa.HorizonteDaProjecao(hoje, &limite); err != nil || !fim.Equal(limite) {
		t.Errorf("limite = %v, %v", fim, err)
	}
	for _, invalido := range []time.Time{data(2026, 10, 18), data(2027, 1, 18)} {
		if _, err := domaincaixa.HorizonteDaProjecao(hoje, &invalido); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("%v = %v", invalido, err)
		}
	}
}

func TestPeriodoDoMovimentoDiario(t *testing.T) {
	hoje := data(2026, 10, 19)

	de, ate, err := domaincaixa.PeriodoDoMovimentoDiario(hoje, nil, nil)
	if err != nil || !de.Equal(data(2026, 10, 1)) || !ate.Equal(hoje) {
		t.Errorf("padrão = %v a %v, %v", de, ate, err)
	}

	inicio, fim := data(2026, 7, 1), data(2026, 10, 1)
	if _, _, err := domaincaixa.PeriodoDoMovimentoDiario(hoje, &inicio, &fim); err != nil {
		t.Errorf("92 dias = %v", err)
	}
	inicio = data(2026, 6, 30)
	if _, _, err := domaincaixa.PeriodoDoMovimentoDiario(hoje, &inicio, &fim); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("93 dias = %v", err)
	}
	inicio = data(2026, 10, 2)
	if _, _, err := domaincaixa.PeriodoDoMovimentoDiario(hoje, &inicio, &fim); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("invertido = %v", err)
	}
}
