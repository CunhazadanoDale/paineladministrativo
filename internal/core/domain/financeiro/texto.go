package financeiro

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

func textoObrigatorio(valor, campo string, maximo int) (string, error) {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return "", domain.ErroValidacao(campo + " é obrigatório")
	}
	if utf8.RuneCountInString(valor) > maximo {
		return "", domain.ErroValidacao(campo + " deve ter no máximo " + strconv.Itoa(maximo) + " caracteres")
	}

	return valor, nil
}
