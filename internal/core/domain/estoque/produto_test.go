package estoque_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

func novoProdutoDeTeste(t *testing.T) *domainestoque.Produto {
	t.Helper()

	slug, err := domainestoque.NovoSlug("Cimento CP II 50kg")
	if err != nil {
		t.Fatalf("slug de teste inválido: %v", err)
	}

	produto, err := domainestoque.NovoProduto(domainestoque.ProdutoInput{
		Categoria:     novaSubcategoriaDeTeste(t),
		Nome:          "Cimento CP II 50kg",
		Slug:          slug,
		UnidadeMedida: domainestoque.UnidadeMedidaSc,
	})
	if err != nil {
		t.Fatalf("criação do produto falhou: %v", err)
	}

	return produto
}

func TestNovoProdutoNasceAtivoESemSaldo(t *testing.T) {
	produto := novoProdutoDeTeste(t)

	if produto.ID == uuid.Nil {
		t.Error("produto criado sem id")
	}
	if !produto.Ativo {
		t.Error("produto novo deveria nascer ativo")
	}
	if !produto.Saldo.Vazio() {
		t.Errorf("saldo inicial = %d, esperado 0", produto.Saldo.Quantidade())
	}
	if produto.CategoriaID == uuid.Nil {
		t.Error("produto criado sem categoria")
	}
}

func TestNovoProdutoExigeSubcategoria(t *testing.T) {
	slug, err := domainestoque.NovoSlug("Cimento")
	if err != nil {
		t.Fatalf("slug de teste inválido: %v", err)
	}

	if _, err := domainestoque.NovoProduto(domainestoque.ProdutoInput{
		Categoria:     novaCategoriaRaiz(t),
		Nome:          "Cimento",
		Slug:          slug,
		UnidadeMedida: domainestoque.UnidadeMedidaSc,
	}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("produto em categoria raiz = %v, esperado erro de validação", err)
	}
}

func TestNovoProdutoExigeCategoria(t *testing.T) {
	slug, err := domainestoque.NovoSlug("Cimento")
	if err != nil {
		t.Fatalf("slug de teste inválido: %v", err)
	}

	if _, err := domainestoque.NovoProduto(domainestoque.ProdutoInput{
		Nome:          "Cimento",
		Slug:          slug,
		UnidadeMedida: domainestoque.UnidadeMedidaSc,
	}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("produto sem categoria = %v, esperado erro de validação", err)
	}
}

func TestNovoProdutoRejeitaNomeVazio(t *testing.T) {
	slug, err := domainestoque.NovoSlug("Cimento")
	if err != nil {
		t.Fatalf("slug de teste inválido: %v", err)
	}

	if _, err := domainestoque.NovoProduto(domainestoque.ProdutoInput{
		Categoria:     novaSubcategoriaDeTeste(t),
		Nome:          "  ",
		Slug:          slug,
		UnidadeMedida: domainestoque.UnidadeMedidaSc,
	}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("nome vazio = %v, esperado erro de validação", err)
	}
}

func TestNovoProdutoRejeitaNomeAcimaDe200(t *testing.T) {
	slug, err := domainestoque.NovoSlug("Cimento")
	if err != nil {
		t.Fatalf("slug de teste inválido: %v", err)
	}

	if _, err := domainestoque.NovoProduto(domainestoque.ProdutoInput{
		Categoria:     novaSubcategoriaDeTeste(t),
		Nome:          strings.Repeat("a", 201),
		Slug:          slug,
		UnidadeMedida: domainestoque.UnidadeMedidaSc,
	}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("nome longo = %v, esperado erro de validação", err)
	}
}

func TestNovoProdutoRejeitaUnidadeInvalida(t *testing.T) {
	slug, err := domainestoque.NovoSlug("Cimento")
	if err != nil {
		t.Fatalf("slug de teste inválido: %v", err)
	}

	if _, err := domainestoque.NovoProduto(domainestoque.ProdutoInput{
		Categoria:     novaSubcategoriaDeTeste(t),
		Nome:          "Cimento",
		Slug:          slug,
		UnidadeMedida: domainestoque.UnidadeMedida("invalida"),
	}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("unidade inválida = %v, esperado erro de validação", err)
	}
}

