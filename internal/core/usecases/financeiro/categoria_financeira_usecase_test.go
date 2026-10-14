package financeiro_test

import (
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/financeiro"
	"github.com/google/uuid"
)

func (c *cenario) novaRaiz(t *testing.T, nome, natureza, linha string, ordem int) *domainfinanceiro.CategoriaFinanceira {
	t.Helper()

	raiz, err := c.categorias.Criar(c.ctx, portsin.CriarCategoriaFinanceiraInput{
		UsuarioID: c.financeiroMIA,
		Natureza:  texto(natureza),
		LinhaDRE:  texto(linha),
		Dados:     portsin.DadosCategoriaFinanceiraInput{Nome: nome, Ordem: ordem},
	})
	if err != nil {
		t.Fatalf("criação da raiz %q falhou: %v", nome, err)
	}

	return raiz
}

func (c *cenario) novaSub(t *testing.T, pai *domainfinanceiro.CategoriaFinanceira, nome string) *domainfinanceiro.CategoriaFinanceira {
	t.Helper()

	sub, err := c.categorias.Criar(c.ctx, portsin.CriarCategoriaFinanceiraInput{
		UsuarioID:      c.financeiroMIA,
		CategoriaPaiID: &pai.ID,
		Dados:          portsin.DadosCategoriaFinanceiraInput{Nome: nome},
	})
	if err != nil {
		t.Fatalf("criação da subcategoria %q falhou: %v", nome, err)
	}

	return sub
}

func (c *cenario) situacaoDaCategoria(usuarioID uuid.UUID, categoria *domainfinanceiro.CategoriaFinanceira) portsin.MudarSituacaoCategoriaFinanceiraInput {
	return portsin.MudarSituacaoCategoriaFinanceiraInput{
		UsuarioID:   usuarioID,
		CategoriaID: categoria.ID,
		Versao:      c.categoriasRepo.itens[categoria.ID].Versao,
	}
}

func TestCriarCategoriaRaiz(t *testing.T) {
	c := novoCenario(t)

	raiz, err := c.categorias.Criar(c.ctx, portsin.CriarCategoriaFinanceiraInput{
		UsuarioID: c.financeiroMIA,
		Natureza:  texto(" Despesa "),
		LinhaDRE:  texto("DESPESA_FIXA"),
		Dados:     portsin.DadosCategoriaFinanceiraInput{Nome: " Despesas administrativas ", Codigo: texto("4.2"), Ordem: 3},
	})
	if err != nil {
		t.Fatalf("Criar falhou: %v", err)
	}

	if raiz.EhSubcategoria() || raiz.Natureza != domainfinanceiro.NaturezaDespesa || raiz.LinhaDRE != domainfinanceiro.LinhaDespesaFixa {
		t.Errorf("raiz criada = %+v", raiz)
	}
	if raiz.Nome != "Despesas administrativas" || *raiz.Codigo != "4.2" || raiz.Ordem != 3 {
		t.Errorf("dados = %q %v %d", raiz.Nome, raiz.Codigo, raiz.Ordem)
	}
	if _, ok := c.categoriasRepo.itens[raiz.ID]; !ok {
		t.Error("raiz não foi gravada")
	}
}

func TestCriarCategoriaRaizExigeClassificacaoValida(t *testing.T) {
	c := novoCenario(t)
	dados := portsin.DadosCategoriaFinanceiraInput{Nome: "Despesas"}

	casos := []struct {
		natureza *string
		linha    *string
		trecho   string
	}{
		{nil, nil, "natureza e linha da DRE são obrigatórias na categoria raiz"},
		{texto("despesa"), nil, "linha da DRE deve ser"},
		{nil, texto("despesa_fixa"), "natureza deve ser"},
		{texto("receita"), texto("despesa_fixa"), "linha da DRE despesa_fixa não aceita natureza receita"},
	}

	for _, caso := range casos {
		_, err := c.categorias.Criar(c.ctx, portsin.CriarCategoriaFinanceiraInput{
			UsuarioID: c.financeiroMIA, Natureza: caso.natureza, LinhaDRE: caso.linha, Dados: dados,
		})
		esperarErro(t, err, domain.ErrValidacao, caso.trecho)
	}

	if len(c.categoriasRepo.itens) != 0 {
		t.Fatalf("nenhuma categoria deveria ter sido gravada, há %d", len(c.categoriasRepo.itens))
	}
}

