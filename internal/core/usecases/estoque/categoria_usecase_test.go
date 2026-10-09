package estoque_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	"github.com/google/uuid"
)

func TestCriarCategoriaExigeAdministrador(t *testing.T) {
	c := novoCenario(t)

	if _, err := c.categoria.Criar(c.ctx, portsin.CriarCategoriaInput{
		UsuarioID: c.visitanteID,
		Nome:      "Materiais",
	}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("criação por visitante = %v, esperado erro de permissão", err)
	}
}

func TestCriarCategoriaRaizGeraSlugAutomaticamente(t *testing.T) {
	c := novoCenario(t)

	id, err := c.categoria.Criar(c.ctx, portsin.CriarCategoriaInput{
		UsuarioID: c.usuarioID,
		Nome:      "  Materiais Gerais  ",
	})
	if err != nil {
		t.Fatalf("criação falhou: %v", err)
	}

	categoria, err := c.categoria.Obter(c.ctx, id)
	if err != nil {
		t.Fatalf("busca falhou: %v", err)
	}
	if categoria.Slug.Valor() != "materiais-gerais" {
		t.Errorf("slug = %q, esperado materiais-gerais", categoria.Slug.Valor())
	}
	if categoria.CategoriaPaiID != nil {
		t.Error("categoria raiz não deveria ter pai")
	}
	if !categoria.Ativo {
		t.Error("categoria nova deveria nascer ativa")
	}
}

func TestCriarCategoriaComNomeRepetidoFalha(t *testing.T) {
	c := novoCenario(t)

	c.novaCategoriaRaiz(t, "Materiais")

	if _, err := c.categoria.Criar(c.ctx, portsin.CriarCategoriaInput{
		UsuarioID: c.usuarioID,
		Nome:      "materiais",
	}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("nome repetido = %v, esperado erro de conflito", err)
	}
}

func TestCriarSubcategoriaValidaPaiInexistente(t *testing.T) {
	c := novoCenario(t)

	fantasma := uuid.New()
	if _, err := c.categoria.Criar(c.ctx, portsin.CriarCategoriaInput{
		UsuarioID:      c.usuarioID,
		Nome:           "Alvenaria",
		CategoriaPaiID: &fantasma,
	}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("pai inexistente = %v, esperado erro de não encontrado", err)
	}
}

func TestCriarSubcategoriaValidaPaiInativo(t *testing.T) {
	c := novoCenario(t)

	raiz := c.novaCategoriaRaiz(t, "Materiais")
	raiz.Ativo = false
	if _, err := c.categoriasRepo.Atualizar(c.ctx, raiz); err != nil {
		t.Fatalf("inativação do pai falhou: %v", err)
	}

	paiID := raiz.ID
	if _, err := c.categoria.Criar(c.ctx, portsin.CriarCategoriaInput{
		UsuarioID:      c.usuarioID,
		Nome:           "Alvenaria",
		CategoriaPaiID: &paiID,
	}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("pai inativo = %v, esperado erro de validação", err)
	}
}

func TestCriarSubcategoriaVinculaPai(t *testing.T) {
	c := novoCenario(t)

	raiz := c.novaCategoriaRaiz(t, "Materiais")
	sub := c.novaSubcategoria(t, "Alvenaria", raiz)

	if sub.CategoriaPaiID == nil || *sub.CategoriaPaiID != raiz.ID {
		t.Errorf("pai = %v, esperado %s", sub.CategoriaPaiID, raiz.ID)
	}
}

