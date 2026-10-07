package solicitacao

import (
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type Valor struct {
	centavos int64
}

func NovoValor(centavos int64) (Valor, error) {
	if centavos <= 0 {
		return Valor{}, domain.ErroValidacao("valor deve ser maior que zero")
	}

	return Valor{centavos: centavos}, nil
}

func ValorDe(centavos int64) Valor {
	return Valor{centavos: centavos}
}

func (v Valor) Centavos() int64 {
	return v.centavos
}
