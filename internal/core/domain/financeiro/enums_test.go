package financeiro_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
)

func TestNovoBancoAceitaOCatalogo(t *testing.T) {
	esperados := map[string]string{
		"001": "Banco do Brasil",
		"033": "Santander",
		"077": "Inter",
		"237": "Bradesco",
		"341": "Itaú",
		"403": "Cora",
		"748": "Sicredi",
	}

	for codigo, nome := range esperados {
		banco, err := domainfinanceiro.NovoBanco(" " + codigo + " ")
		if err != nil {
			t.Fatalf("NovoBanco(%q) falhou: %v", codigo, err)
		}
		if banco.String() != codigo || banco.Nome() != nome {
			t.Errorf("banco %q = %q/%q", codigo, banco.String(), banco.Nome())
		}
	}

	if len(domainfinanceiro.Bancos()) != len(esperados) {
		t.Errorf("catálogo com %d bancos", len(domainfinanceiro.Bancos()))
	}
}

func TestBancosDevolveOCatalogoEmOrdemDeCodigo(t *testing.T) {
	bancos := domainfinanceiro.Bancos()
	for i := 1; i < len(bancos); i++ {
		if bancos[i-1].String() >= bancos[i].String() {
			t.Fatalf("catálogo fora de ordem: %v", bancos)
		}
	}

	bancos[0] = domainfinanceiro.BancoDe("999")
	if domainfinanceiro.Bancos()[0] != domainfinanceiro.BancoDoBrasil {
		t.Error("alterar a cópia não pode mudar o catálogo")
	}
}

func TestNovoBancoRecusaForaDoCatalogo(t *testing.T) {
	for _, valor := range []string{"", "1", "104", "abc", "0001"} {
		if _, err := domainfinanceiro.NovoBanco(valor); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("NovoBanco(%q) erro = %v", valor, err)
		}
	}

	if domainfinanceiro.BancoDe("104").Nome() != "" {
		t.Error("banco fora do catálogo não tem nome")
	}
}

func TestNovoTipoConta(t *testing.T) {
	for _, valor := range []string{"corrente", " Poupanca ", "PAGAMENTO", "aplicacao"} {
		if _, err := domainfinanceiro.NovoTipoConta(valor); err != nil {
			t.Errorf("NovoTipoConta(%q) falhou: %v", valor, err)
		}
	}

	for _, valor := range []string{"", "caixa", "poupança"} {
		if _, err := domainfinanceiro.NovoTipoConta(valor); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("NovoTipoConta(%q) erro = %v", valor, err)
		}
	}
}

func TestNovaNatureza(t *testing.T) {
	for _, valor := range []string{"receita", " Despesa ", "TRANSFERENCIA"} {
		if _, err := domainfinanceiro.NovaNatureza(valor); err != nil {
			t.Errorf("NovaNatureza(%q) falhou: %v", valor, err)
		}
	}

	for _, valor := range []string{"", "custo", "transferência"} {
		if _, err := domainfinanceiro.NovaNatureza(valor); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("NovaNatureza(%q) erro = %v", valor, err)
		}
	}
}

func TestNovaLinhaDRE(t *testing.T) {
	validas := []string{
		"receita_operacional", "deducao_venda", "compra_mercadoria", "despesa_variavel",
		"despesa_fixa", "resultado_financeiro", "fora_dre", " ENTRE_CONTAS ",
	}
	for _, valor := range validas {
		if _, err := domainfinanceiro.NovaLinhaDRE(valor); err != nil {
			t.Errorf("NovaLinhaDRE(%q) falhou: %v", valor, err)
		}
	}

	for _, valor := range []string{"", "operacional", "cmv"} {
		if _, err := domainfinanceiro.NovaLinhaDRE(valor); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("NovaLinhaDRE(%q) erro = %v", valor, err)
		}
	}
}

func TestNovaClassificacaoAmarraNaturezaELinha(t *testing.T) {
	aceitas := map[string][]string{
		"receita_operacional":  {"receita"},
		"deducao_venda":        {"despesa"},
		"compra_mercadoria":    {"despesa"},
		"despesa_variavel":     {"despesa"},
		"despesa_fixa":         {"despesa"},
		"resultado_financeiro": {"receita", "despesa"},
		"fora_dre":             {"receita", "despesa"},
		"entre_contas":         {"transferencia"},
	}

	for linha, naturezas := range aceitas {
		for _, natureza := range []string{"receita", "despesa", "transferencia"} {
			classificacao, err := domainfinanceiro.NovaClassificacao(natureza, linha)
			deveAceitar := contem(naturezas, natureza)

			if deveAceitar && err != nil {
				t.Errorf("%s/%s deveria ser aceita: %v", natureza, linha, err)
			}
			if !deveAceitar && !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("%s/%s deveria ser recusada, erro = %v", natureza, linha, err)
			}
			if deveAceitar && (classificacao.Natureza().String() != natureza || classificacao.LinhaDRE().String() != linha) {
				t.Errorf("classificação %s/%s = %+v", natureza, linha, classificacao)
			}
		}
	}
}

func TestNovaClassificacaoValidaCadaCampo(t *testing.T) {
	if _, err := domainfinanceiro.NovaClassificacao("custo", "despesa_fixa"); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("natureza inválida erro = %v", err)
	}
	if _, err := domainfinanceiro.NovaClassificacao("despesa", "operacional"); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("linha inválida erro = %v", err)
	}
}

func contem(lista []string, valor string) bool {
	for _, item := range lista {
		if item == valor {
			return true
		}
	}

	return false
}
