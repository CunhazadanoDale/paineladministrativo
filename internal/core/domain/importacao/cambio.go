package importacao

import (
	"math/big"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

const (
	cambioMicroDoReal = 1_000_000
	cambioMicroMaximo = 1_000_000_000
)

type Cambio struct {
	micro int64
}

func NovoCambio(micro int64) (Cambio, error) {
	if micro <= 0 {
		return Cambio{}, domain.ErroValidacao("câmbio deve ser maior que zero")
	}
	if micro > cambioMicroMaximo {
		return Cambio{}, domain.ErroValidacao("câmbio acima do limite permitido")
	}

	return Cambio{micro: micro}, nil
}

func CambioDe(micro int64) Cambio {
	return Cambio{micro: micro}
}

func CambioDoReal() Cambio {
	return Cambio{micro: cambioMicroDoReal}
}

func (c Cambio) Micro() int64 {
	return c.micro
}

func (c Cambio) EmReais(valor ValorMoeda) (int64, error) {
	return dividirArredondando(
		new(big.Int).Mul(big.NewInt(valor.Centesimos()), big.NewInt(c.micro)),
		big.NewInt(cambioMicroDoReal),
	)
}

func cambioMedio(centavos, centesimos int64) Cambio {
	if centesimos == 0 {
		return Cambio{}
	}

	micro, err := dividirArredondando(
		new(big.Int).Mul(big.NewInt(centavos), big.NewInt(cambioMicroDoReal)),
		big.NewInt(centesimos),
	)
	if err != nil {
		return Cambio{}
	}

	return Cambio{micro: micro}
}

func dividirArredondando(numerador, denominador *big.Int) (int64, error) {
	quociente, resto := new(big.Int).QuoRem(numerador, denominador, new(big.Int))
	if new(big.Int).Mul(resto, big.NewInt(2)).Cmp(denominador) >= 0 {
		quociente.Add(quociente, big.NewInt(1))
	}
	if !quociente.IsInt64() {
		return 0, domain.ErroValidacao("valor em reais acima do limite permitido")
	}

	return quociente.Int64(), nil
}
