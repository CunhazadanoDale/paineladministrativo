package dre_test

import (
	"testing"
	"time"

	domaindre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/dre"
)

func TestIntervaloDoPeriodoTerminaNoPrimeiroDiaDoMesSeguinte(t *testing.T) {
	intervalo := domaindre.PeriodoMensalDe(mes(t, "2026-11"), mes(t, "2026-12")).Intervalo()

	if !intervalo.Inicio.Equal(time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("início = %s", intervalo.Inicio)
	}
	if !intervalo.Fim.Equal(time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("fim = %s, esperado 2027-01-01 exclusivo", intervalo.Fim)
	}
	if !intervalo.InicioInstante().Equal(time.Date(2026, time.November, 1, 3, 0, 0, 0, time.UTC)) {
		t.Errorf("instante inicial = %s, esperado meia-noite de Brasília", intervalo.InicioInstante())
	}
	if !intervalo.FimInstante().Equal(time.Date(2027, time.January, 1, 3, 0, 0, 0, time.UTC)) {
		t.Errorf("instante final = %s", intervalo.FimInstante())
	}
}

func TestIntervaloDaSemanaTemSeteDias(t *testing.T) {
	intervalo := domaindre.SemanaDe(time.Date(2026, time.October, 14, 0, 0, 0, 0, time.UTC)).Intervalo()

	if !intervalo.Inicio.Equal(time.Date(2026, time.October, 12, 0, 0, 0, 0, time.UTC)) ||
		!intervalo.Fim.Equal(time.Date(2026, time.October, 19, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("intervalo = %s a %s", intervalo.Inicio, intervalo.Fim)
	}
}
