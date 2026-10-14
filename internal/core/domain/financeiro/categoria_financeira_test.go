package financeiro_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	"github.com/google/uuid"
)

func classificacaoDeTeste(t *testing.T, natureza, linha string) *domainfinanceiro.Classificacao {
	t.Helper()

	classificacao, err := domainfinanceiro.NovaClassificacao(natureza, linha)
	if err != nil {
		t.Fatalf("classificação de teste inválida: %v", err)
	}

	return &classificacao
}

func raizDeTeste(t *testing.T) *domainfinanceiro.CategoriaFinanceira {
	t.Helper()

	raiz, err := domainfinanceiro.NovaCategoriaFinanceira(
		domainfinanceiro.DadosCategoriaFinanceira{Nome: "Despesas administrativas", Codigo: texto("4.2"), Ordem: 3},
		classificacaoDeTeste(t, "despesa", "despesa_fixa"),
		nil,
	)
	if err != nil {
		t.Fatalf("raiz de teste inválida: %v", err)
	}

	return raiz
}

func subcategoriaDeTeste(t *testing.T, raiz *domainfinanceiro.CategoriaFinanceira) *domainfinanceiro.CategoriaFinanceira {
	t.Helper()

	sub, err := domainfinanceiro.NovaCategoriaFinanceira(
		domainfinanceiro.DadosCategoriaFinanceira{Nome: "Aluguel", Codigo: texto("4.2.01")},
		nil,
		raiz,
	)
	if err != nil {
		t.Fatalf("subcategoria de teste inválida: %v", err)
	}

	return sub
}

func TestNovaCategoriaRaizUsaAClassificacao(t *testing.T) {
	raiz, err := domainfinanceiro.NovaCategoriaFinanceira(
		domainfinanceiro.DadosCategoriaFinanceira{Nome: "  Receitas financeiras ", Codigo: texto(" 3.9 "), Ordem: 2},
		classificacaoDeTeste(t, "receita", "resultado_financeiro"),
		nil,
	)
	if err != nil {
		t.Fatalf("NovaCategoriaFinanceira falhou: %v", err)
	}

	if raiz.ID == uuid.Nil || raiz.EhSubcategoria() || raiz.Lancavel() || !raiz.Ativa || raiz.Versao != 1 {
		t.Errorf("raiz criada = %+v", raiz)
	}
	if raiz.Nome != "Receitas financeiras" || raiz.Codigo == nil || *raiz.Codigo != "3.9" || raiz.Ordem != 2 {
		t.Errorf("dados da raiz = %q %v %d", raiz.Nome, raiz.Codigo, raiz.Ordem)
	}
	if raiz.Natureza != domainfinanceiro.NaturezaReceita || raiz.LinhaDRE != domainfinanceiro.LinhaResultadoFinanceiro {
		t.Errorf("classificação = %s/%s", raiz.Natureza, raiz.LinhaDRE)
	}
}

func TestNovaCategoriaRaizExigeClassificacao(t *testing.T) {
	dados := domainfinanceiro.DadosCategoriaFinanceira{Nome: "Despesas"}

	if _, err := domainfinanceiro.NovaCategoriaFinanceira(dados, nil, nil); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("sem classificação erro = %v", err)
	}
	if _, err := domainfinanceiro.NovaCategoriaFinanceira(dados, &domainfinanceiro.Classificacao{}, nil); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("classificação vazia erro = %v", err)
	}
}

func TestNovaSubcategoriaHerdaDoPai(t *testing.T) {
	raiz := raizDeTeste(t)

	sub := subcategoriaDeTeste(t, raiz)

	if !sub.EhSubcategoria() || !sub.Lancavel() || *sub.CategoriaPaiID != raiz.ID {
		t.Errorf("subcategoria criada = %+v", sub)
	}
	if sub.Natureza != raiz.Natureza || sub.LinhaDRE != raiz.LinhaDRE {
		t.Errorf("subcategoria não herdou: %s/%s", sub.Natureza, sub.LinhaDRE)
	}
	if sub.Codigo == nil || *sub.Codigo != "4.2.01" || sub.Ordem != 0 {
		t.Errorf("dados da subcategoria = %v %d", sub.Codigo, sub.Ordem)
	}
}

