package estoque

import (
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type Custo struct {
	centavos int64
}

func NovoCusto(centavos int64) (Custo, error) {
	if centavos < 0 {
		return Custo{}, domain.ErroValidacao("custo unitário não pode ser negativo")
	}

	return Custo{centavos: centavos}, nil
}

func CustoDe(centavos int64) Custo {
	return Custo{centavos: centavos}
}

func (c Custo) Centavos() int64 {
	return c.centavos
}
