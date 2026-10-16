package notafiscal

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

const (
	casasDeCentavos           = 2
	casasDeQuantidade         = 4
	casasDeValorUnitario      = 10
	digitosInteirosMaximos    = 13
	digitosInteirosDoUnitario = 11
)

func CentavosDe(texto, campo string) (int64, error) {
	return decimalEscalado(texto, campo, casasDeCentavos)
}

func CentavosOpcionaisDe(texto, campo string) (int64, error) {
	if strings.TrimSpace(texto) == "" {
		return 0, nil
	}

	return CentavosDe(texto, campo)
}

func QuantidadeDe(texto, campo string) (int64, error) {
	quantidade, err := decimalEscalado(texto, campo, casasDeQuantidade)
	if err != nil {
		return 0, err
	}
	if quantidade == 0 {
		return 0, domain.ErroValidacao(campo + " deve ser maior que zero")
	}

	return quantidade, nil
}

func ValorUnitarioDe(texto, campo string) (string, error) {
	limpo := strings.TrimSpace(texto)
	inteiro, fracao, temFracao := strings.Cut(limpo, ".")
	if !somenteDigitos(inteiro) || len(inteiro) > digitosInteirosDoUnitario ||
		(temFracao && (!somenteDigitos(fracao) || len(fracao) > casasDeValorUnitario)) {
		return "", domain.ErroValidacao(campo + " inválido")
	}

	return limpo, nil
}

func decimalEscalado(texto, campo string, casas int) (int64, error) {
	limpo := strings.TrimSpace(texto)
	inteiro, fracao, temFracao := strings.Cut(limpo, ".")

	switch {
	case !somenteDigitos(inteiro) || len(inteiro) > digitosInteirosMaximos:
		return 0, domain.ErroValidacao(campo + " inválido")
	case temFracao && !somenteDigitos(fracao):
		return 0, domain.ErroValidacao(campo + " inválido")
	case len(fracao) > casas:
		return 0, domain.ErroValidacao(campo + " com casas decimais demais")
	}

	fracao += strings.Repeat("0", casas-len(fracao))

	var valor int64
	for _, caractere := range inteiro + fracao {
		valor = valor*10 + int64(caractere-'0')
	}

	return valor, nil
}

func somenteDigitos(texto string) bool {
	if texto == "" {
		return false
	}
	for posicao := 0; posicao < len(texto); posicao++ {
		if !numerico(texto[posicao]) {
			return false
		}
	}

	return true
}
