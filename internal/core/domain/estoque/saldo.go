package estoque

import (
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type Saldo struct {
	quantidade int
}

func NovoSaldo(quantidade int) (Saldo, error) {
	if quantidade < 0 {
		return Saldo{}, domain.ErroValidacao("saldo não pode ser negativo")
	}

	return Saldo{quantidade: quantidade}, nil
}

func SaldoDe(quantidade int) Saldo {
	return Saldo{quantidade: quantidade}
}

func (s Saldo) Quantidade() int {
	return s.quantidade
}

func (s Saldo) Vazio() bool {
	return s.quantidade == 0
}

func (s Saldo) Aplicar(delta int) (Saldo, error) {
	proximo := s.quantidade + delta
	if proximo < 0 {
		return Saldo{}, domain.ErroValidacao("saldo insuficiente para a movimentação")
	}

	return Saldo{quantidade: proximo}, nil
}

type EstoqueMinimo struct {
	quantidade int
}

func NovoEstoqueMinimo(quantidade int) (EstoqueMinimo, error) {
	if quantidade < 0 {
		return EstoqueMinimo{}, domain.ErroValidacao("estoque mínimo não pode ser negativo")
	}

	return EstoqueMinimo{quantidade: quantidade}, nil
}

func EstoqueMinimoDe(quantidade int) EstoqueMinimo {
	return EstoqueMinimo{quantidade: quantidade}
}

func (e EstoqueMinimo) Quantidade() int {
	return e.quantidade
}

func (e EstoqueMinimo) Atingido(saldo Saldo) bool {
	return e.quantidade > 0 && saldo.quantidade <= e.quantidade
}
