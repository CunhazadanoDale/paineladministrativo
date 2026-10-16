package receber

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

type rotulo struct {
	nome     string
	feminino bool
}

func masculino(nome string) rotulo {
	return rotulo{nome: nome}
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

func textoLimitado(valor, campo string, maximo int) (string, error) {
	valor = strings.TrimSpace(valor)
	if utf8.RuneCountInString(valor) > maximo {
		return "", domain.ErroValidacao(campo + " deve ter no máximo " + strconv.Itoa(maximo) + " caracteres")
	}

	return valor, nil
}

func limitarTexto(valor string, maximo int) string {
	if utf8.RuneCountInString(valor) <= maximo {
		return valor
	}

	return string([]rune(valor)[:maximo])
}

func idOpcional(valor *uuid.UUID) *uuid.UUID {
	if valor == nil || *valor == uuid.Nil {
		return nil
	}

	copia := *valor

	return &copia
}

func mesmoID(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	return *a == *b
}
