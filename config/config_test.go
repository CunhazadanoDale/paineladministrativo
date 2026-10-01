package config

import "testing"

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