func TestCriarSubcategoriaHerdaDoPai(t *testing.T) {
	c := novoCenario(t)
	raiz := c.novaRaiz(t, "Financeiro", "receita", "resultado_financeiro", 0)

	sub := c.novaSub(t, raiz, "Rendimentos")

	if !sub.Lancavel() || *sub.CategoriaPaiID != raiz.ID {
		t.Errorf("subcategoria criada = %+v", sub)
	}
	if sub.Natureza != domainfinanceiro.NaturezaReceita || sub.LinhaDRE != domainfinanceiro.LinhaResultadoFinanceiro {
		t.Errorf("classificação = %s/%s", sub.Natureza, sub.LinhaDRE)
	}
}

func TestCriarSubcategoriaRecusaClassificacaoEPaiInvalido(t *testing.T) {
	c := novoCenario(t)
	raiz := c.novaRaiz(t, "Despesas", "despesa", "despesa_fixa", 0)
	sub := c.novaSub(t, raiz, "Aluguel")
	inativa := c.novaRaiz(t, "Antigas", "despesa", "despesa_fixa", 0)
	if _, err := c.categorias.Desativar(c.ctx, c.situacaoDaCategoria(c.financeiroMIA, inativa)); err != nil {
		t.Fatalf("Desativar falhou: %v", err)
	}

	_, err := c.categorias.Criar(c.ctx, portsin.CriarCategoriaFinanceiraInput{
		UsuarioID: c.financeiroMIA, CategoriaPaiID: &raiz.ID, LinhaDRE: texto("despesa_fixa"),
		Dados: portsin.DadosCategoriaFinanceiraInput{Nome: "Energia"},
	})
	esperarErro(t, err, domain.ErrValidacao, "subcategoria herda natureza e linha da DRE da categoria pai")

	inexistente := uuid.New()
	for _, paiID := range []uuid.UUID{inexistente, sub.ID, inativa.ID} {
		_, err := c.categorias.Criar(c.ctx, portsin.CriarCategoriaFinanceiraInput{
			UsuarioID: c.financeiroMIA, CategoriaPaiID: &paiID,
			Dados: portsin.DadosCategoriaFinanceiraInput{Nome: "Energia"},
		})
		esperarErro(t, err, domain.ErrValidacao, "categoria pai deve ser uma categoria raiz ativa")
	}
}

func TestCriarCategoriaExigeFinanceiroEConfereDuplicidade(t *testing.T) {
	c := novoCenario(t)
	c.novaRaiz(t, "Despesas", "despesa", "despesa_fixa", 0)

	_, err := c.categorias.Criar(c.ctx, portsin.CriarCategoriaFinanceiraInput{
		UsuarioID: c.comum, Natureza: texto("despesa"), LinhaDRE: texto("despesa_fixa"),
		Dados: portsin.DadosCategoriaFinanceiraInput{Nome: "Outra"},
	})
	esperarErro(t, err, domain.ErrPermissao, "perfil sem permissão para o financeiro")

	_, err = c.categorias.Criar(c.ctx, portsin.CriarCategoriaFinanceiraInput{
		UsuarioID: c.administrador, Natureza: texto("receita"), LinhaDRE: texto("fora_dre"),
		Dados: portsin.DadosCategoriaFinanceiraInput{Nome: "DESPESAS"},
	})
	esperarErro(t, err, domain.ErrConflito, "já existe uma categoria com este nome neste nível")
}

