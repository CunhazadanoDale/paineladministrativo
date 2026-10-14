package fornecedor_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
)

func TestNovoPaisNormalizaEAssumeBrasilQuandoVazio(t *testing.T) {
	casos := map[string]domainfornecedor.Pais{
		"":     domainfornecedor.PaisBrasil,
		"   ":  domainfornecedor.PaisBrasil,
		" cn ": "CN",
		"US":   "US",
		"br":   domainfornecedor.PaisBrasil,
	}

	for entrada, esperado := range casos {
		pais, err := domainfornecedor.NovoPais(entrada)
		if err != nil {
			t.Fatalf("NovoPais(%q) falhou: %v", entrada, err)
		}
		if pais != esperado {
			t.Errorf("NovoPais(%q) = %q, esperado %q", entrada, pais, esperado)
		}
	}
}

func TestNovoPaisRecusaFormatoInvalido(t *testing.T) {
	for _, entrada := range []string{"B", "BRA", "1A", "Ç1", "B R"} {
		if _, err := domainfornecedor.NovoPais(entrada); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("NovoPais(%q) erro = %v, esperado validação", entrada, err)
		}
	}
}

func TestPaisBrasil(t *testing.T) {
	if !domainfornecedor.PaisDe("BR").Brasil() || domainfornecedor.PaisDe("CN").Brasil() {
		t.Error("Brasil() deveria ser verdadeiro só para BR")
	}
}
