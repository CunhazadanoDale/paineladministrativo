package estoque

import (
	"math"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type Preco struct {
	centavos int64
}

func NovoPreco(centavos int64) (Preco, error) {
	if centavos <= 0 {
		return Preco{}, domain.ErroValidacao("preço deve ser maior que zero")
	}

	return Preco{centavos: centavos}, nil
}

func PrecoDe(centavos int64) Preco {
	return Preco{centavos: centavos}
}

func (p Preco) Centavos() int64 {
	return p.centavos
}

func (p Preco) MenorQue(outro Preco) bool {
	return p.centavos < outro.centavos
}

type Peso struct {
	quilogramas float64
}

func NovoPeso(quilogramas float64) (Peso, error) {
	if math.IsNaN(quilogramas) || math.IsInf(quilogramas, 0) || quilogramas <= 0 {
		return Peso{}, domain.ErroValidacao("peso deve ser um número maior que zero")
	}

	return Peso{quilogramas: quilogramas}, nil
}

func PesoDe(quilogramas float64) Peso {
	return Peso{quilogramas: quilogramas}
}

func (p Peso) Quilos() float64 {
	return p.quilogramas
}
