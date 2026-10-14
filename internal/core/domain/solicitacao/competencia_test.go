package solicitacao_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
)

func TestNovaCompetenciaAceitaAnoEMes(t *testing.T) {
	competencia, err := domainsolicitacao.NovaCompetencia(" 2026-10 ")
	if err != nil {
		t.Fatalf("NovaCompetencia falhou: %v", err)
	}

	esperado := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
	if !competencia.Inicio().Equal(esperado) {
		t.Errorf("início = %s, esperado %s", competencia.Inicio(), esperado)
	}
	if competencia.String() != "2026-10" {
		t.Errorf("texto = %q, esperado 2026-10", competencia.String())
	}
}

func TestNovaCompetenciaRejeitaFormatoEIntervaloInvalidos(t *testing.T) {
	for _, valor := range []string{"", "2026-13", "2026-10-01", "10/2026", "1999-12", "2100-01"} {
		if _, err := domainsolicitacao.NovaCompetencia(valor); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("NovaCompetencia(%q) = %v, esperado erro de validação", valor, err)
		}
	}
}

func TestCompetenciaDeUsaOPrimeiroDiaDoMes(t *testing.T) {
	competencia := domainsolicitacao.CompetenciaDe(time.Date(2026, time.November, 30, 23, 0, 0, 0, time.UTC))

	if competencia.String() != "2026-11" || competencia.Inicio().Day() != 1 {
		t.Errorf("competência = %s, esperado 2026-11-01", competencia.Inicio())
	}
}
