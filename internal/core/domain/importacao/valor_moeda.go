package importacao

import (
	"math"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type ValorMoeda struct {
	centesimos int64
}

func NovoValorMoeda(centesimos int64) (ValorMoeda, error) {
	if centesimos < 0 {
		return ValorMoeda{}, domain.ErroValidacao("valor do item na moeda não pode ser negativo")
	}

	return ValorMoeda{centesimos: centesimos}, nil
}

func ValorMoedaDe(centesimos int64) ValorMoeda {
	return ValorMoeda{centesimos: centesimos}
}

func (v ValorMoeda) Centesimos() int64 {
	return v.centesimos
}

func (v ValorMoeda) Somar(outro ValorMoeda) (ValorMoeda, error) {
	if outro.centesimos > 0 && v.centesimos > math.MaxInt64-outro.centesimos {
		return ValorMoeda{}, domain.ErroValidacao("valor da importação acima do limite permitido")
	}

	return ValorMoeda{centesimos: v.centesimos + outro.centesimos}, nil
}
