package solicitacao_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

func data(ano int, mes time.Month, dia int) time.Time {
	return time.Date(ano, mes, dia, 0, 0, 0, 0, time.UTC)
}

func TestInicioDaSemanaEhASegundaFeira(t *testing.T) {
	casos := map[time.Time]time.Time{
		data(2026, time.October, 14): data(2026, time.October, 12),
		data(2026, time.October, 12): data(2026, time.October, 12),
		data(2026, time.October, 18): data(2026, time.October, 12),
		data(2026, time.November, 1): data(2026, time.October, 26),
	}

	for dia, esperado := range casos {
		if obtido := domainsolicitacao.InicioDaSemana(dia.Add(15 * time.Hour)); !obtido.Equal(esperado) {
			t.Errorf("início da semana de %s = %s, esperado %s", dia.Format("2006-01-02"), obtido.Format("2006-01-02"), esperado.Format("2006-01-02"))
		}
	}
}

func TestNovoPeriodoResumoUsaOitoSemanasPorPadrao(t *testing.T) {
	periodo, err := domainsolicitacao.NovoPeriodoResumo(nil, nil, data(2026, time.October, 14))
	if err != nil {
		t.Fatalf("período falhou: %v", err)
	}
	if !periodo.De.Equal(data(2026, time.October, 12)) || !periodo.Ate.Equal(data(2026, time.December, 6)) {
		t.Errorf("período = %s a %s", periodo.De.Format("2006-01-02"), periodo.Ate.Format("2006-01-02"))
	}

	de := data(2026, time.March, 1)
	soComInicio, err := domainsolicitacao.NovoPeriodoResumo(&de, nil, data(2026, time.October, 14))
	if err != nil || !soComInicio.Ate.Equal(data(2026, time.April, 25)) {
		t.Errorf("período só com início = %+v, %v", soComInicio, err)
	}
}

func TestNovoPeriodoResumoRecusaIntervaloInvertidoOuLongo(t *testing.T) {
	de := data(2026, time.March, 1)
	antes := data(2026, time.February, 1)
	longe := data(2027, time.March, 3)
	umAno := data(2027, time.March, 1)

	if _, err := domainsolicitacao.NovoPeriodoResumo(&de, &antes, time.Now()); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("invertido = %v", err)
	}
	if _, err := domainsolicitacao.NovoPeriodoResumo(&de, &longe, time.Now()); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("mais de um ano = %v", err)
	}
	if _, err := domainsolicitacao.NovoPeriodoResumo(&de, &umAno, time.Now()); err != nil {
		t.Errorf("um ano exato = %v, esperado aceito", err)
	}
}

func TestNovoResumoSomaOsTotaisComAsAtrasadas(t *testing.T) {
	empresa := uuid.New()
	periodo := domainsolicitacao.PeriodoResumo{De: data(2026, time.October, 12), Ate: data(2026, time.October, 25)}

	resumo := domainsolicitacao.NovoResumoContasAPagar(periodo, []domainsolicitacao.LinhaResumoSemanal{
		{Semana: data(2026, time.October, 12), EmpresaID: empresa,
			Previsto: domainsolicitacao.TotalResumo{Quantidade: 1, ValorCentavos: 100},
			APagar:   domainsolicitacao.TotalResumo{Quantidade: 2, ValorCentavos: 300},
			Vencido:  domainsolicitacao.TotalResumo{Quantidade: 1, ValorCentavos: 50}, EstimadoCentavos: 100},
		{Semana: data(2026, time.October, 19), EmpresaID: empresa,
			APagar: domainsolicitacao.TotalResumo{Quantidade: 1, ValorCentavos: 700}, EstimadoCentavos: 20},
	}, []domainsolicitacao.AtrasoPorEmpresa{
		{EmpresaID: empresa, Vencido: domainsolicitacao.TotalResumo{Quantidade: 3, ValorCentavos: 900}},
	})

	totais := resumo.Totais
	if totais.Previsto != (domainsolicitacao.TotalResumo{Quantidade: 1, ValorCentavos: 100}) ||
		totais.APagar != (domainsolicitacao.TotalResumo{Quantidade: 3, ValorCentavos: 1000}) ||
		totais.Vencido != (domainsolicitacao.TotalResumo{Quantidade: 4, ValorCentavos: 950}) ||
		totais.EstimadoCentavos != 120 {
		t.Errorf("totais = %+v", totais)
	}
}
