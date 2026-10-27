package domain

import (
	"errors"
	"testing"
)

func TestCentavosDeImportacaoAceitaFormatosDoExcel(t *testing.T) {
	casos := map[string]int64{
		"1234,56":        123456,
		"1.234,56":       123456,
		"R$ 1.234,5":     123450,
		"1234.56":        123456,
		"89":             8900,
		" 0,9 ":          90,
		"R$1.000.000,00": 100000000,
	}

	for entrada, esperado := range casos {
		centavos, err := CentavosDeImportacao(entrada, "preco")
		if err != nil || centavos == nil || *centavos != esperado {
			t.Errorf("%q = %v, %v; esperado %d", entrada, centavos, err, esperado)
		}
	}
}

func TestCentavosDeImportacaoVazioEInvalido(t *testing.T) {
	if centavos, err := CentavosDeImportacao("  ", "preco"); centavos != nil || err != nil {
		t.Errorf("vazio = %v, %v", centavos, err)
	}

	for _, entrada := range []string{"abc", "-10,00", "1,234", "12,3,4", ",50", "99999999999999,00"} {
		if _, err := CentavosDeImportacao(entrada, "preco"); !errors.Is(err, ErrValidacao) {
			t.Errorf("%q deveria ser recusado, erro = %v", entrada, err)
		}
	}
}

func TestDecimalEInteiroDeImportacao(t *testing.T) {
	if peso, err := DecimalDeImportacao("1.250,5", "peso_kg"); err != nil || *peso != 1250.5 {
		t.Errorf("peso = %v, %v", peso, err)
	}
	if peso, err := DecimalDeImportacao("", "peso_kg"); peso != nil || err != nil {
		t.Errorf("peso vazio = %v, %v", peso, err)
	}
	if _, err := DecimalDeImportacao("doze", "peso_kg"); !errors.Is(err, ErrValidacao) {
		t.Errorf("peso inválido, erro = %v", err)
	}
	if minimo, err := InteiroDeImportacao(" 15 ", "estoque_minimo"); err != nil || *minimo != 15 {
		t.Errorf("mínimo = %v, %v", minimo, err)
	}
	if _, err := InteiroDeImportacao("1,5", "estoque_minimo"); !errors.Is(err, ErrValidacao) {
		t.Errorf("mínimo inválido, erro = %v", err)
	}
}

func TestFlagDeImportacao(t *testing.T) {
	for entrada, esperado := range map[string]bool{"": false, "Não": false, "n": false, "SIM": true, "s": true} {
		if valor, err := FlagDeImportacao(entrada, "destaque"); err != nil || valor != esperado {
			t.Errorf("%q = %v, %v", entrada, valor, err)
		}
	}
	if _, err := FlagDeImportacao("talvez", "destaque"); err == nil || err.Error() != "erro de validação: destaque deve ser sim ou nao" {
		t.Errorf("erro = %v", err)
	}
}
