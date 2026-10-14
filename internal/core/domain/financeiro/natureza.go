package financeiro

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type Natureza string

const (
	NaturezaReceita       Natureza = "receita"
	NaturezaDespesa       Natureza = "despesa"
	NaturezaTransferencia Natureza = "transferencia"
)

func NovaNatureza(valor string) (Natureza, error) {
	natureza := Natureza(strings.ToLower(strings.TrimSpace(valor)))
	if err := natureza.Validado(); err != nil {
		return "", err
	}

	return natureza, nil
}

func NaturezaDe(valor string) Natureza {
	return Natureza(valor)
}

func (n Natureza) String() string {
	return string(n)
}

func (n Natureza) Validado() error {
	switch n {
	case NaturezaReceita, NaturezaDespesa, NaturezaTransferencia:
		return nil
	default:
		return domain.ErroValidacao("natureza deve ser receita, despesa ou transferencia")
	}
}
