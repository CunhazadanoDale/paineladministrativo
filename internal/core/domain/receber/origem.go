package receber

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type Origem string

const (
	OrigemManual              Origem = "manual"
	OrigemPedido              Origem = "pedido"
	OrigemNotaFiscalDuplicata Origem = "nota_fiscal_duplicata"
)

func NovaOrigem(valor string) (Origem, error) {
	origem := Origem(strings.ToLower(strings.TrimSpace(valor)))

	switch origem {
	case OrigemManual, OrigemPedido, OrigemNotaFiscalDuplicata:
		return origem, nil
	default:
		return "", domain.ErroValidacao("origem deve ser manual, pedido ou nota_fiscal_duplicata")
	}
}

func OrigemDe(valor string) Origem {
	return Origem(valor)
}

func (o Origem) String() string {
	return string(o)
}

func (o Origem) Gerada() bool {
	return o != OrigemManual
}