func TestNovaSubcategoriaRecusaClassificacaoPropria(t *testing.T) {
	_, err := domainfinanceiro.NovaCategoriaFinanceira(
		domainfinanceiro.DadosCategoriaFinanceira{Nome: "Aluguel"},
		classificacaoDeTeste(t, "despesa", "despesa_fixa"),
		raizDeTeste(t),
	)

	if !errors.Is(err, domain.ErrValidacao) || !strings.Contains(err.Error(), "herda natureza e linha da DRE") {
		t.Errorf("erro = %v", err)
	}
}

func TestNovaSubcategoriaExigePaiRaizAtivo(t *testing.T) {
	raiz := raizDeTeste(t)
	sub := subcategoriaDeTeste(t, raiz)
	dados := domainfinanceiro.DadosCategoriaFinanceira{Nome: "Terceiro nível"}

	if _, err := domainfinanceiro.NovaCategoriaFinanceira(dados, nil, sub); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("pai subcategoria erro = %v", err)
	}

	raiz.Ativa = false
	if _, err := domainfinanceiro.NovaCategoriaFinanceira(dados, nil, raiz); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("pai inativo erro = %v", err)
	}
}

func TestCategoriaFinanceiraValidaDados(t *testing.T) {
	casos := map[string]domainfinanceiro.DadosCategoriaFinanceira{
		"sem nome":          {Nome: "  "},
		"nome muito longo":  {Nome: strings.Repeat("a", 121)},
		"ordem negativa":    {Nome: "Despesas", Ordem: -1},
		"ordem fora do int": {Nome: "Despesas", Ordem: 2147483648},
		"código com letra":  {Nome: "Despesas", Codigo: texto("4.A")},
		"código com vazio":  {Nome: "Despesas", Codigo: texto("4..1")},
		"código com ponto":  {Nome: "Despesas", Codigo: texto(".4")},
		"código final":      {Nome: "Despesas", Codigo: texto("4.")},
		"código longo":      {Nome: "Despesas", Codigo: texto("1.2.3.4.5.6.7.8.9.10.")},
		"código 21":         {Nome: "Despesas", Codigo: texto(strings.Repeat("1", 21))},
	}

	for nome, dados := range casos {
		t.Run(nome, func(t *testing.T) {
			_, err := domainfinanceiro.NovaCategoriaFinanceira(dados, classificacaoDeTeste(t, "despesa", "despesa_fixa"), nil)
			if !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("erro = %v", err)
			}
		})
	}
}

func TestCategoriaFinanceiraCodigoEmBrancoViraNulo(t *testing.T) {
	raiz, err := domainfinanceiro.NovaCategoriaFinanceira(
		domainfinanceiro.DadosCategoriaFinanceira{Nome: "Despesas", Codigo: texto("   ")},
		classificacaoDeTeste(t, "despesa", "despesa_fixa"),
		nil,
	)
	if err != nil {
		t.Fatalf("NovaCategoriaFinanceira falhou: %v", err)
	}
	if raiz.Codigo != nil {
		t.Errorf("código = %q", *raiz.Codigo)
	}
}

func TestCategoriaFinanceiraAlterarMantemClassificacao(t *testing.T) {
	raiz := raizDeTeste(t)
	agora := raiz.CriadoEm.Add(time.Hour)

	err := raiz.Alterar(domainfinanceiro.DadosCategoriaFinanceira{Nome: "Administrativo", Ordem: 7}, agora)
	if err != nil {
		t.Fatalf("Alterar falhou: %v", err)
	}

	if raiz.Nome != "Administrativo" || raiz.Codigo != nil || raiz.Ordem != 7 || !raiz.AtualizadoEm.Equal(agora) {
		t.Errorf("raiz alterada = %+v", raiz)
	}
	if raiz.Natureza != domainfinanceiro.NaturezaDespesa || raiz.LinhaDRE != domainfinanceiro.LinhaDespesaFixa {
		t.Error("alterar não pode mudar a classificação")
	}

	if err := raiz.Alterar(domainfinanceiro.DadosCategoriaFinanceira{Nome: ""}, agora.Add(time.Hour)); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("nome vazio erro = %v", err)
	}
	if raiz.Nome != "Administrativo" || !raiz.AtualizadoEm.Equal(agora) {
		t.Error("dados inválidos não podem alterar a categoria")
	}
}