func TestObterEListarCategorias(t *testing.T) {
	c := novoCenario(t)
	raiz := c.novaRaiz(t, "Despesas", "despesa", "despesa_fixa", 0)

	lida, err := c.categorias.Obter(c.ctx, raiz.ID)
	if err != nil || lida.ID != raiz.ID {
		t.Fatalf("Obter = %v, %v", lida, err)
	}
	_, err = c.categorias.Obter(c.ctx, uuid.New())
	esperarErro(t, err, domain.ErrNotFound, "categoria financeira não encontrada")

	ativa := true
	_, err = c.categorias.Listar(c.ctx, portsin.ListarCategoriasFinanceirasInput{
		Pai: &raiz.ID, SomenteRaiz: true, Natureza: "DESPESA", LinhaDRE: "despesa_fixa", Ativa: &ativa, Busca: "alu",
		Filtro: domain.PaginacaoFiltro{Size: 1000},
	})
	if err != nil {
		t.Fatalf("Listar falhou: %v", err)
	}
	filtro := c.categoriasRepo.ultimoFiltro
	if *filtro.Pai != raiz.ID || !filtro.SomenteRaiz || *filtro.Natureza != domainfinanceiro.NaturezaDespesa ||
		*filtro.LinhaDRE != domainfinanceiro.LinhaDespesaFixa || !*filtro.Ativa || filtro.Busca != "alu" || filtro.Size != domain.TamanhoPaginaMaximo {
		t.Errorf("filtro = %+v", filtro)
	}

	_, err = c.categorias.Listar(c.ctx, portsin.ListarCategoriasFinanceirasInput{Natureza: "custo"})
	esperarErro(t, err, domain.ErrValidacao, "natureza deve ser")
	_, err = c.categorias.Listar(c.ctx, portsin.ListarCategoriasFinanceirasInput{LinhaDRE: "cmv"})
	esperarErro(t, err, domain.ErrValidacao, "linha da DRE deve ser")
}

func TestPlanoDeContasEmArvore(t *testing.T) {
	c := novoCenario(t)
	despesas := c.novaRaiz(t, "Despesas", "despesa", "despesa_fixa", 2)
	receitas := c.novaRaiz(t, "Receitas", "receita", "receita_operacional", 1)
	aluguel := c.novaSub(t, despesas, "Aluguel")
	energia := c.novaSub(t, despesas, "Energia")
	if _, err := c.categorias.Desativar(c.ctx, c.situacaoDaCategoria(c.financeiroMIA, energia)); err != nil {
		t.Fatalf("Desativar falhou: %v", err)
	}

	plano, err := c.categorias.Plano(c.ctx, false)
	if err != nil {
		t.Fatalf("Plano falhou: %v", err)
	}
	if len(plano) != 2 || plano[0].Categoria.ID != receitas.ID || len(plano[1].Subcategorias) != 2 {
		t.Fatalf("plano = %+v", plano)
	}

	ativo, err := c.categorias.Plano(c.ctx, true)
	if err != nil {
		t.Fatalf("Plano ativo falhou: %v", err)
	}
	if len(ativo[1].Subcategorias) != 1 || ativo[1].Subcategorias[0].ID != aluguel.ID {
		t.Errorf("subcategorias ativas = %v", ativo[1].Subcategorias)
	}
}

