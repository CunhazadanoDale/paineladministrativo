package estoque

import (
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type Quantidade struct {
	valor int
}

func NovaQuantidade(valor int) (Quantidade, error) {
	if valor == 0 {
		return Quantidade{}, domain.ErroValidacao("quantidade deve ser diferente de zero")
	}

	return Quantidade{valor: valor}, nil
}

func QuantidadeDe(valor int) Quantidade {
	return Quantidade{valor: valor}
}

func (q Quantidade) Valor() int {
	return q.valor
}

func (q Quantidade) Absoluta() int {
	if q.valor < 0 {
		return -q.valor
	}

	return q.valor
}
