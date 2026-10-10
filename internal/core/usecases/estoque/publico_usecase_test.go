package estoque_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	"github.com/google/uuid"
)

func TestListarCategoriasPublicasIncluiSomenteAtivas(t *testing.T) {
	c := novoCenario(t)
	raiz := c.novaCategoriaRaiz(t, "Materiais")
	c.novaSubcategoria(t, "Alvenaria", raiz)
	desativada := c.novaSubcategoria(t, "Demolição", raiz)

	if err := c.categoria.AlternarAtivo(c.ctx, portsin.AlternarAtivoCategoriaInput{
		UsuarioID:   c.usuarioID,
		CategoriaID: desativada.ID,
		Ativo:       false,
	}); err != nil {
		t.Fatalf("desativação falhou: %v", err)
	}

	categorias, err := c.publico.ListarCategorias(c.ctx)
	if err != nil {
		t.Fatalf("listagem falhou: %v", err)
	}
	if len(categorias) != 2 {
		t.Fatalf("categorias = %d, esperado 2 ativas (raiz e Alvenaria)", len(categorias))
	}

	ids := map[uuid.UUID]bool{}
	for _, categoria := range categorias {
		ids[categoria.ID] = true
	}

	if !ids[raiz.ID] {
		t.Error("categoria raiz deveria aparecer na listagem pública")
	}
	if ids[desativada.ID] {
		t.Error("categoria desativada não deveria aparecer na listagem pública")
	}
}

func TestListarProdutosPublicosOcultaInativosESemSaldo(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))

	comEstoque := c.novoProduto(t, sub, "Cimento CP II 50kg")
	c.movimentar(t, comEstoque, "entrada", 10)

	c.novoProduto(t, sub, "Areia Fina")

	inativo := c.novoProduto(t, sub, "Tijolo 8 furos")
	c.movimentar(t, inativo, "entrada", 5)
	if err := c.produto.AlternarAtivo(c.ctx, portsin.AlternarAtivoProdutoInput{
		UsuarioID: c.usuarioID,
		ProdutoID: inativo.ID,
		Ativo:     false,
	}); err != nil {
		t.Fatalf("desativação falhou: %v", err)
	}

	produtos, err := c.publico.ListarProdutos(c.ctx, portsin.ListarProdutosPublicosInput{})
	if err != nil {
		t.Fatalf("listagem falhou: %v", err)
	}
	if len(produtos) != 1 {
		t.Fatalf("produtos públicos = %d, esperado 1", len(produtos))
	}
	if produtos[0].ID != comEstoque.ID {
		t.Errorf("produto = %s, esperado o que tem estoque", produtos[0].ID)
	}
	if len(produtos[0].Imagens) != 0 {
		t.Errorf("imagens = %d, esperado lista vazia sem imagens", len(produtos[0].Imagens))
	}
}

func TestListarProdutosPublicosFiltraPorCategoriaRaiz(t *testing.T) {
	c := novoCenario(t)
	raiz := c.novaCategoriaRaiz(t, "Materiais")
	sub := c.novaSubcategoria(t, "Alvenaria", raiz)
	outraRaiz := c.novaCategoriaRaiz(t, "Hidráulica")
	subOutra := c.novaSubcategoria(t, "Tubos", outraRaiz)

	naRaiz := c.novoProduto(t, sub, "Cimento CP II 50kg")
	c.movimentar(t, naRaiz, "entrada", 4)

	naOutra := c.novoProduto(t, subOutra, "Tubo PVC 100mm")
	c.movimentar(t, naOutra, "entrada", 4)

	produtos, err := c.publico.ListarProdutos(c.ctx, portsin.ListarProdutosPublicosInput{
		CategoriaSlug: raiz.Slug.Valor(),
	})
	if err != nil {
		t.Fatalf("listagem falhou: %v", err)
	}
	if len(produtos) != 1 {
		t.Fatalf("produtos da raiz = %d, esperado 1 (só o da subcategoria)", len(produtos))
	}
	if produtos[0].ID != naRaiz.ID {
		t.Errorf("produto = %s, esperado o da subcategoria da raiz", produtos[0].ID)
	}
}

func TestListarProdutosPublicosComCategoriaDesconhecidaFalha(t *testing.T) {
	c := novoCenario(t)

	if _, err := c.publico.ListarProdutos(c.ctx, portsin.ListarProdutosPublicosInput{
		CategoriaSlug: "categoria-que-nao-existe",
	}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erro = %v, esperado erro não encontrado", err)
	}
}

func TestListarProdutosPublicosIncorporaImagens(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))
	produto := c.novoProduto(t, sub, "Cimento CP II 50kg")
	c.movimentar(t, produto, "entrada", 4)

	primeira := c.novaImagem(t, produto, "image/png", 0)
	segunda := c.novaImagem(t, produto, "image/webp", 1)

	produtos, err := c.publico.ListarProdutos(c.ctx, portsin.ListarProdutosPublicosInput{})
	if err != nil {
		t.Fatalf("listagem falhou: %v", err)
	}
	if len(produtos) != 1 {
		t.Fatalf("produtos = %d, esperado 1", len(produtos))
	}

	imagens := produtos[0].Imagens
	if len(imagens) != 2 {
		t.Fatalf("imagens = %d, esperado 2", len(imagens))
	}
	if imagens[0].ID != primeira.ID || imagens[1].ID != segunda.ID {
		t.Errorf("ordem das imagens = %+v, esperado pela ordem declarada", imagens)
	}
}

