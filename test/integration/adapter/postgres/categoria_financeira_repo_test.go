package postgres_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/financeiro"
	"github.com/CunhazadanoDale/paineladministrativo.git/test/helpers"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type cenarioBancoPlano struct {
	ctx        context.Context
	banco      *sqlx.DB
	categorias *postgres.CategoriaFinanceiraRepository
}

func novoCenarioPlano(t *testing.T) *cenarioBancoPlano {
	t.Helper()

	banco := helpers.BancoDoTeste(t)

	return &cenarioBancoPlano{
		ctx:        context.Background(),
		banco:      banco,
		categorias: postgres.NewCategoriaFinanceiraRepository(banco),
	}
}

func codigoDe(valor string) *string {
	return &valor
}

func (c *cenarioBancoPlano) novaRaiz(t *testing.T, nome, natureza, linha string, ordem int, codigo *string) *domainfinanceiro.CategoriaFinanceira {
	t.Helper()

	classificacao, err := domainfinanceiro.NovaClassificacao(natureza, linha)
	if err != nil {
		t.Fatalf("classificação de teste inválida: %v", err)
	}
	raiz, err := domainfinanceiro.NovaCategoriaFinanceira(
		domainfinanceiro.DadosCategoriaFinanceira{Nome: nome, Codigo: codigo, Ordem: ordem}, &classificacao, nil,
	)
	if err != nil {
		t.Fatalf("raiz de teste inválida: %v", err)
	}
	if err := c.categorias.Criar(c.ctx, raiz); err != nil {
		t.Fatalf("criação da raiz %q falhou: %v", nome, err)
	}

	return raiz
}

func (c *cenarioBancoPlano) subDeDominio(t *testing.T, pai *domainfinanceiro.CategoriaFinanceira, nome string, ordem int, codigo *string) *domainfinanceiro.CategoriaFinanceira {
	t.Helper()

	sub, err := domainfinanceiro.NovaCategoriaFinanceira(
		domainfinanceiro.DadosCategoriaFinanceira{Nome: nome, Codigo: codigo, Ordem: ordem}, nil, pai,
	)
	if err != nil {
		t.Fatalf("subcategoria de teste inválida: %v", err)
	}

	return sub
}

func (c *cenarioBancoPlano) novaSub(t *testing.T, pai *domainfinanceiro.CategoriaFinanceira, nome string, ordem int, codigo *string) *domainfinanceiro.CategoriaFinanceira {
	t.Helper()

	sub := c.subDeDominio(t, pai, nome, ordem, codigo)
	if err := c.categorias.Criar(c.ctx, sub); err != nil {
		t.Fatalf("criação da subcategoria %q falhou: %v", nome, err)
	}

	return sub
}

func (c *cenarioBancoPlano) obter(t *testing.T, id uuid.UUID) *domainfinanceiro.CategoriaFinanceira {
	t.Helper()

	categoria, err := c.categorias.Obter(c.ctx, id)
	if err != nil || categoria == nil {
		t.Fatalf("Obter(%s) = %v, %v", id, categoria, err)
	}

	return categoria
}

func (c *cenarioBancoPlano) desativar(t *testing.T, categoria *domainfinanceiro.CategoriaFinanceira) {
	t.Helper()

	if _, err := categoria.Desativar(false, time.Now().UTC()); err != nil {
		t.Fatalf("Desativar de domínio falhou: %v", err)
	}
	if err := c.categorias.Desativar(c.ctx, categoria); err != nil {
		t.Fatalf("Desativar falhou: %v", err)
	}
}

func idsDasCategorias(categorias []*domainfinanceiro.CategoriaFinanceira) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(categorias))
	for _, categoria := range categorias {
		ids = append(ids, categoria.ID)
	}

	return ids
}

func conferirIDs(t *testing.T, obtidos []uuid.UUID, esperadas ...*domainfinanceiro.CategoriaFinanceira) {
	t.Helper()

	if len(obtidos) != len(esperadas) {
		t.Fatalf("%d categorias, esperava %d", len(obtidos), len(esperadas))
	}
	for i, esperada := range esperadas {
		if obtidos[i] != esperada.ID {
			t.Errorf("posição %d diferente de %q", i, esperada.Nome)
		}
	}
}

