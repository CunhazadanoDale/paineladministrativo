package estoque_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

func TestProdutoMovimentarEntradaAumentaSaldo(t *testing.T) {
	produto := novoProdutoDeTeste(t)

	movimento, err := produto.Movimentar(
		domainestoque.TipoMovimentoEntrada,
		domainestoque.QuantidadeDe(20),
		uuid.New(),
		nil,
		"nota fiscal 123",
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("movimentação falhou: %v", err)
	}
	if produto.Saldo.Quantidade() != 20 {
		t.Errorf("saldo = %d, esperado 20", produto.Saldo.Quantidade())
	}
	if movimento.SaldoApos.Quantidade() != 20 {
		t.Errorf("saldo do movimento = %d, esperado 20", movimento.SaldoApos.Quantidade())
	}
	if movimento.ProdutoID != produto.ID {
		t.Error("movimento apontando para outro produto")
	}
}

func TestProdutoMovimentarSaidaDiminuiSaldo(t *testing.T) {
	produto := novoProdutoDeTeste(t)
	usuarioID := uuid.New()

	if _, err := produto.Movimentar(
		domainestoque.TipoMovimentoEntrada,
		domainestoque.QuantidadeDe(10),
		usuarioID,
		nil,
		"",
		time.Now().UTC(),
	); err != nil {
		t.Fatalf("entrada falhou: %v", err)
	}

	if _, err := produto.Movimentar(
		domainestoque.TipoMovimentoSaida,
		domainestoque.QuantidadeDe(4),
		usuarioID,
		nil,
		"consumo na obra",
		time.Now().UTC(),
	); err != nil {
		t.Fatalf("saída falhou: %v", err)
	}
	if produto.Saldo.Quantidade() != 6 {
		t.Errorf("saldo = %d, esperado 6", produto.Saldo.Quantidade())
	}
}

func TestProdutoMovimentarBloqueiaSaldoNegativo(t *testing.T) {
	produto := novoProdutoDeTeste(t)

	if _, err := produto.Movimentar(
		domainestoque.TipoMovimentoSaida,
		domainestoque.QuantidadeDe(1),
		uuid.New(),
		nil,
		"",
		time.Now().UTC(),
	); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("saída sem saldo = %v, esperado erro de validação", err)
	}
	if !produto.Saldo.Vazio() {
		t.Errorf("saldo = %d, esperado permanecer 0", produto.Saldo.Quantidade())
	}
}

func TestProdutoMovimentarAjusteNegativoPodeZerar(t *testing.T) {
	produto := novoProdutoDeTeste(t)
	usuarioID := uuid.New()

	if _, err := produto.Movimentar(
		domainestoque.TipoMovimentoEntrada,
		domainestoque.QuantidadeDe(5),
		usuarioID,
		nil,
		"",
		time.Now().UTC(),
	); err != nil {
		t.Fatalf("entrada falhou: %v", err)
	}

	if _, err := produto.Movimentar(
		domainestoque.TipoMovimentoAjuste,
		domainestoque.QuantidadeDe(-5),
		usuarioID,
		nil,
		"contagem de inventário",
		time.Now().UTC(),
	); err != nil {
		t.Fatalf("ajuste falhou: %v", err)
	}
	if !produto.Saldo.Vazio() {
		t.Errorf("saldo = %d, esperado 0", produto.Saldo.Quantidade())
	}
}

func TestProdutoMovimentarBloqueiaAjusteNegativoAlemDoSaldo(t *testing.T) {
	produto := novoProdutoDeTeste(t)

	if _, err := produto.Movimentar(
		domainestoque.TipoMovimentoAjuste,
		domainestoque.QuantidadeDe(-1),
		uuid.New(),
		nil,
		"",
		time.Now().UTC(),
	); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("ajuste estourando saldo = %v, esperado erro de validação", err)
	}
}

func TestProdutoMovimentarRejeitaUsuarioAusente(t *testing.T) {
	produto := novoProdutoDeTeste(t)

	if _, err := produto.Movimentar(
		domainestoque.TipoMovimentoEntrada,
		domainestoque.QuantidadeDe(1),
		uuid.Nil,
		nil,
		"",
		time.Now().UTC(),
	); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("movimentação sem usuário = %v, esperado erro de validação", err)
	}
}

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

	if _, err := produto.Movimentar(
		domainestoque.TipoMovimentoEntrada,
		domainestoque.QuantidadeDe(6),
		uuid.New(),
		nil,
		"",
		time.Now().UTC(),
	); err != nil {
		t.Fatalf("entrada falhou: %v", err)
	}

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
