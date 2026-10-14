package fornecedor

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

const (
	tamanhoMaximoEmail    = 255
	tamanhoMaximoTelefone = 40
)

func textoObrigatorio(valor string, campo string, maximo int) (string, error) {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return "", domain.ErroValidacao(campo + " é obrigatório")
	}
	if utf8.RuneCountInString(valor) > maximo {
		return "", domain.ErroValidacao(campo + " deve ter no máximo " + strconv.Itoa(maximo) + " caracteres")
	}

	return valor, nil
}

func textoOpcional(valor *string, campo string, maximo int) (*string, error) {
	if valor == nil {
		return nil, nil
	}

	limpo := strings.TrimSpace(*valor)
	if limpo == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(limpo) > maximo {
		return nil, domain.ErroValidacao(campo + " deve ter no máximo " + strconv.Itoa(maximo) + " caracteres")
	}

	return &limpo, nil
}

func emailOpcional(valor *string, campo string) (*string, error) {
	email, err := textoOpcional(valor, campo, tamanhoMaximoEmail)
	if err != nil || email == nil {
		return email, err
	}

	usuario, dominio, ok := strings.Cut(*email, "@")
	if !ok || usuario == "" || dominio == "" || strings.ContainsAny(*email, " ,;") || strings.Contains(dominio, "@") {
		return nil, domain.ErroValidacao(campo + " inválido")
	}

	return email, nil
}