func TestCategoriaFinanceiraCriarRaizESubcategoria(t *testing.T) {
	c := novoCenarioPlano(t)
	raiz := c.novaRaiz(t, "Despesas administrativas", "despesa", "despesa_fixa", 3, codigoDe("4.2"))
	sub := c.novaSub(t, raiz, "Aluguel", 0, codigoDe("4.2.01"))

	lida := c.obter(t, sub.ID)
	if lida.CategoriaPaiID == nil || *lida.CategoriaPaiID != raiz.ID || !lida.Lancavel() {
		t.Errorf("subcategoria lida = %+v", lida)
	}
	if lida.Natureza != domainfinanceiro.NaturezaDespesa || lida.LinhaDRE != domainfinanceiro.LinhaDespesaFixa {
		t.Errorf("classificação herdada = %s/%s", lida.Natureza, lida.LinhaDRE)
	}
	if lida.Codigo == nil || *lida.Codigo != "4.2.01" || lida.Versao != 1 {
		t.Errorf("código %v versão %d", lida.Codigo, lida.Versao)
	}

	if lidaRaiz := c.obter(t, raiz.ID); lidaRaiz.EhSubcategoria() || lidaRaiz.Ordem != 3 {
		t.Errorf("raiz lida = %+v", lidaRaiz)
	}

	if inexistente, err := c.categorias.Obter(c.ctx, uuid.New()); err != nil || inexistente != nil {
		t.Errorf("categoria inexistente = %v, %v", inexistente, err)
	}
}

func TestCategoriaFinanceiraSubcategoriaUsaAClassificacaoGravadaNoPai(t *testing.T) {
	c := novoCenarioPlano(t)
	raiz := c.novaRaiz(t, "Financeiro", "despesa", "resultado_financeiro", 0, nil)

	sub := c.subDeDominio(t, raiz, "Tarifas", 0, nil)
	sub.Natureza = domainfinanceiro.NaturezaReceita
	if err := c.categorias.Criar(c.ctx, sub); err != nil {
		t.Fatalf("Criar falhou: %v", err)
	}

	if lida := c.obter(t, sub.ID); lida.Natureza != domainfinanceiro.NaturezaDespesa || sub.Natureza != domainfinanceiro.NaturezaDespesa {
		t.Errorf("natureza gravada %s, em memória %s", lida.Natureza, sub.Natureza)
	}
}

func TestCategoriaFinanceiraSubcategoriaExigePaiRaizAtivo(t *testing.T) {
	c := novoCenarioPlano(t)
	raiz := c.novaRaiz(t, "Despesas", "despesa", "despesa_fixa", 0, nil)
	sub := c.novaSub(t, raiz, "Aluguel", 0, nil)
	inativa := c.novaRaiz(t, "Antigas", "despesa", "despesa_fixa", 0, nil)
	semFilhas := c.subDeDominio(t, inativa, "Ainda ativa", 0, nil)
	c.desativar(t, inativa)

	terceiroNivel := c.subDeDominio(t, raiz, "Terceiro nível", 0, nil)
	terceiroNivel.CategoriaPaiID = &sub.ID
	esperarErro(t, c.categorias.Criar(c.ctx, terceiroNivel), domain.ErrValidacao, "categoria pai deve ser uma categoria raiz ativa")
	esperarErro(t, c.categorias.Criar(c.ctx, semFilhas), domain.ErrValidacao, "categoria pai deve ser uma categoria raiz ativa")

	orfa := c.subDeDominio(t, raiz, "Órfã", 0, nil)
	paiInexistente := uuid.New()
	orfa.CategoriaPaiID = &paiInexistente
	esperarErro(t, c.categorias.Criar(c.ctx, orfa), domain.ErrValidacao, "categoria pai deve ser uma categoria raiz ativa")
}

