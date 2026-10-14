package financeiro_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
)

func texto(valor string) *string {
	return &valor
}

func TestNovaAgenciaSeparaNumeroEDigito(t *testing.T) {
	casos := []struct {
		valor     string
		digito    *string
		numero    string
		esperado  *string
		formatado string
	}{
		{valor: "0812", numero: "0812", formatado: "0812"},
		{valor: " 1234-5 ", numero: "1234", esperado: texto("5"), formatado: "1234-5"},
		{valor: "1.234-x", numero: "1234", esperado: texto("X"), formatado: "1234-X"},
		{valor: "1234", digito: texto(" 7 "), numero: "1234", esperado: texto("7"), formatado: "1234-7"},
		{valor: "1234", digito: texto("  "), numero: "1234", formatado: "1234"},
		{valor: "0001", numero: "0001", formatado: "0001"},
	}

	for _, caso := range casos {
		agencia, err := domainfinanceiro.NovaAgencia(caso.valor, caso.digito)
		if err != nil {
			t.Fatalf("NovaAgencia(%q) falhou: %v", caso.valor, err)
		}
		if agencia.Numero() != caso.numero || agencia.Formatado() != caso.formatado {
			t.Errorf("NovaAgencia(%q) = %q/%q", caso.valor, agencia.Numero(), agencia.Formatado())
		}
		if !mesmoTexto(agencia.Digito(), caso.esperado) {
			t.Errorf("NovaAgencia(%q) dígito = %v", caso.valor, agencia.Digito())
		}
	}
}

func TestNovaAgenciaRecusaFormatoInvalido(t *testing.T) {
	casos := []struct {
		valor  string
		digito *string
	}{
		{valor: ""},
		{valor: "   "},
		{valor: "123456"},
		{valor: "12a4"},
		{valor: "0000"},
		{valor: "1234-"},
		{valor: "1234-56"},
		{valor: "1234-Y"},
		{valor: "1234", digito: texto("12")},
		{valor: "1234-5", digito: texto("5")},
		{valor: "12/34"},
	}

	for _, caso := range casos {
		if _, err := domainfinanceiro.NovaAgencia(caso.valor, caso.digito); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("NovaAgencia(%q, %v) erro = %v", caso.valor, caso.digito, err)
		}
	}
}

func TestNovaContaAceitaDigitoDuplo(t *testing.T) {
	conta, err := domainfinanceiro.NovaConta("12.345.678-9x", nil)
	if err != nil {
		t.Fatalf("NovaConta falhou: %v", err)
	}
	if conta.Numero() != "12345678" || conta.Formatado() != "12345678-9X" {
		t.Errorf("conta = %q/%q", conta.Numero(), conta.Formatado())
	}

	longa := "12345678901234567890"
	if _, err := domainfinanceiro.NovaConta(longa, nil); err != nil {
		t.Errorf("conta com 20 dígitos falhou: %v", err)
	}
	if _, err := domainfinanceiro.NovaConta(longa+"1", nil); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("conta com 21 dígitos erro = %v", err)
	}
	if _, err := domainfinanceiro.NovaConta("1234-567", nil); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("dígito com 3 caracteres erro = %v", err)
	}
}

func TestNumeroBancarioDeReconstroi(t *testing.T) {
	numero := domainfinanceiro.NumeroBancarioDe("54321", texto("0"))
	if numero.Formatado() != "54321-0" || numero.Vazio() {
		t.Errorf("número reconstruído = %q", numero.Formatado())
	}

	semDigito := domainfinanceiro.NumeroBancarioDe("54321", nil)
	if semDigito.Digito() != nil || semDigito.Formatado() != "54321" {
		t.Errorf("número sem dígito = %q", semDigito.Formatado())
	}

	if !(domainfinanceiro.NumeroBancario{}).Vazio() {
		t.Error("valor zero deveria ser vazio")
	}
}

func mesmoTexto(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	return *a == *b
}
