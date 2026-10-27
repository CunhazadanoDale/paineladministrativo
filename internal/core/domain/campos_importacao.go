package domain

import (
	"strconv"
	"strings"
	"time"
)

const digitosMaximosDeReais = 13

func FlagDeImportacao(valor, campo string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(valor)) {
	case "", "nao", "não", "n":
		return false, nil
	case "sim", "s":
		return true, nil
	default:
		return false, ErroValidacao(campo + " deve ser sim ou nao")
	}
}

func CentavosDeImportacao(valor, campo string) (*int64, error) {
	numero := numeroDaPlanilha(strings.TrimPrefix(strings.TrimSpace(valor), "R$"))
	if numero == "" {
		return nil, nil
	}

	inteiro, decimal, _ := strings.Cut(numero, ".")
	if inteiro == "" || len(inteiro) > digitosMaximosDeReais || len(decimal) > 2 || !soDigitos(inteiro) || !soDigitos(decimal) {
		return nil, ErroValidacao(campo + " deve ser um valor em reais, como 1234,56")
	}

	centavos, err := strconv.ParseInt(inteiro+(decimal + "00")[:2], 10, 64)
	if err != nil {
		return nil, ErroValidacao(campo + " deve ser um valor em reais, como 1234,56")
	}

	return &centavos, nil
}

func InteiroDeImportacao(valor, campo string) (*int, error) {
	limpo := strings.TrimSpace(valor)
	if limpo == "" {
		return nil, nil
	}

	numero, err := strconv.ParseInt(limpo, 10, 32)
	if err != nil {
		return nil, ErroValidacao(campo + " deve ser um número inteiro")
	}
	inteiro := int(numero)

	return &inteiro, nil
}

func DecimalDeImportacao(valor, campo string) (*float64, error) {
	numero := numeroDaPlanilha(valor)
	if numero == "" {
		return nil, nil
	}

	decimal, err := strconv.ParseFloat(numero, 64)
	if err != nil {
		return nil, ErroValidacao(campo + " deve ser um número, como 12,5")
	}

	return &decimal, nil
}

func DataDeImportacao(valor, campo string) (*time.Time, error) {
	limpo := strings.TrimSpace(valor)
	if limpo == "" {
		return nil, nil
	}

	for _, formato := range []string{"02/01/2006", "2/1/2006", "2006-01-02"} {
		if data, err := time.Parse(formato, limpo); err == nil {
			return &data, nil
		}
	}

	return nil, ErroValidacao(campo + " deve ser uma data como 31/12/2026")
}

func numeroDaPlanilha(valor string) string {
	limpo := strings.ReplaceAll(strings.TrimSpace(valor), " ", "")
	if strings.Contains(limpo, ",") {
		limpo = strings.ReplaceAll(strings.ReplaceAll(limpo, ".", ""), ",", ".")
	}

	return limpo
}

func soDigitos(valor string) bool {
	for _, r := range valor {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}