func TestCategoriaFinanceiraNomesUnicosPorNivelECodigoUnico(t *testing.T) {
	c := novoCenarioPlano(t)
	despesas := c.novaRaiz(t, "Despesas", "despesa", "despesa_fixa", 0, codigoDe("4"))
	impostos := c.novaRaiz(t, "Impostos", "despesa", "deducao_venda", 0, nil)
	c.novaSub(t, despesas, "Outros", 0, nil)

	repetida, err := domainfinanceiro.NovaClassificacao("receita", "receita_operacional")
	if err != nil {
		t.Fatalf("classificação inválida: %v", err)
	}
	raizRepetida, err := domainfinanceiro.NovaCategoriaFinanceira(domainfinanceiro.DadosCategoriaFinanceira{Nome: "DESPESAS"}, &repetida, nil)
	if err != nil {
		t.Fatalf("raiz de domínio inválida: %v", err)
	}
	esperarErro(t, c.categorias.Criar(c.ctx, raizRepetida), domain.ErrConflito, "já existe uma categoria com este nome neste nível")
	esperarErro(t, c.categorias.Criar(c.ctx, c.subDeDominio(t, despesas, "outros", 0, nil)), domain.ErrConflito, "já existe uma categoria com este nome neste nível")
	esperarErro(t, c.categorias.Criar(c.ctx, c.subDeDominio(t, impostos, "ISS", 0, codigoDe("4"))), domain.ErrConflito, "já existe uma categoria com este código")

	c.novaSub(t, impostos, "Outros", 0, nil)
	c.novaSub(t, despesas, "Despesas", 0, nil)
}

func TestCategoriaFinanceiraListarFiltraEOrdena(t *testing.T) {
	c := novoCenarioPlano(t)
	despesas := c.novaRaiz(t, "Despesas", "despesa", "despesa_fixa", 2, codigoDe("4"))
	receitas := c.novaRaiz(t, "Receitas", "receita", "receita_operacional", 1, codigoDe("3"))
	aluguel := c.novaSub(t, despesas, "Aluguel", 1, codigoDe("4.01"))
	energia := c.novaSub(t, despesas, "energia elétrica", 1, codigoDe("4.02"))
	servicos := c.novaSub(t, receitas, "Serviços 100%", 0, codigoDe("3.01"))
	antiga := c.novaSub(t, despesas, "Antiga", 0, codigoDe("40.1"))
	c.desativar(t, antiga)

	despesa := domainfinanceiro.NaturezaDespesa
	fixa := domainfinanceiro.LinhaDespesaFixa
	ativa := true
	casos := map[string]struct {
		filtro    portsout.CategoriaFinanceiraFiltro
		esperadas []*domainfinanceiro.CategoriaFinanceira
	}{
		"todas":            {portsout.CategoriaFinanceiraFiltro{}, []*domainfinanceiro.CategoriaFinanceira{antiga, servicos, aluguel, energia, receitas, despesas}},
		"raízes":           {portsout.CategoriaFinanceiraFiltro{SomenteRaiz: true}, []*domainfinanceiro.CategoriaFinanceira{receitas, despesas}},
		"filhas do pai":    {portsout.CategoriaFinanceiraFiltro{Pai: &despesas.ID}, []*domainfinanceiro.CategoriaFinanceira{antiga, aluguel, energia}},
		"natureza":         {portsout.CategoriaFinanceiraFiltro{Natureza: &despesa, Ativa: &ativa}, []*domainfinanceiro.CategoriaFinanceira{aluguel, energia, despesas}},
		"linha":            {portsout.CategoriaFinanceiraFiltro{LinhaDRE: &fixa, SomenteRaiz: true}, []*domainfinanceiro.CategoriaFinanceira{despesas}},
		"busca no nome":    {portsout.CategoriaFinanceiraFiltro{Busca: " ENERGIA "}, []*domainfinanceiro.CategoriaFinanceira{energia}},
		"busca no código":  {portsout.CategoriaFinanceiraFiltro{Busca: "4.0"}, []*domainfinanceiro.CategoriaFinanceira{aluguel, energia}},
		"busca literal":    {portsout.CategoriaFinanceiraFiltro{Busca: "100%"}, []*domainfinanceiro.CategoriaFinanceira{servicos}},
		"curinga escapado": {portsout.CategoriaFinanceiraFiltro{Busca: "_"}, nil},
		"página":           {portsout.CategoriaFinanceiraFiltro{PaginacaoFiltro: domain.PaginacaoFiltro{Page: 2, Size: 4}}, []*domainfinanceiro.CategoriaFinanceira{receitas, despesas}},
	}

	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			categorias, err := c.categorias.Listar(c.ctx, caso.filtro)
			if err != nil {
				t.Fatalf("Listar falhou: %v", err)
			}
			conferirIDs(t, idsDasCategorias(categorias), caso.esperadas...)
		})
	}
}

