package cliente

import (
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

const pontosBaseMaximo = 10000

type Percentual struct {
	pontosBase int
}

func NovoPercentual(pontosBase int) (Percentual, error) {
	if pontosBase < 0 || pontosBase > pontosBaseMaximo {
		return Percentual{}, domain.ErroValidacao("percentual deve estar entre 0 e 10000 pontos-base (0% a 100%)")
	}

	return Percentual{pontosBase: pontosBase}, nil
}

func PercentualDe(pontosBase int) Percentual {
	return Percentual{pontosBase: pontosBase}
}

func (p Percentual) PontosBase() int {
	return p.pontosBase
}
