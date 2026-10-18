package solicitacao

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type Origem string

const (
	OrigemManual         Origem = "manual"
	OrigemRecorrencia    Origem = "recorrencia"
	OrigemRecompra       Origem = "recompra"
	OrigemRepasseFomento Origem = "repasse_fomento"
)

func NovaOrigem(valor string) (Origem, error) {
	origem := Origem(strings.ToLower(strings.TrimSpace(valor)))

	switch origem {
	case OrigemManual, OrigemRecorrencia, OrigemRecompra, OrigemRepasseFomento:
		return origem, nil
	default:
		return "", domain.ErroValidacao("origem deve ser manual, recorrencia, recompra ou repasse_fomento")
	}
}

func OrigemDe(valor string) Origem {
	return Origem(valor)
}

func (o Origem) String() string {
	return string(o)
}

func (o Origem) Sistemica() bool {
	return o == OrigemRecompra || o == OrigemRepasseFomento
}