func TestCategoriaFinanceiraListarPlanoMontaArvore(t *testing.T) {
	c := novoCenarioPlano(t)
	despesas := c.novaRaiz(t, "Despesas", "despesa", "despesa_fixa", 2, nil)
	receitas := c.novaRaiz(t, "Receitas", "receita", "receita_operacional", 1, nil)
	aluguel := c.novaSub(t, despesas, "Aluguel", 0, nil)
	antiga := c.novaSub(t, despesas, "Antiga", 0, nil)
	c.desativar(t, antiga)
	vazia := c.novaRaiz(t, "Vazia", "despesa", "fora_dre", 3, nil)
	c.desativar(t, vazia)

	todas, err := c.categorias.ListarPlano(c.ctx, false)
	if err != nil {
		t.Fatalf("ListarPlano falhou: %v", err)
	}
	plano := domainfinanceiro.MontarPlano(todas)
	if len(plano) != 3 || plano[0].Categoria.ID != receitas.ID || plano[1].Categoria.ID != despesas.ID || plano[2].Categoria.ID != vazia.ID {
		t.Fatalf("raízes do plano = %v", plano)
	}
	conferirIDs(t, idsDasCategorias(plano[1].Subcategorias), aluguel, antiga)

	ativas, err := c.categorias.ListarPlano(c.ctx, true)
	if err != nil {
		t.Fatalf("ListarPlano ativas falhou: %v", err)
	}
	planoAtivo := domainfinanceiro.MontarPlano(ativas)
	if len(planoAtivo) != 2 {
		t.Fatalf("plano ativo com %d raízes", len(planoAtivo))
	}
	conferirIDs(t, idsDasCategorias(planoAtivo[1].Subcategorias), aluguel)
}

func TestCategoriaFinanceiraAtualizarAvancaVersao(t *testing.T) {
	c := novoCenarioPlano(t)
	raiz := c.novaRaiz(t, "Despesas", "despesa", "despesa_fixa", 0, codigoDe("4"))
	c.novaRaiz(t, "Receitas", "receita", "receita_operacional", 0, codigoDe("3"))

	if err := raiz.Alterar(domainfinanceiro.DadosCategoriaFinanceira{Nome: "Despesas fixas", Codigo: codigoDe("4.1"), Ordem: 5}, time.Now().UTC()); err != nil {
		t.Fatalf("Alterar falhou: %v", err)
	}
	if err := c.categorias.Atualizar(c.ctx, raiz); err != nil {
		t.Fatalf("Atualizar falhou: %v", err)
	}
	lida := c.obter(t, raiz.ID)
	if lida.Nome != "Despesas fixas" || *lida.Codigo != "4.1" || lida.Ordem != 5 || lida.Versao != 2 || raiz.Versao != 2 {
		t.Errorf("categoria lida = %+v", lida)
	}

	antiga := *raiz
	antiga.Versao = 1
	esperarErro(t, c.categorias.Atualizar(c.ctx, &antiga), domain.ErrConflito, "alterada por outra pessoa")

	if err := raiz.Alterar(domainfinanceiro.DadosCategoriaFinanceira{Nome: "Despesas fixas", Codigo: codigoDe("3")}, time.Now().UTC()); err != nil {
		t.Fatalf("Alterar falhou: %v", err)
	}
	esperarErro(t, c.categorias.Atualizar(c.ctx, raiz), domain.ErrConflito, "já existe uma categoria com este código")

	sumida := *raiz
	sumida.ID = uuid.New()
	esperarErro(t, c.categorias.Atualizar(c.ctx, &sumida), domain.ErrNotFound, "categoria financeira não encontrada")
}