func TestAlterarCategoriaPreservaSlugEVinculos(t *testing.T) {
	c := novoCenario(t)

	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))

	err := c.categoria.Alterar(c.ctx, portsin.AlterarCategoriaInput{
		UsuarioID:   c.usuarioID,
		CategoriaID: sub.ID,
		Nome:        "Alvenaria Leve",
		Ordem:       3,
		Icone:       "brick",
	})
	if err != nil {
		t.Fatalf("alteração falhou: %v", err)
	}

	salva, err := c.categoria.Obter(c.ctx, sub.ID)
	if err != nil {
		t.Fatalf("busca falhou: %v", err)
	}
	if salva.Nome != "Alvenaria Leve" {
		t.Errorf("nome = %q", salva.Nome)
	}
	if salva.Slug.Valor() != "alvenaria" {
		t.Errorf("slug = %q, esperado manter o slug original", salva.Slug.Valor())
	}
	if salva.Ordem != 3 || salva.Icone != "brick" {
		t.Errorf("ordem = %d e icone = %q, esperado 3 e brick", salva.Ordem, salva.Icone)
	}
	if salva.CategoriaPaiID == nil {
		t.Error("alteração não deveria desvincular o pai")
	}
}

func TestAlterarCategoriaExigeAdministrador(t *testing.T) {
	c := novoCenario(t)

	raiz := c.novaCategoriaRaiz(t, "Materiais")

	err := c.categoria.Alterar(c.ctx, portsin.AlterarCategoriaInput{
		UsuarioID:   c.visitanteID,
		CategoriaID: raiz.ID,
		Nome:        "Outro Nome",
	})
	if !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("alteração por visitante = %v, esperado erro de permissão", err)
	}
}

func TestAlternarAtivoDeCategoriaComSubcategoriasFalha(t *testing.T) {
	c := novoCenario(t)

	raiz := c.novaCategoriaRaiz(t, "Materiais")
	c.novaSubcategoria(t, "Alvenaria", raiz)

	err := c.categoria.AlternarAtivo(c.ctx, portsin.AlternarAtivoCategoriaInput{
		UsuarioID:   c.usuarioID,
		CategoriaID: raiz.ID,
		Ativo:       false,
	})
	if !errors.Is(err, domain.ErrConflito) {
		t.Errorf("inativação com subcategorias = %v, esperado erro de conflito", err)
	}
}

func TestAlternarAtivoDeCategoriaSemFilhosFunciona(t *testing.T) {
	c := novoCenario(t)

	raiz := c.novaCategoriaRaiz(t, "Materiais")

	err := c.categoria.AlternarAtivo(c.ctx, portsin.AlternarAtivoCategoriaInput{
		UsuarioID:   c.usuarioID,
		CategoriaID: raiz.ID,
		Ativo:       false,
	})
	if err != nil {
		t.Fatalf("inativação falhou: %v", err)
	}

	salva, err := c.categoria.Obter(c.ctx, raiz.ID)
	if err != nil {
		t.Fatalf("busca falhou: %v", err)
	}
	if salva.Ativo {
		t.Error("categoria deveria estar inativa")
	}
}

func TestObterCategoriaInexistenteFalha(t *testing.T) {
	c := novoCenario(t)

	if _, err := c.categoria.Obter(c.ctx, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("busca de categoria inexistente = %v, esperado erro de não encontrado", err)
	}
}

func TestListarCategoriasFiltraPorAtivo(t *testing.T) {
	c := novoCenario(t)

	c.novaCategoriaRaiz(t, "Materiais")
	inativa := c.novaCategoriaRaiz(t, "Ferramentas")
	if err := c.categoria.AlternarAtivo(c.ctx, portsin.AlternarAtivoCategoriaInput{
		UsuarioID:   c.usuarioID,
		CategoriaID: inativa.ID,
		Ativo:       false,
	}); err != nil {
		t.Fatalf("inativação falhou: %v", err)
	}

	ativas := true
	itens, err := c.categoria.Listar(c.ctx, portsin.ListarCategoriasInput{Ativo: &ativas})
	if err != nil {
		t.Fatalf("listagem falhou: %v", err)
	}
	if len(itens) != 1 || itens[0].Nome != "Materiais" {
		t.Errorf("itens = %+v, esperado apenas Materiais", itens)
	}
}
