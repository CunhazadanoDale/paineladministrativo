package estoque_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
)

func TestProdutoEstoqueBaixo(t *testing.T) {
	produto := novoProdutoDeTeste(t)
	minimo, err := domainestoque.NovoEstoqueMinimo(5)
	if err != nil {
		t.Fatalf("NovoEstoqueMinimo(5) falhou: %v", err)
	}
	produto.EstoqueMinimo = &minimo

	if !produto.EstoqueBaixo() {
		t.Error("saldo zerado com mínimo 5 deveria estar baixo")
	}

	produto.Saldo = domainestoque.SaldoDe(6)

	if produto.EstoqueBaixo() {
		t.Error("saldo acima do mínimo não deveria estar baixo")
	}
}

func TestProdutoAlterarPrecoValidaPromocional(t *testing.T) {
	produto := novoProdutoDeTeste(t)
	agora := time.Now().UTC()

	cheio, err := domainestoque.NovoPreco(10000)
	if err != nil {
		t.Fatalf("NovoPreco falhou: %v", err)
	}
	promocional, err := domainestoque.NovoPreco(8000)
	if err != nil {
		t.Fatalf("NovoPreco promocional falhou: %v", err)
	}

	if err := produto.AlterarPreco(&cheio, &promocional, agora); err != nil {
		t.Fatalf("alteração de preço válida falhou: %v", err)
	}
	if produto.Preco == nil || produto.Preco.Centavos() != 10000 {
		t.Error("preço cheio não registrado")
	}

	if err := produto.AlterarPreco(&cheio, &cheio, agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("promoção igual ao cheio = %v, esperado erro de validação", err)
	}
}
