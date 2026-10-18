package comissao_test

import (
	"math"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	"github.com/google/uuid"
)

func competenciaDeTeste(t *testing.T, texto string) domaincomissao.Competencia {
	t.Helper()

	competencia, err := domaincomissao.NovaCompetencia(texto)
	if err != nil {
		t.Fatalf("NovaCompetencia: %v", err)
	}

	return competencia
}

func TestNovoFechamentoSomaAjustesEVenceNoDia15DoMesSeguinte(t *testing.T) {
	saldo := domaincomissao.SaldoAberto{RepresentanteID: uuid.New(), ComissoesCentavos: 10000, AjustesCentavos: -2500}

	fechamento, err := domaincomissao.NovoFechamento(uuid.New(), competenciaDeTeste(t, "2026-12"), saldo, uuid.New(), uuid.New(), agora)
	if err != nil {
		t.Fatalf("NovoFechamento: %v", err)
	}
	if fechamento.TotalCentavos != 7500 || fechamento.ComissoesCentavos != 10000 || fechamento.AjustesCentavos != -2500 {
		t.Fatalf("valores = %+v", fechamento)
	}
	if !fechamento.Vencimento.Equal(time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)) || fechamento.ID == uuid.Nil {
		t.Fatalf("vencimento = %v", fechamento.Vencimento)
	}
}

func TestNovoFechamentoRecusaTotalSemValorAPagar(t *testing.T) {
	competencia := competenciaDeTeste(t, "2026-09")
	casos := []struct {
		nome   string
		saldo  domaincomissao.SaldoAberto
		trecho string
	}{
		{"zerado", domaincomissao.SaldoAberto{RepresentanteID: uuid.New(), ComissoesCentavos: 500, AjustesCentavos: -500}, "nada a pagar"},
		{"negativo", domaincomissao.SaldoAberto{RepresentanteID: uuid.New(), AjustesCentavos: -1}, "nada a pagar"},
		{"sem representante", domaincomissao.SaldoAberto{ComissoesCentavos: 100}, "representante do fechamento"},
		{"estouro", domaincomissao.SaldoAberto{RepresentanteID: uuid.New(), ComissoesCentavos: math.MaxInt64, AjustesCentavos: 1}, "excede o limite"},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			_, err := domaincomissao.NovoFechamento(uuid.New(), competencia, caso.saldo, uuid.New(), uuid.New(), agora)
			esperarErro(t, err, domain.ErrValidacao, caso.trecho)
		})
	}
}

func TestCompetenciaDeFechamentoPrecisaEstarEncerradaEmBrasilia(t *testing.T) {
	outubro := competenciaDeTeste(t, "2026-10")

	esperarErro(t, domaincomissao.ConferirCompetenciaDeFechamento(outubro, agora), domain.ErrValidacao, "mês já encerrado")
	esperarErro(t, domaincomissao.ConferirCompetenciaDeFechamento(outubro, time.Date(2026, 11, 1, 2, 0, 0, 0, time.UTC)),
		domain.ErrValidacao, "mês já encerrado")
	if err := domaincomissao.ConferirCompetenciaDeFechamento(outubro, time.Date(2026, 11, 1, 3, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("outubro já encerrou em Brasília: %v", err)
	}
}
