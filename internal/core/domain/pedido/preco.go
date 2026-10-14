package pedido

import (
	"math"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type Preco struct {
	centavos int64
}

func NovoPreco(centavos int64) (Preco, error) {
	if centavos < 0 {
		return Preco{}, domain.ErroValidacao("preço unitário não pode ser negativo")
	}

	return Preco{centavos: centavos}, nil
}

func PrecoDe(centavos int64) Preco {
	return Preco{centavos: centavos}
}

func (p Preco) Centavos() int64 {
	return p.centavos
}

func (p Preco) Vezes(quantidade int) (int64, error) {
	if quantidade < 0 {
		return 0, domain.ErroValidacao("quantidade não pode ser negativa")
	}
	if quantidade > 0 && p.centavos > math.MaxInt64/int64(quantidade) {
		return 0, erroValorAcimaDoLimite()
	}

	return p.centavos * int64(quantidade), nil
}

func somarCentavos(a, b int64) (int64, error) {
	if b > 0 && a > math.MaxInt64-b {
		return 0, erroValorAcimaDoLimite()
	}

	return a + b, nil
}

func erroValorAcimaDoLimite() error {
	return domain.ErroValidacao("valor do pedido acima do limite permitido")
}
