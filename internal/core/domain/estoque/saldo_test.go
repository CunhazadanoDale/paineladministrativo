package estoque_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
)

func TestNovoSaldoRejeitaNegativo(t *testing.T) {
	for _, quantidade := range []int{-1, -100} {
		if _, err := domainestoque.NovoSaldo(quantidade); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("NovoSaldo(%d) = %v, esperado erro de validação", quantidade, err)
		}
	}
}

func TestSaldoAplicarAceitaDentroDoDisponivel(t *testing.T) {
	saldo, err := domainestoque.NovoSaldo(10)
	if err != nil {
		t.Fatalf("NovoSaldo(10) falhou: %v", err)
	}

	proximo, err := saldo.Aplicar(-10)
	if err != nil {
		t.Fatalf("Aplicar(-10) falhou: %v", err)
	}
	if proximo.Quantidade() != 0 {
		t.Errorf("saldo = %d, esperado 0", proximo.Quantidade())
	}
	if !proximo.Vazio() {
		t.Error("saldo zerado deveria ser vazio")
	}
}

func TestSaldoAplicarBloqueiaNegativo(t *testing.T) {
	saldo, err := domainestoque.NovoSaldo(5)
	if err != nil {
		t.Fatalf("NovoSaldo(5) falhou: %v", err)
	}

	if _, err := saldo.Aplicar(-6); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("Aplicar(-6) = %v, esperado erro de validação", err)
	}
	if saldo.Quantidade() != 5 {
		t.Errorf("saldo alterado após tentativa inválida = %d, esperado 5", saldo.Quantidade())
	}
}

func TestEstoqueMinimoAtingido(t *testing.T) {
	minimo, err := domainestoque.NovoEstoqueMinimo(3)
	if err != nil {
		t.Fatalf("NovoEstoqueMinimo(3) falhou: %v", err)
	}

	if !minimo.Atingido(domainestoque.SaldoDe(3)) {
		t.Error("saldo igual ao mínimo deveria estar atingido")
	}
	if minimo.Atingido(domainestoque.SaldoDe(4)) {
		t.Error("saldo acima do mínimo não deveria estar atingido")
	}
}