func TestObterProdutoPorSlugPublico(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))
	produto := c.novoProduto(t, sub, "Cimento CP II 50kg")
	c.movimentar(t, produto, "entrada", 4)

	produtos, err := c.publico.ListarProdutos(c.ctx, portsin.ListarProdutosPublicosInput{})
	if err != nil {
		t.Fatalf("listagem falhou: %v", err)
	}
	if len(produtos) != 1 || produtos[0].ID != produto.ID {
		t.Fatalf("produto = %+v, esperado o criado na listagem pública", produtos)
	}

	detalhado, err := c.publico.ObterProdutoPorSlug(c.ctx, produto.Slug.Valor())
	if err != nil {
		t.Fatalf("detalhe falhou: %v", err)
	}
	if detalhado.ID != produto.ID {
		t.Errorf("produto = %s, esperado %s", detalhado.ID, produto.ID)
	}

	if _, err := c.publico.ObterProdutoPorSlug(c.ctx, "slug-que-nao-existe"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("slug desconhecido = %v, esperado erro não encontrado", err)
	}

	if _, err := c.publico.ObterProdutoPorSlug(c.ctx, produto.Slug.Valor()); err != nil {
		t.Fatalf("detalhe antes de esconder falhou: %v", err)
	}

	if err := c.produto.AlternarAtivo(c.ctx, portsin.AlternarAtivoProdutoInput{
		UsuarioID: c.usuarioID,
		ProdutoID: produto.ID,
		Ativo:     false,
	}); err != nil {
		t.Fatalf("desativação falhou: %v", err)
	}

	if _, err := c.publico.ObterProdutoPorSlug(c.ctx, produto.Slug.Valor()); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("produto inativo = %v, esperado erro não encontrado", err)
	}
}

func TestListarDestaquesPublicos(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))

	destacado := c.novoProduto(t, sub, "Cimento CP II 50kg")
	c.movimentar(t, destacado, "entrada", 4)
	if err := c.produto.AlternarDestaque(c.ctx, portsin.AlternarDestaqueProdutoInput{
		UsuarioID: c.usuarioID,
		ProdutoID: destacado.ID,
		Destaque:  true,
	}); err != nil {
		t.Fatalf("destaque falhou: %v", err)
	}

	semEstoque := c.novoProduto(t, sub, "Areia Fina")
	if err := c.produto.AlternarDestaque(c.ctx, portsin.AlternarDestaqueProdutoInput{
		UsuarioID: c.usuarioID,
		ProdutoID: semEstoque.ID,
		Destaque:  true,
	}); err != nil {
		t.Fatalf("destaque falhou: %v", err)
	}

	comum := c.novoProduto(t, sub, "Tijolo 8 furos")
	c.movimentar(t, comum, "entrada", 4)

	destaques, err := c.publico.ListarDestaques(c.ctx, domain.PaginacaoFiltro{})
	if err != nil {
		t.Fatalf("listagem de destaques falhou: %v", err)
	}
	if len(destaques) != 1 {
		t.Fatalf("destaques = %d, esperado 1", len(destaques))
	}
	if destaques[0].ID != destacado.ID {
		t.Errorf("destaque = %s, esperado o produto com estoque", destaques[0].ID)
	}
}

func TestVitrineEscondeCategoriaInativaESeusProdutos(t *testing.T) {
	c := novoCenario(t)
	raiz := c.novaCategoriaRaiz(t, "Materiais")
	sub := c.novaSubcategoria(t, "Alvenaria", raiz)
	produto := c.novoProduto(t, sub, "Cimento CP II 50kg")
	c.movimentar(t, produto, "entrada", 10)

	c.categoriasRepo.itens[raiz.ID].Ativo = false

	categorias, err := c.publico.ListarCategorias(c.ctx)
	if err != nil {
		t.Fatalf("listagem de categorias falhou: %v", err)
	}
	for _, categoria := range categorias {
		if categoria.ID == sub.ID {
			t.Error("subcategoria de pai inativo apareceu na vitrine")
		}
	}

	produtos, err := c.publico.ListarProdutos(c.ctx, portsin.ListarProdutosPublicosInput{})
	if err != nil {
		t.Fatalf("listagem de produtos falhou: %v", err)
	}
	if len(produtos) != 0 {
		t.Errorf("%d produtos de categoria inativa na vitrine, esperado nenhum", len(produtos))
	}

	if _, err := c.publico.ObterProdutoPorSlug(c.ctx, produto.Slug.Valor()); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("produto de categoria inativa = %v, esperado não encontrado", err)
	}
}
