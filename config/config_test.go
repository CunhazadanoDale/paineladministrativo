package config

import (
	"strings"
	"testing"
)

func TestSepararOrigens(t *testing.T) {
	casos := []struct {
		valor    string
		esperado []string
	}{
		{"", []string{}},
		{"   ", []string{}},
		{"http://localhost:5173", []string{"http://localhost:5173"}},
		{"http://localhost:5173,http://localhost:3000", []string{"http://localhost:5173", "http://localhost:3000"}},
		{" https://painel.com.br , https://admin.painel.com.br ,, ", []string{"https://painel.com.br", "https://admin.painel.com.br"}},
	}

	for _, caso := range casos {
		origens := separarOrigens(caso.valor)

		if len(origens) != len(caso.esperado) {
			t.Errorf("%q separou em %v, esperado %v", caso.valor, origens, caso.esperado)
			continue
		}

		for i, origem := range origens {
			if origem != caso.esperado[i] {
				t.Errorf("%q separou em %v, esperado %v", caso.valor, origens, caso.esperado)
				break
			}
		}
	}
}

func TestValidarExigeSegredoComTamanhoMinimo(t *testing.T) {
	casos := []struct {
		segredo string
		valido  bool
	}{
		{"", false},
		{"curto-demais-16c", false},
		{strings.Repeat("a", 31), false},
		{strings.Repeat("a", 32), true},
	}

	for _, caso := range casos {
		err := (&Config{JWTSecret: caso.segredo}).Validar()
		if (err == nil) != caso.valido {
			t.Errorf("segredo de %d caracteres: erro %v, esperado válido=%v", len(caso.segredo), err, caso.valido)
		}
	}
}

func TestLoadConfigUsaPortaPadrao(t *testing.T) {
	t.Setenv("PORT", "")

	if porta := LoadConfig().AppPort; porta != "8080" {
		t.Errorf("porta %q, esperado %q", porta, "8080")
	}

	t.Setenv("PORT", "9090")
	if porta := LoadConfig().AppPort; porta != "9090" {
		t.Errorf("porta %q, esperado %q", porta, "9090")
	}
}
