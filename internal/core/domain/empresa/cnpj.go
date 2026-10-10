package empresa

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

const (
	tamanhoCNPJ          = 14
	tamanhoRaizCNPJ      = 12
	mensagemCNPJInvalido = "cnpj inválido"
)

type CNPJ struct {
	digitos string
}

func NovoCNPJ(valor string) (CNPJ, error) {
	digitos, ok := normalizarCNPJ(valor)
	if !ok || len(digitos) != tamanhoCNPJ || todosIguais(digitos) {
		return CNPJ{}, domain.ErroValidacao(mensagemCNPJInvalido)
	}

	for posicao, caractere := range digitos {
		if posicao < tamanhoRaizCNPJ && !alfanumerico(caractere) {
			return CNPJ{}, domain.ErroValidacao(mensagemCNPJInvalido)
		}
		if posicao >= tamanhoRaizCNPJ && !numerico(caractere) {
			return CNPJ{}, domain.ErroValidacao(mensagemCNPJInvalido)
		}
	}

	if digitosVerificadores(digitos[:tamanhoRaizCNPJ]) != digitos[tamanhoRaizCNPJ:] {
		return CNPJ{}, domain.ErroValidacao(mensagemCNPJInvalido)
	}

	return CNPJ{digitos: digitos}, nil
}

func CNPJDe(digitos string) CNPJ {
	return CNPJ{digitos: digitos}
}

func (c CNPJ) Digitos() string {
	return c.digitos
}

func (c CNPJ) Vazio() bool {
	return c.digitos == ""
}

func (c CNPJ) Formatado() string {
	if len(c.digitos) != tamanhoCNPJ {
		return c.digitos
	}

	return c.digitos[0:2] + "." + c.digitos[2:5] + "." + c.digitos[5:8] + "/" + c.digitos[8:12] + "-" + c.digitos[12:14]
}

func normalizarCNPJ(valor string) (string, bool) {
	var construtor strings.Builder

	for _, caractere := range strings.ToUpper(valor) {
		switch {
		case caractere == '.' || caractere == '/' || caractere == '-' || caractere == ' ':
			continue
		case alfanumerico(caractere):
			construtor.WriteRune(caractere)
		default:
			return "", false
		}
	}

	return construtor.String(), true
}

func digitosVerificadores(raiz string) string {
	pesos := []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}

	valores := make([]int, 0, tamanhoCNPJ)
	for _, caractere := range raiz {
		valores = append(valores, int(caractere-'0'))
	}

	primeiro := digitoVerificador(valores, pesos)
	segundo := digitoVerificador(append(valores, primeiro), append([]int{6}, pesos...))

	return string(rune('0'+primeiro)) + string(rune('0'+segundo))
}

func digitoVerificador(valores, pesos []int) int {
	soma := 0
	for indice, valor := range valores {
		soma += valor * pesos[indice]
	}

	resto := soma % 11
	if resto < 2 {
		return 0
	}

	return 11 - resto
}

func todosIguais(texto string) bool {
	return strings.Count(texto, texto[:1]) == len(texto)
}

func alfanumerico(caractere rune) bool {
	return numerico(caractere) || (caractere >= 'A' && caractere <= 'Z')
}

func numerico(caractere rune) bool {
	return caractere >= '0' && caractere <= '9'
}
