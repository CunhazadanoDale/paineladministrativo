package estoque_test

import (
	"testing"

	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	"github.com/google/uuid"
)

func TestResumoConsolidaProdutosComPreco(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))

	preco := int64(2500)
	minimo := 10

	comPrecoID, err := c.produto.Criar(c.ctx, portsin.CriarProdutoInput{
		UsuarioID:     c.usuarioID,
		CategoriaID:   sub.ID,
		Nome:          "Cimento CP II 50kg",
		UnidadeMedida: "sc",
		PrecoCentavos: &preco,
		EstoqueMinimo: &minimo,
	})
	if err != nil {
		t.Fatalf("criação do produto com preço falhou: %v", err)
	}

	comPreco, err := c.produto.Obter(c.ctx, comPrecoID)
	if err != nil {
		t.Fatalf("busca do produto com preço falhou: %v", err)
	}

	sobConsulta, err := c.produto.Obter(c.ctx, c.novoProduto(t, sub, "Areia Fina").ID)
	if err != nil {
		t.Fatalf("busca do produto sob consulta falhou: %v", err)
	}

	c.movimentar(t, comPreco, "entrada", 8)
	c.movimentar(t, sobConsulta, "entrada", 4)

	resumo, err := c.resumo.Resumo(c.ctx)
	if err != nil {
		t.Fatalf("resumo falhou: %v", err)
	}

	if resumo.Produtos.TotalProdutos != 2 {
		t.Errorf("total de produtos = %d, esperado 2", resumo.Produtos.TotalProdutos)
	}
	if resumo.Produtos.ProdutosAtivos != 2 {
		t.Errorf("produtos ativos = %d, esperado 2", resumo.Produtos.ProdutosAtivos)
	}
	if resumo.Produtos.ValorEstoqueCentavos != 20000 {
		t.Errorf("valor em estoque = %d, esperado 20000 centavos", resumo.Produtos.ValorEstoqueCentavos)
	}
	if resumo.Produtos.ProdutosEstoqueBaixo != 1 {
		t.Errorf("produtos com estoque baixo = %d, esperado 1", resumo.Produtos.ProdutosEstoqueBaixo)
	}
	if len(resumo.UltimosMovimentos) != 2 {
		t.Fatalf("últimos movimentos = %d, esperado 2", len(resumo.UltimosMovimentos))
	}

	nomes := map[string]bool{}
	for _, movimento := range resumo.UltimosMovimentos {
		nomes[movimento.ProdutoNome] = true

		if movimento.ProdutoID == uuid.Nil {
			t.Error("movimento do resumo sem produto")
		}
		if movimento.ID == uuid.Nil {
			t.Error("movimento do resumo sem identificador")
		}
	}

	if !nomes["Cimento CP II 50kg"] || !nomes["Areia Fina"] {
		t.Errorf("nomes dos produtos = %v, esperado os dois produtos movimentados", nomes)
	}
}

func TestResumoIgnoraProdutosInativos(t *testing.T) {
	c := novoCenario(t)
	sub := c.novaSubcategoria(t, "Alvenaria", c.novaCategoriaRaiz(t, "Materiais"))

	preco := int64(3000)
	minimo := 5

	produtoID, err := c.produto.Criar(c.ctx, portsin.CriarProdutoInput{
		UsuarioID:     c.usuarioID,
		CategoriaID:   sub.ID,
		Nome:          "Tijolo 8 furos",
		UnidadeMedida: "un",
		PrecoCentavos: &preco,
		EstoqueMinimo: &minimo,
	})
	if err != nil {
		t.Fatalf("criação do produto falhou: %v", err)
	}

	produto, err := c.produto.Obter(c.ctx, produtoID)
	if err != nil {
		t.Fatalf("busca do produto falhou: %v", err)
	}

	c.movimentar(t, produto, "entrada", 3)

	if err := c.produto.AlternarAtivo(c.ctx, portsin.AlternarAtivoProdutoInput{
		UsuarioID: c.usuarioID,
		ProdutoID: produto.ID,
		Ativo:     false,
	}); err != nil {
		t.Fatalf("desativação do produto falhou: %v", err)
	}

	resumo, err := c.resumo.Resumo(c.ctx)
	if err != nil {
		t.Fatalf("resumo falhou: %v", err)
	}

	if resumo.Produtos.TotalProdutos != 1 {
		t.Errorf("total de produtos = %d, esperado 1", resumo.Produtos.TotalProdutos)
	}
	if resumo.Produtos.ProdutosAtivos != 0 {
		t.Errorf("produtos ativos = %d, esperado 0", resumo.Produtos.ProdutosAtivos)
	}
	if resumo.Produtos.ValorEstoqueCentavos != 0 {
		t.Errorf("valor em estoque = %d, esperado 0", resumo.Produtos.ValorEstoqueCentavos)
	}
	if resumo.Produtos.ProdutosEstoqueBaixo != 0 {
		t.Errorf("produtos com estoque baixo = %d, esperado 0", resumo.Produtos.ProdutosEstoqueBaixo)
	}
}

func TestResumoSemMovimentosDevolveListaVazia(t *testing.T) {
	c := novoCenario(t)

	resumo, err := c.resumo.Resumo(c.ctx)
	if err != nil {
		t.Fatalf("resumo falhou: %v", err)
	}

	if resumo.Produtos.TotalProdutos != 0 || resumo.Produtos.ValorEstoqueCentavos != 0 {
		t.Error("resumo sem estoque deveria devolver totais zerados")
	}
	if resumo.UltimosMovimentos == nil {
		t.Fatal("resumo deveria devolver lista de movimentos vazia e não nula")
	}
	if len(resumo.UltimosMovimentos) != 0 {
		t.Errorf("últimos movimentos = %d, esperado 0", len(resumo.UltimosMovimentos))
	}
}
