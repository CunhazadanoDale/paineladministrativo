package receber_test

import (
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	"github.com/google/uuid"
)

func TestNovoPeriodoResumoPadraoComecaNaSegundaDeBrasilia(t *testing.T) {
	domingoANoiteEmBrasilia := time.Date(2026, 10, 19, 1, 0, 0, 0, time.UTC)

	periodo, err := domainreceber.NovoPeriodoResumo(nil, nil, domingoANoiteEmBrasilia)
	if err != nil {
		t.Fatalf("NovoPeriodoResumo falhou: %v", err)
	}
	if !periodo.De.Equal(data(2026, 10, 12)) || !periodo.Ate.Equal(data(2026, 12, 6)) {
		t.Fatalf("período = %s a %s", periodo.De, periodo.Ate)
	}
}

func TestNovoPeriodoResumoValida(t *testing.T) {
	de, ate := data(2026, 10, 20), data(2026, 10, 1)
	_, err := domainreceber.NovoPeriodoResumo(&de, &ate, agora)
	esperarErro(t, err, domain.ErrValidacao, "anterior ao início")

	longe := data(2027, 12, 1)
	_, err = domainreceber.NovoPeriodoResumo(&de, &longe, agora)
	esperarErro(t, err, domain.ErrValidacao, "no máximo 1 ano")
}

func TestNovoResumoSomaOsTotais(t *testing.T) {
	empresa := uuid.New()
	resumo := domainreceber.NovoResumoContasAReceber(domainreceber.PeriodoResumo{De: data(2026, 10, 12), Ate: data(2026, 12, 6)},
		[]domainreceber.LinhaResumoSemanal{
			{Semana: data(2026, 10, 12), EmpresaID: empresa, AReceber: domainreceber.TotalResumo{Quantidade: 1, SaldoCentavos: 100},
				Vencido: domainreceber.TotalResumo{Quantidade: 1, SaldoCentavos: 30}},
			{Semana: data(2026, 10, 19), EmpresaID: empresa, AReceber: domainreceber.TotalResumo{Quantidade: 2, SaldoCentavos: 200},
				Antecipado: domainreceber.TotalResumo{Quantidade: 1, SaldoCentavos: 50}},
		},
		[]domainreceber.AtrasoPorEmpresa{{EmpresaID: empresa, Vencido: domainreceber.TotalResumo{Quantidade: 3, SaldoCentavos: 70}}},
	)

	if resumo.Totais.AReceber != (domainreceber.TotalResumo{Quantidade: 3, SaldoCentavos: 300}) ||
		resumo.Totais.Antecipado != (domainreceber.TotalResumo{Quantidade: 1, SaldoCentavos: 50}) ||
		resumo.Totais.Vencido != (domainreceber.TotalResumo{Quantidade: 4, SaldoCentavos: 100}) {
		t.Fatalf("totais = %+v", resumo.Totais)
	}
}

func TestLimiteDoAlertaDeBoleto(t *testing.T) {
	limite, err := domainreceber.LimiteDoAlertaDeBoleto(nil, agora)
	if err != nil || !limite.Equal(data(2026, 10, 20)) {
		t.Fatalf("limite padrão = %s, %v", limite, err)
	}

	zero := 0
	if limite, err := domainreceber.LimiteDoAlertaDeBoleto(&zero, agora); err != nil || !limite.Equal(data(2026, 10, 15)) {
		t.Fatalf("limite de hoje = %s, %v", limite, err)
	}

	demais := 91
	_, err = domainreceber.LimiteDoAlertaDeBoleto(&demais, agora)
	esperarErro(t, err, domain.ErrValidacao, "entre 0 e 90")
}