func TestCategoriaFinanceiraDesativar(t *testing.T) {
	raiz := raizDeTeste(t)
	agora := raiz.CriadoEm.Add(time.Hour)

	mudou, err := raiz.Desativar(true, agora)
	if !errors.Is(err, domain.ErrConflito) || mudou || !raiz.Ativa {
		t.Errorf("com subcategorias ativas = %v/%v", mudou, err)
	}

	mudou, err = raiz.Desativar(false, agora)
	if err != nil || !mudou || raiz.Ativa || !raiz.AtualizadoEm.Equal(agora) {
		t.Errorf("sem subcategorias ativas = %v/%v", mudou, err)
	}

	mudou, err = raiz.Desativar(true, agora.Add(time.Hour))
	if err != nil || mudou || !raiz.AtualizadoEm.Equal(agora) {
		t.Errorf("já inativa = %v/%v", mudou, err)
	}
}

func TestCategoriaFinanceiraAtivarSubcategoriaExigePaiAtivo(t *testing.T) {
	raiz := raizDeTeste(t)
	sub := subcategoriaDeTeste(t, raiz)
	agora := sub.CriadoEm.Add(time.Hour)

	if mudou, err := sub.Ativar(raiz, agora); err != nil || mudou {
		t.Errorf("já ativa = %v/%v", mudou, err)
	}

	if _, err := sub.Desativar(false, agora); err != nil {
		t.Fatalf("Desativar falhou: %v", err)
	}
	if sub.Lancavel() {
		t.Error("subcategoria inativa não é lançável")
	}

	raiz.Ativa = false
	if mudou, err := sub.Ativar(raiz, agora); !errors.Is(err, domain.ErrValidacao) || mudou || sub.Ativa {
		t.Errorf("pai inativo = %v/%v", mudou, err)
	}

	if _, err := sub.Ativar(nil, agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("sem pai erro = %v", err)
	}
	if _, err := sub.Ativar(raizDeTeste(t), agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("pai de outra categoria erro = %v", err)
	}

	raiz.Ativa = true
	if mudou, err := sub.Ativar(raiz, agora); err != nil || !mudou || !sub.Lancavel() {
		t.Errorf("pai ativo = %v/%v", mudou, err)
	}
}

func TestCategoriaFinanceiraAtivarRaizNaoPrecisaDePai(t *testing.T) {
	raiz := raizDeTeste(t)
	agora := raiz.CriadoEm.Add(time.Hour)

	if _, err := raiz.Desativar(false, agora); err != nil {
		t.Fatalf("Desativar falhou: %v", err)
	}
	if mudou, err := raiz.Ativar(nil, agora); err != nil || !mudou || !raiz.Ativa {
		t.Errorf("ativar raiz = %v/%v", mudou, err)
	}
}

func TestCategoriaFinanceiraConferirVersao(t *testing.T) {
	raiz := raizDeTeste(t)

	if err := raiz.ConferirVersao(1); err != nil {
		t.Errorf("versão atual erro = %v", err)
	}
	if err := raiz.ConferirVersao(-1); !errors.Is(err, domain.ErrValidacao) || !strings.Contains(err.Error(), "versão da categoria financeira é obrigatória") {
		t.Errorf("versão ausente erro = %v", err)
	}
	if err := raiz.ConferirVersao(3); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("versão divergente erro = %v", err)
	}
}