func TestAlterarCategoriaConfereVersaoEMantemClassificacao(t *testing.T) {
	c := novoCenario(t)
	raiz := c.novaRaiz(t, "Despesas", "despesa", "despesa_fixa", 0)
	dados := portsin.DadosCategoriaFinanceiraInput{Nome: "Despesas fixas", Codigo: texto("4.1"), Ordem: 2}

	_, err := c.categorias.Alterar(c.ctx, portsin.AlterarCategoriaFinanceiraInput{UsuarioID: c.financeiroMIA, CategoriaID: raiz.ID, Dados: dados})
	esperarErro(t, err, domain.ErrValidacao, "versão da categoria financeira é obrigatória")

	_, err = c.categorias.Alterar(c.ctx, portsin.AlterarCategoriaFinanceiraInput{UsuarioID: c.financeiroMIA, CategoriaID: raiz.ID, Dados: dados, Versao: 5})
	esperarErro(t, err, domain.ErrConflito, "alterada por outra pessoa")

	_, err = c.categorias.Alterar(c.ctx, portsin.AlterarCategoriaFinanceiraInput{UsuarioID: c.comum, CategoriaID: raiz.ID, Dados: dados, Versao: 1})
	esperarErro(t, err, domain.ErrPermissao, "perfil sem permissão para o financeiro")

	alterada, err := c.categorias.Alterar(c.ctx, portsin.AlterarCategoriaFinanceiraInput{UsuarioID: c.financeiroMIA, CategoriaID: raiz.ID, Dados: dados, Versao: 1})
	if err != nil {
		t.Fatalf("Alterar falhou: %v", err)
	}
	if alterada.Nome != "Despesas fixas" || *alterada.Codigo != "4.1" || alterada.Ordem != 2 || alterada.Versao != 2 {
		t.Errorf("categoria alterada = %+v", alterada)
	}
	if alterada.Natureza != domainfinanceiro.NaturezaDespesa || alterada.LinhaDRE != domainfinanceiro.LinhaDespesaFixa {
		t.Error("alterar não pode mudar a classificação")
	}
}

func TestDesativarRaizComSubcategoriaAtiva(t *testing.T) {
	c := novoCenario(t)
	raiz := c.novaRaiz(t, "Despesas", "despesa", "despesa_fixa", 0)
	sub := c.novaSub(t, raiz, "Aluguel")

	_, err := c.categorias.Desativar(c.ctx, c.situacaoDaCategoria(c.financeiroMIA, raiz))
	esperarErro(t, err, domain.ErrConflito, "categoria possui subcategorias ativas")

	if _, err := c.categorias.Desativar(c.ctx, c.situacaoDaCategoria(c.financeiroMIA, sub)); err != nil {
		t.Fatalf("Desativar subcategoria falhou: %v", err)
	}
	desativada, err := c.categorias.Desativar(c.ctx, c.situacaoDaCategoria(c.financeiroMIA, raiz))
	if err != nil || desativada.Ativa || desativada.Versao != 2 {
		t.Fatalf("Desativar raiz = %+v, %v", desativada, err)
	}

	gravacoes := c.categoriasRepo.gravacoes
	repetida, err := c.categorias.Desativar(c.ctx, c.situacaoDaCategoria(c.financeiroMIA, raiz))
	if err != nil || repetida.Versao != 2 || c.categoriasRepo.gravacoes != gravacoes {
		t.Fatalf("desativar raiz inativa = %+v, %v", repetida, err)
	}
}

func TestAtivarSubcategoriaExigePaiAtivo(t *testing.T) {
	c := novoCenario(t)
	raiz := c.novaRaiz(t, "Despesas", "despesa", "despesa_fixa", 0)
	sub := c.novaSub(t, raiz, "Aluguel")
	for _, categoria := range []*domainfinanceiro.CategoriaFinanceira{sub, raiz} {
		if _, err := c.categorias.Desativar(c.ctx, c.situacaoDaCategoria(c.financeiroMIA, categoria)); err != nil {
			t.Fatalf("Desativar falhou: %v", err)
		}
	}

	_, err := c.categorias.Ativar(c.ctx, c.situacaoDaCategoria(c.financeiroMIA, sub))
	esperarErro(t, err, domain.ErrValidacao, "categoria pai está inativa")

	_, err = c.categorias.Ativar(c.ctx, portsin.MudarSituacaoCategoriaFinanceiraInput{UsuarioID: c.financeiroMIA, CategoriaID: raiz.ID, Versao: 1})
	esperarErro(t, err, domain.ErrConflito, "alterada por outra pessoa")

	if _, err := c.categorias.Ativar(c.ctx, c.situacaoDaCategoria(c.financeiroMIA, raiz)); err != nil {
		t.Fatalf("Ativar raiz falhou: %v", err)
	}
	ativada, err := c.categorias.Ativar(c.ctx, c.situacaoDaCategoria(c.financeiroMIA, sub))
	if err != nil || !ativada.Lancavel() || ativada.Versao != 3 {
		t.Fatalf("Ativar subcategoria = %+v, %v", ativada, err)
	}

	gravacoes := c.categoriasRepo.gravacoes
	repetida, err := c.categorias.Ativar(c.ctx, c.situacaoDaCategoria(c.financeiroMIA, sub))
	if err != nil || repetida.Versao != 3 || c.categoriasRepo.gravacoes != gravacoes {
		t.Fatalf("ativar subcategoria ativa = %+v, %v", repetida, err)
	}
}