func TestCategoriaFinanceiraDesativarRaizSoSemFilhasAtivas(t *testing.T) {
	c := novoCenarioPlano(t)
	raiz := c.novaRaiz(t, "Despesas", "despesa", "despesa_fixa", 0, nil)
	sub := c.novaSub(t, raiz, "Aluguel", 0, nil)

	possui, err := c.categorias.PossuiSubcategoriasAtivas(c.ctx, raiz.ID)
	if err != nil || !possui {
		t.Fatalf("PossuiSubcategoriasAtivas = %v, %v", possui, err)
	}

	copia := *raiz
	copia.Ativa = false
	esperarErro(t, c.categorias.Desativar(c.ctx, &copia), domain.ErrConflito, "categoria possui subcategorias ativas")

	c.desativar(t, sub)
	if possui, err := c.categorias.PossuiSubcategoriasAtivas(c.ctx, raiz.ID); err != nil || possui {
		t.Fatalf("PossuiSubcategoriasAtivas depois = %v, %v", possui, err)
	}
	c.desativar(t, raiz)
	if lida := c.obter(t, raiz.ID); lida.Ativa || lida.Versao != 2 {
		t.Errorf("raiz lida = ativa %v versão %d", lida.Ativa, lida.Versao)
	}

	antiga := *raiz
	antiga.Versao = 1
	esperarErro(t, c.categorias.Desativar(c.ctx, &antiga), domain.ErrConflito, "alterada por outra pessoa")

	sumida := *raiz
	sumida.ID = uuid.New()
	esperarErro(t, c.categorias.Desativar(c.ctx, &sumida), domain.ErrNotFound, "categoria financeira não encontrada")
}

func TestCategoriaFinanceiraAtivarSubcategoriaExigePaiAtivo(t *testing.T) {
	c := novoCenarioPlano(t)
	raiz := c.novaRaiz(t, "Despesas", "despesa", "despesa_fixa", 0, nil)
	sub := c.novaSub(t, raiz, "Aluguel", 0, nil)
	c.desativar(t, sub)
	c.desativar(t, raiz)

	copia := *sub
	copia.Ativa = true
	esperarErro(t, c.categorias.Ativar(c.ctx, &copia), domain.ErrValidacao, "categoria pai está inativa")

	raiz.Ativa = true
	if err := c.categorias.Ativar(c.ctx, raiz); err != nil {
		t.Fatalf("Ativar raiz falhou: %v", err)
	}
	if err := c.categorias.Ativar(c.ctx, &copia); err != nil {
		t.Fatalf("Ativar subcategoria falhou: %v", err)
	}
	if lida := c.obter(t, sub.ID); !lida.Lancavel() || lida.Versao != 3 {
		t.Errorf("subcategoria lida = ativa %v versão %d", lida.Ativa, lida.Versao)
	}
}

func TestCategoriaFinanceiraCriarSubcategoriaEDesativarPaiAoMesmoTempo(t *testing.T) {
	c := novoCenarioPlano(t)

	for rodada := range 10 {
		raiz := c.novaRaiz(t, "Raiz "+string(rune('A'+rodada)), "despesa", "despesa_fixa", 0, nil)
		sub := c.subDeDominio(t, raiz, "Filha", 0, nil)

		var espera sync.WaitGroup
		espera.Add(2)
		go func() {
			defer espera.Done()
			_ = c.categorias.Criar(c.ctx, sub)
		}()
		go func() {
			defer espera.Done()
			copia := *raiz
			copia.Ativa = false
			_ = c.categorias.Desativar(c.ctx, &copia)
		}()
		espera.Wait()

		var invalidas int
		if err := c.banco.GetContext(c.ctx, &invalidas, `
			SELECT COUNT(*)
			FROM categoria_financeira filha
			JOIN categoria_financeira pai ON pai.id = filha.categoria_pai_id
			WHERE filha.ativa AND NOT pai.ativa
		`); err != nil {
			t.Fatalf("contagem falhou: %v", err)
		}
		if invalidas != 0 {
			t.Fatalf("rodada %d deixou %d subcategorias ativas sob raiz inativa", rodada, invalidas)
		}
	}
}

func TestCategoriaFinanceiraRemover(t *testing.T) {
	c := novoCenarioPlano(t)
	raiz := c.novaRaiz(t, "Despesas", "despesa", "despesa_fixa", 0, nil)
	sub := c.novaSub(t, raiz, "Aluguel", 0, nil)

	esperarErro(t, c.categorias.Remover(c.ctx, raiz.ID), domain.ErrValidacao, "registro em uso")

	if err := c.categorias.Remover(c.ctx, sub.ID); err != nil {
		t.Fatalf("Remover subcategoria falhou: %v", err)
	}
	if err := c.categorias.Remover(c.ctx, raiz.ID); err != nil {
		t.Fatalf("Remover raiz falhou: %v", err)
	}
	esperarErro(t, c.categorias.Remover(c.ctx, raiz.ID), domain.ErrNotFound, "categoria financeira não encontrada")
}