func TestNovoProdutoAceitaSobConsulta(t *testing.T) {
	produto := novoProdutoDeTeste(t)

	if produto.Preco != nil {
		t.Error("produto sem preço deveria ficar sob consulta")
	}
}

func TestNovoProdutoPromocionalExigePrecoCheio(t *testing.T) {
	slug, err := domainestoque.NovoSlug("Cimento")
	if err != nil {
		t.Fatalf("slug de teste inválido: %v", err)
	}

	promocional, err := domainestoque.NovoPreco(8000)
	if err != nil {
		t.Fatalf("preço de teste inválido: %v", err)
	}

	if _, err := domainestoque.NovoProduto(domainestoque.ProdutoInput{
		Categoria:        novaSubcategoriaDeTeste(t),
		Nome:             "Cimento",
		Slug:             slug,
		UnidadeMedida:    domainestoque.UnidadeMedidaSc,
		PrecoPromocional: &promocional,
	}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("promoção sem preço cheio = %v, esperado erro de validação", err)
	}
}

func TestProdutoAlterarDadosAtualizaCamposPreservandoSlugESaldo(t *testing.T) {
	produto := novoProdutoDeTeste(t)
	produto.Saldo = domainestoque.SaldoDe(8)
	agora := time.Now().UTC()

	preco, err := domainestoque.NovoPreco(3200)
	if err != nil {
		t.Fatalf("preço de teste inválido: %v", err)
	}

	if err := produto.AlterarDados(domainestoque.ProdutoInput{
		Categoria:     novaSubcategoriaDeTeste(t),
		Nome:          "  Cimento CP II 50kg (saco)  ",
		Slug:          domainestoque.SlugDe("ignorado"),
		Descricao:     "Uso geral",
		UnidadeMedida: domainestoque.UnidadeMedidaSc,
		Preco:         &preco,
		Destaque:      true,
	}, agora); err != nil {
		t.Fatalf("alteração falhou: %v", err)
	}

	if produto.Nome != "Cimento CP II 50kg (saco)" {
		t.Errorf("nome = %q, esperado sem espaços nas bordas", produto.Nome)
	}
	if produto.Slug.Valor() != "cimento-cp-ii-50kg" {
		t.Errorf("slug = %q, esperado manter o slug original", produto.Slug.Valor())
	}
	if produto.Saldo.Quantidade() != 8 {
		t.Errorf("saldo = %d, esperado 8 (preservado)", produto.Saldo.Quantidade())
	}
	if produto.Preco == nil || produto.Preco.Centavos() != 3200 {
		t.Errorf("preço = %v, esperado 3200 centavos", produto.Preco)
	}
	if !produto.Destaque {
		t.Error("produto deveria estar em destaque")
	}
	if !produto.AtualizadoEm.Equal(agora) {
		t.Error("alteração deveria atualizar o carimbo de data")
	}
}

func TestProdutoAlterarDadosValidaEntradas(t *testing.T) {
	produto := novoProdutoDeTeste(t)
	agora := time.Now().UTC()

	promocional, err := domainestoque.NovoPreco(1500)
	if err != nil {
		t.Fatalf("preço de teste inválido: %v", err)
	}

	if err := produto.AlterarDados(domainestoque.ProdutoInput{
		Categoria:        novaSubcategoriaDeTeste(t),
		Nome:             "Cimento",
		Slug:             produto.Slug,
		UnidadeMedida:    domainestoque.UnidadeMedidaSc,
		PrecoPromocional: &promocional,
	}, agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("promoção sem preço cheio = %v, esperado erro de validação", err)
	}

	if err := produto.AlterarDados(domainestoque.ProdutoInput{
		Categoria:     novaCategoriaRaiz(t),
		Nome:          "Cimento",
		Slug:          produto.Slug,
		UnidadeMedida: domainestoque.UnidadeMedidaSc,
	}, agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("categoria raiz = %v, esperado erro de validação", err)
	}
}
