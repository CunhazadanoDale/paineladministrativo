package importacao

import (
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

const valorMaximoCentesimos = 1_000_000_000_000_000

type ValorMoeda struct {
	centesimos int64
}

func NovoValorMoeda(centesimos int64) (ValorMoeda, error) {
	if centesimos < 0 {
		return ValorMoeda{}, domain.ErroValidacao("valor do item na moeda não pode ser negativo")
	}
	if centesimos > valorMaximoCentesimos {
		return ValorMoeda{}, erroValorAcimaDoLimite()
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
	if outro.centesimos > 0 && v.centesimos > valorMaximoCentesimos-outro.centesimos {
		return ValorMoeda{}, erroValorAcimaDoLimite()
	}

	return ValorMoeda{centesimos: v.centesimos + outro.centesimos}, nil
}

func erroValorAcimaDoLimite() error {
	return domain.ErroValidacao("valor acima do limite permitido")
}
