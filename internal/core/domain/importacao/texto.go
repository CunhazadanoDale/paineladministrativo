package importacao

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type rotulo struct {
	nome     string
	feminino bool
}

func masculino(nome string) rotulo {
	return rotulo{nome: nome}
}

func feminino(nome string) rotulo {
	return rotulo{nome: nome, feminino: true}
}

func (r rotulo) obrigatorio() string {
	if r.feminino {
		return r.nome + " é obrigatória"
	}

	return r.nome + " é obrigatório"
}

func textoObrigatorio(valor string, campo rotulo, maximo int) (string, error) {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return "", domain.ErroValidacao(campo.obrigatorio())
	}

	return textoLimitado(valor, campo.nome, maximo)
}

func textoOpcional(valor *string, campo string, maximo int) (*string, error) {
	if valor == nil {
		return nil, nil
	}

	limpo := strings.TrimSpace(*valor)
	if limpo == "" {
		return nil, nil
	}
	if _, err := textoLimitado(limpo, campo, maximo); err != nil {
		return nil, err
	}

	return &limpo, nil
}

func textoLimitado(valor, campo string, maximo int) (string, error) {
	valor = strings.TrimSpace(valor)
	if utf8.RuneCountInString(valor) > maximo {
		return "", domain.ErroValidacao(campo + " deve ter no máximo " + strconv.Itoa(maximo) + " caracteres")
	}

	return valor, nil
}
