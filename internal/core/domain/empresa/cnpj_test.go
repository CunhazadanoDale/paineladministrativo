package empresa_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
)

func TestNovoCNPJAceitaNumericoEAlfanumerico(t *testing.T) {
	casos := []struct {
		entrada   string
		digitos   string
		formatado string
	}{
		{"11.222.333/0001-81", "11222333000181", "11.222.333/0001-81"},
		{"11222333000181", "11222333000181", "11.222.333/0001-81"},
		{" 11 222 333 0001 81 ", "11222333000181", "11.222.333/0001-81"},
		{"12.ABC.345/01DE-35", "12ABC34501DE35", "12.ABC.345/01DE-35"},
		{"12abc34501de35", "12ABC34501DE35", "12.ABC.345/01DE-35"},
		{"12abi34501de42", "12ABI34501DE42", "12.ABI.345/01DE-42"},
	}

	for _, caso := range casos {
		cnpj, err := domainempresa.NovoCNPJ(caso.entrada)
		if err != nil {
			t.Errorf("NovoCNPJ(%q) falhou: %v", caso.entrada, err)
			continue
		}
		if cnpj.Digitos() != caso.digitos {
			t.Errorf("NovoCNPJ(%q).Digitos() = %q, esperado %q", caso.entrada, cnpj.Digitos(), caso.digitos)
		}
		if cnpj.Formatado() != caso.formatado {
			t.Errorf("NovoCNPJ(%q).Formatado() = %q, esperado %q", caso.entrada, cnpj.Formatado(), caso.formatado)
		}
	}
}

func TestNovoCNPJRejeitaInvalidos(t *testing.T) {
	casos := []string{
		"",
		"11.222.333/0001-82",
		"11.222.333/0001-8",
		"11.222.333/0001-811",
		"00.000.000/0000-00",
		"11.111.111/1111-11",
		"12.ABC.345/01DE-36",
		"12.ABC.345/01DE-3A",
		"11#222333000181",
		"1122233300018１",
		"12ABı34501DE42",
		"12ABſ34501DE28",
	}

	for _, entrada := range casos {
		if _, err := domainempresa.NovoCNPJ(entrada); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("NovoCNPJ(%q) = %v, esperado erro de validação", entrada, err)
		}
	}
}

func TestCNPJDeReconstroiSemValidar(t *testing.T) {
	cnpj := domainempresa.CNPJDe("11222333000181")

	if cnpj.Digitos() != "11222333000181" || cnpj.Formatado() != "11.222.333/0001-81" {
		t.Errorf("CNPJDe reconstruiu %q / %q", cnpj.Digitos(), cnpj.Formatado())
	}
}
