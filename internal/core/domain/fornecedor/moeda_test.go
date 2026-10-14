package fornecedor_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
)

func TestNovaMoedaAceitaListaFechadaEAssumeRealQuandoVazia(t *testing.T) {
	casos := map[string]domainfornecedor.Moeda{
		"":      domainfornecedor.MoedaReal,
		" brl ": domainfornecedor.MoedaReal,
		"usd":   domainfornecedor.MoedaDolar,
		"EUR":   domainfornecedor.MoedaEuro,
		"cny":   domainfornecedor.MoedaYuanChines,
	}

	for entrada, esperada := range casos {
		moeda, err := domainfornecedor.NovaMoeda(entrada)
		if err != nil {
			t.Fatalf("NovaMoeda(%q) falhou: %v", entrada, err)
		}
		if moeda != esperada {
			t.Errorf("NovaMoeda(%q) = %q, esperada %q", entrada, moeda, esperada)
		}
	}
}

func TestNovaMoedaRecusaForaDaLista(t *testing.T) {
	for _, entrada := range []string{"GBP", "JPY", "real", "US"} {
		if _, err := domainfornecedor.NovaMoeda(entrada); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("NovaMoeda(%q) erro = %v, esperado validação", entrada, err)
		}
	}
}

func TestMoedaDeSoReconstroi(t *testing.T) {
	if domainfornecedor.MoedaDe("XYZ").Validado() == nil {
		t.Error("Validado deveria recusar a moeda XYZ reconstruída")
	}
}
