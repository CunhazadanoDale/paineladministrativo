package estoque

import "testing"

func TestResumoDeNuncaDevolveMovimentosNulos(t *testing.T) {
	resumo := ResumoDe(ResumoProdutos{TotalProdutos: 3}, nil)

	if resumo.UltimosMovimentos == nil {
		t.Fatal("resumo deveria devolver lista de movimentos vazia e não nula")
	}
	if len(resumo.UltimosMovimentos) != 0 {
		t.Errorf("últimos movimentos = %d, esperado 0", len(resumo.UltimosMovimentos))
	}
	if resumo.Produtos.TotalProdutos != 3 {
		t.Errorf("total de produtos = %d, esperado 3", resumo.Produtos.TotalProdutos)
	}
}

func TestResumoDeCopiaAListaDeMovimentos(t *testing.T) {
	movimentos := []*MovimentoResumo{{ProdutoNome: "Cimento CP II 50kg"}}

	resumo := ResumoDe(ResumoProdutos{}, movimentos)
	movimentos[0] = &MovimentoResumo{ProdutoNome: "Areia Fina"}

	if resumo.UltimosMovimentos[0].ProdutoNome != "Cimento CP II 50kg" {
		t.Errorf("produto = %q, esperado o movimento original", resumo.UltimosMovimentos[0].ProdutoNome)
	}
}