func TestRemoverCategoriaSoPeloAdministrador(t *testing.T) {
	c := novoCenario(t)
	raiz := c.novaRaiz(t, "Despesas", "despesa", "despesa_fixa", 0)
	sub := c.novaSub(t, raiz, "Aluguel")

	esperarErro(t, c.categorias.Remover(c.ctx, sub.ID, c.financeiroMIA), domain.ErrPermissao, "perfil sem permissão para esta operação")
	esperarErro(t, c.categorias.Remover(c.ctx, raiz.ID, c.administrador), domain.ErrValidacao, "registro em uso")

	if err := c.categorias.Remover(c.ctx, sub.ID, c.administrador); err != nil {
		t.Fatalf("Remover subcategoria falhou: %v", err)
	}
	if err := c.categorias.Remover(c.ctx, raiz.ID, c.administrador); err != nil {
		t.Fatalf("Remover raiz falhou: %v", err)
	}
	esperarErro(t, c.categorias.Remover(c.ctx, raiz.ID, c.administrador), domain.ErrNotFound, "categoria financeira não encontrada")
}

func TestAlterarCategoriaRecusaDadosInvalidosECodigoRepetido(t *testing.T) {
	c := novoCenario(t)
	c.novaRaiz(t, "Receitas", "receita", "receita_operacional", 0)
	if _, err := c.categorias.Alterar(c.ctx, portsin.AlterarCategoriaFinanceiraInput{
		UsuarioID: c.financeiroMIA, CategoriaID: c.novaRaiz(t, "Outras", "receita", "fora_dre", 0).ID,
		Dados: portsin.DadosCategoriaFinanceiraInput{Nome: "Outras", Codigo: texto("3")}, Versao: 1,
	}); err != nil {
		t.Fatalf("Alterar falhou: %v", err)
	}
	despesas := c.novaRaiz(t, "Despesas", "despesa", "despesa_fixa", 0)

	_, err := c.categorias.Alterar(c.ctx, portsin.AlterarCategoriaFinanceiraInput{
		UsuarioID: c.financeiroMIA, CategoriaID: despesas.ID, Dados: portsin.DadosCategoriaFinanceiraInput{Nome: " "}, Versao: 1,
	})
	esperarErro(t, err, domain.ErrValidacao, "nome da categoria financeira é obrigatório")

	_, err = c.categorias.Alterar(c.ctx, portsin.AlterarCategoriaFinanceiraInput{
		UsuarioID: c.financeiroMIA, CategoriaID: despesas.ID,
		Dados: portsin.DadosCategoriaFinanceiraInput{Nome: "Despesas", Codigo: texto("3")}, Versao: 1,
	})
	esperarErro(t, err, domain.ErrConflito, "já existe uma categoria com este código")

	if gravada := c.categoriasRepo.itens[despesas.ID]; gravada.Codigo != nil || gravada.Versao != 1 {
		t.Errorf("categoria gravada = %+v", gravada)
	}
}
