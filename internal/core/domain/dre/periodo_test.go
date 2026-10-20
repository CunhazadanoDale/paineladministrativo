package dre_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaindre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/dre"
)

func mes(t *testing.T, texto string) domaindre.Mes {
	t.Helper()

	valor, err := domaindre.NovoMes(texto, "mês")
	if err != nil {
		t.Fatalf("NovoMes(%q) falhou: %v", texto, err)
	}

	return valor
}

func TestNovoMesAceitaAnoEMes(t *testing.T) {
	valor := mes(t, " 2026-02 ")

	if valor.String() != "2026-02" {
		t.Errorf("mês = %s, esperado 2026-02", valor)
	}
	if !valor.Inicio().Equal(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("início = %s", valor.Inicio())
	}
	if !valor.Fim().Equal(time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("fim = %s", valor.Fim())
	}
}

func TestNovoMesRejeitaFormatoEIntervaloInvalidos(t *testing.T) {
	for _, texto := range []string{"", "2026-13", "2026-10-01", "10/2026", "1999-12", "2100-01"} {
		if _, err := domaindre.NovoMes(texto, "de"); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("NovoMes(%q) = %v, esperado erro de validação", texto, err)
		}
	}
}

func TestMesDeIgnoraDiaEFuso(t *testing.T) {
	brasilia := time.FixedZone("BRT", -3*60*60)
	valor := domaindre.MesDe(time.Date(2026, time.March, 31, 22, 0, 0, 0, brasilia))

	if valor != mes(t, "2026-03") {
		t.Errorf("mês = %s, esperado 2026-03", valor)
	}
}

func TestPeriodoDaDREPadraoSaoOsDozeMesesAteOMesCorrente(t *testing.T) {
	periodo, err := domaindre.NovoPeriodoDaDRE(nil, nil, time.Date(2026, time.October, 20, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NovoPeriodoDaDRE falhou: %v", err)
	}

	if periodo.De().String() != "2025-11" || periodo.Ate().String() != "2026-10" {
		t.Errorf("período = %s a %s, esperado 2025-11 a 2026-10", periodo.De(), periodo.Ate())
	}
	meses := periodo.Meses()
	if len(meses) != 12 || meses[0].String() != "2025-11" || meses[11].String() != "2026-10" {
		t.Errorf("meses = %v", meses)
	}
}

func TestPeriodoComSoOFimContaDozeMesesParaTras(t *testing.T) {
	ate := mes(t, "2026-03")

	periodo, err := domaindre.NovoPeriodoDaDRE(nil, &ate, time.Date(2026, time.October, 20, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NovoPeriodoDaDRE falhou: %v", err)
	}

	if periodo.De().String() != "2025-04" {
		t.Errorf("início = %s, esperado 2025-04", periodo.De())
	}
}

func TestPeriodoRejeitaFimAntesDoInicio(t *testing.T) {
	de, ate := mes(t, "2026-05"), mes(t, "2026-04")

	if _, err := domaindre.NovoPeriodoDaDRE(&de, &ate, time.Now()); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro = %v, esperado erro de validação", err)
	}
}

func TestPeriodoRespeitaOMaximoDeCadaPainel(t *testing.T) {
	de, ate := mes(t, "2024-01"), mes(t, "2025-12")
	if _, err := domaindre.NovoPeriodoDaDRE(&de, &ate, time.Now()); err != nil {
		t.Errorf("24 meses na DRE deveriam passar: %v", err)
	}

	ate = mes(t, "2026-01")
	if _, err := domaindre.NovoPeriodoDaDRE(&de, &ate, time.Now()); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("25 meses na DRE = %v, esperado erro de validação", err)
	}
	if _, err := domaindre.NovoPeriodoDoFaturamento(&de, &ate, time.Now()); err != nil {
		t.Errorf("25 meses no faturamento deveriam passar: %v", err)
	}

	de, ate = mes(t, "2021-01"), mes(t, "2026-01")
	if _, err := domaindre.NovoPeriodoDoFaturamento(&de, &ate, time.Now()); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("61 meses no faturamento = %v, esperado erro de validação", err)
	}
}

func TestHojeUsaOFusoDeBrasilia(t *testing.T) {
	hoje := domaindre.Hoje(time.Date(2026, time.October, 21, 2, 0, 0, 0, time.UTC))

	if !hoje.Equal(time.Date(2026, time.October, 20, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("hoje = %s, esperado 2026-10-20", hoje)
	}
}

func TestSemanaPadraoEAUltimaEncerrada(t *testing.T) {
	hoje := time.Date(2026, time.October, 21, 0, 0, 0, 0, time.UTC)

	semana, err := domaindre.NovaSemana(nil, hoje)
	if err != nil {
		t.Fatalf("NovaSemana falhou: %v", err)
	}

	if !semana.Inicio().Equal(time.Date(2026, time.October, 12, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("início = %s, esperado segunda 2026-10-12", semana.Inicio())
	}
	if !semana.Fim().Equal(time.Date(2026, time.October, 18, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("fim = %s, esperado domingo 2026-10-18", semana.Fim())
	}
}

func TestSemanaInformadaVaiParaASegundaFeira(t *testing.T) {
	hoje := time.Date(2026, time.October, 21, 0, 0, 0, 0, time.UTC)
	domingo := time.Date(2026, time.October, 25, 0, 0, 0, 0, time.UTC)
	quarta := time.Date(2026, time.October, 21, 0, 0, 0, 0, time.UTC)

	semana, err := domaindre.NovaSemana(&domingo, hoje)
	if err != nil {
		t.Fatalf("semana corrente deveria passar: %v", err)
	}
	if !semana.Inicio().Equal(time.Date(2026, time.October, 19, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("início = %s, esperado 2026-10-19", semana.Inicio())
	}
	if domaindre.SemanaDe(quarta) != semana {
		t.Errorf("SemanaDe(quarta) deveria cair na mesma semana")
	}

	proxima := time.Date(2026, time.October, 26, 0, 0, 0, 0, time.UTC)
	if _, err := domaindre.NovaSemana(&proxima, hoje); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("semana futura = %v, esperado erro de validação", err)
	}
}
