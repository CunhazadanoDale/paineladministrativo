package financeiro_test

import (
	"testing"

	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	"github.com/google/uuid"
)

func TestMontarPlanoAgrupaSubcategoriasNaOrdemRecebida(t *testing.T) {
	despesas := raizDeTeste(t)
	receitas, err := domainfinanceiro.NovaCategoriaFinanceira(
		domainfinanceiro.DadosCategoriaFinanceira{Nome: "Receitas"},
		classificacaoDeTeste(t, "receita", "receita_operacional"),
		nil,
	)
	if err != nil {
		t.Fatalf("raiz de receitas inválida: %v", err)
	}
	aluguel := subcategoriaDeTeste(t, despesas)
	energia, err := domainfinanceiro.NovaCategoriaFinanceira(domainfinanceiro.DadosCategoriaFinanceira{Nome: "Energia"}, nil, despesas)
	if err != nil {
		t.Fatalf("subcategoria de energia inválida: %v", err)
	}
	orfaID := uuid.New()
	orfa := &domainfinanceiro.CategoriaFinanceira{ID: uuid.New(), Nome: "Órfã", CategoriaPaiID: &orfaID}

	plano := domainfinanceiro.MontarPlano([]*domainfinanceiro.CategoriaFinanceira{energia, receitas, orfa, despesas, aluguel})

	if len(plano) != 2 || plano[0].Categoria != receitas || plano[1].Categoria != despesas {
		t.Fatalf("raízes do plano = %+v", plano)
	}
	if plano[0].Subcategorias == nil || len(plano[0].Subcategorias) != 0 {
		t.Errorf("raiz sem filhas deveria ter lista vazia, veio %v", plano[0].Subcategorias)
	}
	if len(plano[1].Subcategorias) != 2 || plano[1].Subcategorias[0] != energia || plano[1].Subcategorias[1] != aluguel {
		t.Errorf("subcategorias de despesas = %v", plano[1].Subcategorias)
	}
}

func TestMontarPlanoVazio(t *testing.T) {
	plano := domainfinanceiro.MontarPlano(nil)
	if plano == nil || len(plano) != 0 {
		t.Errorf("plano vazio = %v", plano)
	}
}
