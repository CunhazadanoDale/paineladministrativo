package solicitacao

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type Origem string

const (
	OrigemManual      Origem = "manual"
	OrigemRecorrencia Origem = "recorrencia"
)

func NovaOrigem(valor string) (Origem, error) {
	origem := Origem(strings.ToLower(strings.TrimSpace(valor)))

	switch origem {
	case OrigemManual, OrigemRecorrencia:
		return origem, nil
	default:
		return "", domain.ErroValidacao("origem deve ser manual ou recorrencia")
	}
}

func OrigemDe(valor string) Origem {
	return Origem(valor)
}

func (o Origem) String() string {
	return string(o)
}
