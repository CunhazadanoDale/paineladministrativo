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

func novaCategoriaRaiz(t *testing.T) *domainestoque.Categoria {
	t.Helper()

	slug, err := domainestoque.NovoSlug("Materiais")
	if err != nil {
		t.Fatalf("slug de teste inválido: %v", err)
	}

	categoria, err := domainestoque.NovaCategoria("Materiais", slug, nil, 0, "")
	if err != nil {
		t.Fatalf("criação da categoria falhou: %v", err)
	}

	return categoria
}

func novaSubcategoriaDeTeste(t *testing.T) *domainestoque.Categoria {
	t.Helper()

	slug, err := domainestoque.NovoSlug("Alvenaria")
	if err != nil {
		t.Fatalf("slug de teste inválido: %v", err)
	}

	subcategoria, err := domainestoque.NovaCategoria("Alvenaria", slug, novaCategoriaRaiz(t), 1, "brick")
	if err != nil {
		t.Fatalf("criação da subcategoria falhou: %v", err)
	}

	return subcategoria
}

func TestNovaCategoriaRaizNaoTemPai(t *testing.T) {
	categoria := novaCategoriaRaiz(t)

	if categoria.ID == uuid.Nil {
		t.Error("categoria criada sem id")
	}
	if categoria.CategoriaPaiID != nil {
		t.Error("categoria raiz não deveria ter pai")
	}
	if categoria.EhSubcategoria() {
		t.Error("categoria raiz não é subcategoria")
	}
	if !categoria.Ativo {
		t.Error("categoria nova deveria nascer ativa")
	}
}

func TestNovaSubcategoriaVinculaPai(t *testing.T) {
	subcategoria := novaSubcategoriaDeTeste(t)

	if subcategoria.CategoriaPaiID == nil {
		t.Fatal("subcategoria deveria ter pai")
	}
	if !subcategoria.EhSubcategoria() {
		t.Error("categoria com pai é subcategoria")
	}
}

func TestNovaCategoriaRejeitaNomeVazio(t *testing.T) {
	slug, err := domainestoque.NovoSlug("Materiais")
	if err != nil {
		t.Fatalf("slug de teste inválido: %v", err)
	}

	if _, err := domainestoque.NovaCategoria("   ", slug, nil, 0, ""); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("nome vazio = %v, esperado erro de validação", err)
	}
}

func TestNovaCategoriaRejeitaNomeAcimaDe120(t *testing.T) {
	slug, err := domainestoque.NovoSlug("Materiais")
	if err != nil {
		t.Fatalf("slug de teste inválido: %v", err)
	}

	if _, err := domainestoque.NovaCategoria(strings.Repeat("a", 121), slug, nil, 0, ""); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("nome longo = %v, esperado erro de validação", err)
	}
}

func TestNovaCategoriaRejeitaTresNiveis(t *testing.T) {
	subcategoria := novaSubcategoriaDeTeste(t)

	slug, err := domainestoque.NovoSlug("Tijolos")
	if err != nil {
		t.Fatalf("slug de teste inválido: %v", err)
	}

	if _, err := domainestoque.NovaCategoria("Tijolos", slug, subcategoria, 0, ""); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("terceiro nível = %v, esperado erro de validação", err)
	}
}

func TestNovaCategoriaRejeitaPaiInativo(t *testing.T) {
	pai := novaCategoriaRaiz(t)
	pai.Ativo = false

	slug, err := domainestoque.NovoSlug("Alvenaria")
	if err != nil {
		t.Fatalf("slug de teste inválido: %v", err)
	}

	if _, err := domainestoque.NovaCategoria("Alvenaria", slug, pai, 0, ""); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("pai inativo = %v, esperado erro de validação", err)
	}
}

func TestCategoriaAlternarAtivoBloqueiaComSubcategorias(t *testing.T) {
	categoria := novaCategoriaRaiz(t)
	agora := time.Now().UTC()

	if err := categoria.AlternarAtivo(false, true, agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("inativar com subcategorias = %v, esperado erro de conflito", err)
	}
	if err := categoria.AlternarAtivo(false, false, agora); err != nil {
		t.Errorf("inativar sem subcategorias falhou: %v", err)
	}
	if categoria.Ativo {
		t.Error("categoria deveria estar inativa")
	}
}

func TestCategoriaAlterarDadosAtualizaCampos(t *testing.T) {
	categoria := novaCategoriaRaiz(t)
	agora := time.Now().UTC()

	if err := categoria.AlterarDados("  Materiais Gerais  ", 5, "caixa", agora); err != nil {
		t.Fatalf("alteração falhou: %v", err)
	}
	if categoria.Nome != "Materiais Gerais" {
		t.Errorf("nome = %q, esperado sem espaços nas bordas", categoria.Nome)
	}
	if categoria.Ordem != 5 || categoria.Icone != "caixa" {
		t.Errorf("ordem = %d e icone = %q, esperado 5 e caixa", categoria.Ordem, categoria.Icone)
	}
	if !categoria.AtualizadoEm.Equal(agora) {
		t.Error("alteração deveria atualizar o carimbo de data")
	}
	if categoria.Slug.Valor() != "materiais" {
		t.Errorf("slug = %q, esperado manter o slug original", categoria.Slug.Valor())
	}
}

func TestCategoriaAlterarDadosValidaEntradas(t *testing.T) {
	categoria := novaCategoriaRaiz(t)
	agora := time.Now().UTC()

	if err := categoria.AlterarDados("   ", 0, "", agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("nome vazio = %v, esperado erro de validação", err)
	}
	if err := categoria.AlterarDados("Nome", -1, "", agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("ordem negativa = %v, esperado erro de validação", err)
	}
	if err := categoria.AlterarDados("Nome", 0, strings.Repeat("x", 61), agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("icone longo demais = %v, esperado erro de validação", err)
	}
}
