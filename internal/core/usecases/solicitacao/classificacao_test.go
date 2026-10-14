package solicitacao_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
	"github.com/google/uuid"
)

func (c *cenario) novoFornecedor(ativo bool) uuid.UUID {
	id := uuid.New()
	c.fornecedoresRepo.itens[id] = &domainfornecedor.Fornecedor{ID: id, Nome: "Fornecedor " + id.String()[:8], Ativo: ativo, Versao: 1}

	return id
}

func (c *cenario) novaCategoria(natureza domainfinanceiro.Natureza, subcategoria, ativa bool) uuid.UUID {
	id := uuid.New()
	categoria := &domainfinanceiro.CategoriaFinanceira{ID: id, Nome: "Categoria " + id.String()[:8], Natureza: natureza, Ativa: ativa, Versao: 1}
	if subcategoria {
		paiID := uuid.New()
		categoria.CategoriaPaiID = &paiID
	}
	c.categoriasRepo.itens[id] = categoria

	return id
}

func (c *cenario) aprovarSolicitacao(t *testing.T, id uuid.UUID) {
	t.Helper()

	aprovador := c.novoUsuario("Aprovador "+uuid.NewString()[:8], false, false)
	c.designar(t, aprovador)
	if err := c.solicitacao.Aprovar(c.ctx, id, aprovador); err != nil {
		t.Fatalf("aprovação falhou: %v", err)
	}
}

func (c *cenario) entradaCriacao(solicitante uuid.UUID) portsin.CriarSolicitacaoInput {
	return portsin.CriarSolicitacaoInput{
		SolicitanteID:  solicitante,
		EmpresaID:      c.empresaID,
		ValorCentavos:  10000,
		PrazoPagamento: time.Date(2099, time.March, 10, 0, 0, 0, 0, time.UTC),
		Observacao:     "aluguel do galpão",
		FormaPagamento: "transferencia",
	}
}

func TestCriarSolicitacaoComClassificacaoGuardaOsCampos(t *testing.T) {
	c := novoCenario(t)
	solicitante := c.novoUsuario("Ana", false, false)
	fornecedorID := c.novoFornecedor(true)
	categoriaID := c.novaCategoria(domainfinanceiro.NaturezaDespesa, true, true)

	entrada := c.entradaCriacao(solicitante)
	entrada.FornecedorID = &fornecedorID
	entrada.CategoriaFinanceiraID = &categoriaID
	entrada.Competencia = "2099-02"
	entrada.Estimado = true

	id, err := c.solicitacao.Criar(c.ctx, entrada)
	if err != nil {
		t.Fatalf("criação falhou: %v", err)
	}

	salva, err := c.solicitacao.Obter(c.ctx, id, solicitante)
	if err != nil {
		t.Fatalf("consulta falhou: %v", err)
	}
	if *salva.FornecedorID != fornecedorID || *salva.CategoriaFinanceiraID != categoriaID {
		t.Error("classificação informada não foi guardada")
	}
	if salva.Competencia.String() != "2099-02" || !salva.Estimado || salva.Versao != 1 {
		t.Errorf("competência = %s, estimado = %v, versão = %d", salva.Competencia, salva.Estimado, salva.Versao)
	}
	if salva.FormaPagamento != domainsolicitacao.FormaPagamentoTransferencia {
		t.Errorf("forma = %q, esperado transferencia", salva.FormaPagamento)
	}

	historico, err := c.solicitacao.ListarHistorico(c.ctx, id, solicitante)
	if err != nil || len(historico) != 1 || historico[0].Evento != domainsolicitacao.EventoCriada {
		t.Errorf("histórico = %+v, %v; esperado evento criada", historico, err)
	}
}

func TestCriarSolicitacaoSemCompetenciaUsaOMesDoVencimento(t *testing.T) {
	c := novoCenario(t)
	solicitante := c.novoUsuario("Ana", false, false)

	id, err := c.solicitacao.Criar(c.ctx, c.entradaCriacao(solicitante))
	if err != nil {
		t.Fatalf("criação falhou: %v", err)
	}

	salva, err := c.solicitacao.Obter(c.ctx, id, solicitante)
	if err != nil {
		t.Fatalf("consulta falhou: %v", err)
	}
	if salva.Competencia.String() != "2099-03" || salva.FornecedorID != nil || salva.CategoriaFinanceiraID != nil {
		t.Errorf("competência = %s, fornecedor = %v, categoria = %v", salva.Competencia, salva.FornecedorID, salva.CategoriaFinanceiraID)
	}
}

func TestCriarSolicitacaoRecusaClassificacaoInvalida(t *testing.T) {
	c := novoCenario(t)
	solicitante := c.novoUsuario("Ana", false, false)
	inexistente := uuid.New()
	fornecedorInativo := c.novoFornecedor(false)
	raiz := c.novaCategoria(domainfinanceiro.NaturezaDespesa, false, true)
	subInativa := c.novaCategoria(domainfinanceiro.NaturezaDespesa, true, false)
	receita := c.novaCategoria(domainfinanceiro.NaturezaReceita, true, true)

	casos := map[string]struct {
		ajustar  func(*portsin.CriarSolicitacaoInput)
		mensagem string
	}{
		"fornecedor inexistente": {func(e *portsin.CriarSolicitacaoInput) { e.FornecedorID = &inexistente }, "fornecedor não encontrado"},
		"fornecedor inativo":     {func(e *portsin.CriarSolicitacaoInput) { e.FornecedorID = &fornecedorInativo }, "fornecedor inativo não recebe conta a pagar"},
		"categoria inexistente":  {func(e *portsin.CriarSolicitacaoInput) { e.CategoriaFinanceiraID = &inexistente }, "categoria financeira não encontrada"},
		"categoria raiz":         {func(e *portsin.CriarSolicitacaoInput) { e.CategoriaFinanceiraID = &raiz }, "categoria financeira deve ser uma subcategoria ativa"},
		"subcategoria inativa":   {func(e *portsin.CriarSolicitacaoInput) { e.CategoriaFinanceiraID = &subInativa }, "categoria financeira deve ser uma subcategoria ativa"},
		"categoria de receita":   {func(e *portsin.CriarSolicitacaoInput) { e.CategoriaFinanceiraID = &receita }, "categoria financeira deve ser de despesa"},
		"competência inválida":   {func(e *portsin.CriarSolicitacaoInput) { e.Competencia = "03/2099" }, "competência deve estar no formato AAAA-MM"},
	}

	for nome, caso := range casos {
		entrada := c.entradaCriacao(solicitante)
		caso.ajustar(&entrada)

		_, err := c.solicitacao.Criar(c.ctx, entrada)
		if err == nil || err.Error() != domain.ErroValidacao(caso.mensagem).Error() {
			t.Errorf("%s = %v, esperado %q", nome, err, caso.mensagem)
		}
	}
}

func TestClassificarSolicitacaoAprovadaPeloFinanceiro(t *testing.T) {
	c := novoCenario(t)
	solicitante := c.novoUsuario("Ana", false, false)
	financeiro := c.novoUsuario("Fábio", false, true)
	fornecedorID := c.novoFornecedor(true)
	categoriaID := c.novaCategoria(domainfinanceiro.NaturezaDespesa, true, true)

	id := c.criarSolicitacao(t, solicitante, 10000)
	c.aprovarSolicitacao(t, id)

	err := c.solicitacao.Classificar(c.ctx, portsin.ClassificarSolicitacaoInput{
		SolicitacaoID:         id,
		UsuarioID:             financeiro,
		FornecedorID:          &fornecedorID,
		CategoriaFinanceiraID: &categoriaID,
		Competencia:           "2026-07",
		Versao:                2,
	})
	if err != nil {
		t.Fatalf("classificação falhou: %v", err)
	}

	salva, err := c.solicitacao.Obter(c.ctx, id, financeiro)
	if err != nil {
		t.Fatalf("consulta falhou: %v", err)
	}
	if salva.Versao != 3 || salva.Competencia.String() != "2026-07" || *salva.CategoriaFinanceiraID != categoriaID {
		t.Errorf("salva = versão %d, competência %s, categoria %v", salva.Versao, salva.Competencia, salva.CategoriaFinanceiraID)
	}

	historico, err := c.solicitacao.ListarHistorico(c.ctx, id, financeiro)
	if err != nil {
		t.Fatalf("histórico falhou: %v", err)
	}
	topo := historico[0]
	if topo.Evento != domainsolicitacao.EventoClassificada || topo.ParaStatus != domainsolicitacao.StatusAprovado ||
		topo.DeStatus == nil || *topo.DeStatus != domainsolicitacao.StatusAprovado || topo.UsuarioID != financeiro {
		t.Errorf("último registro = %+v, esperado classificação sem troca de status", topo)
	}
}

func TestClassificarSolicitacaoRecusaPerfilVersaoStatusECampos(t *testing.T) {
	c := novoCenario(t)
	solicitante := c.novoUsuario("Ana", false, false)
	financeiro := c.novoUsuario("Fábio", false, true)
	categoriaID := c.novaCategoria(domainfinanceiro.NaturezaDespesa, true, true)
	receita := c.novaCategoria(domainfinanceiro.NaturezaReceita, true, true)

	aprovada := c.criarSolicitacao(t, solicitante, 10000)
	c.aprovarSolicitacao(t, aprovada)
	pendente := c.criarSolicitacao(t, solicitante, 10000)

	outraEmpresa := c.novaEmpresa(true)
	financeiroDeOutra := c.novoUsuario("Gil", false, true)
	c.restringirAcesso(financeiroDeOutra, outraEmpresa)

	valida := func() portsin.ClassificarSolicitacaoInput {
		return portsin.ClassificarSolicitacaoInput{
			SolicitacaoID:         aprovada,
			UsuarioID:             financeiro,
			CategoriaFinanceiraID: &categoriaID,
			Competencia:           "2026-07",
			Versao:                2,
		}
	}

	casos := map[string]struct {
		ajustar  func(*portsin.ClassificarSolicitacaoInput)
		esperado error
	}{
		"sem perfil financeiro":  {func(e *portsin.ClassificarSolicitacaoInput) { e.UsuarioID = solicitante }, domain.ErrPermissao},
		"empresa fora do acesso": {func(e *portsin.ClassificarSolicitacaoInput) { e.UsuarioID = financeiroDeOutra }, domain.ErrPermissao},
		"inexistente":            {func(e *portsin.ClassificarSolicitacaoInput) { e.SolicitacaoID = uuid.New() }, domain.ErrNotFound},
		"versão ausente":         {func(e *portsin.ClassificarSolicitacaoInput) { e.Versao = 0 }, domain.ErrValidacao},
		"versão divergente":      {func(e *portsin.ClassificarSolicitacaoInput) { e.Versao = 1 }, domain.ErrConflito},
		"pendente de aprovação":  {func(e *portsin.ClassificarSolicitacaoInput) { e.SolicitacaoID = pendente; e.Versao = 1 }, domain.ErrConflito},
		"sem categoria":          {func(e *portsin.ClassificarSolicitacaoInput) { e.CategoriaFinanceiraID = nil }, domain.ErrValidacao},
		"sem competência":        {func(e *portsin.ClassificarSolicitacaoInput) { e.Competencia = "" }, domain.ErrValidacao},
		"competência inválida":   {func(e *portsin.ClassificarSolicitacaoInput) { e.Competencia = "2026-13" }, domain.ErrValidacao},
		"categoria de receita":   {func(e *portsin.ClassificarSolicitacaoInput) { e.CategoriaFinanceiraID = &receita }, domain.ErrValidacao},
	}

	for nome, caso := range casos {
		entrada := valida()
		caso.ajustar(&entrada)

		if err := c.solicitacao.Classificar(c.ctx, entrada); !errors.Is(err, caso.esperado) {
			t.Errorf("%s = %v, esperado %v", nome, err, caso.esperado)
		}
	}

	salva, err := c.solicitacao.Obter(c.ctx, aprovada, financeiro)
	if err != nil {
		t.Fatalf("consulta falhou: %v", err)
	}
	if salva.Versao != 2 || salva.CategoriaFinanceiraID != nil {
		t.Errorf("tentativas recusadas alteraram a solicitação: versão %d, categoria %v", salva.Versao, salva.CategoriaFinanceiraID)
	}
}

func TestListarSolicitacoesComFiltrosDeContasAPagar(t *testing.T) {
	c := novoCenario(t)
	solicitante := c.novoUsuario("Ana", false, false)
	categoriaID := c.novaCategoria(domainfinanceiro.NaturezaDespesa, true, true)

	classificada := c.entradaCriacao(solicitante)
	classificada.CategoriaFinanceiraID = &categoriaID
	classificadaID, err := c.solicitacao.Criar(c.ctx, classificada)
	if err != nil {
		t.Fatalf("criação falhou: %v", err)
	}
	semCategoriaID := c.criarSolicitacao(t, solicitante, 10000)

	vencida := c.solicitacoesRepo.itens[semCategoriaID]
	vencida.Prazo = domainsolicitacao.PrazoPagamentoDe(time.Now().UTC().AddDate(0, 0, -2))

	listar := func(ajustar func(*portsin.ListarSolicitacoesInput)) ([]*domainsolicitacao.Solicitacao, error) {
		entrada := portsin.ListarSolicitacoesInput{UsuarioID: solicitante, Filtro: domain.PaginacaoFiltro{Page: 1, Size: 50}}
		ajustar(&entrada)
		return c.solicitacao.Listar(c.ctx, entrada)
	}

	porCategoria, err := listar(func(e *portsin.ListarSolicitacoesInput) { e.Categoria = &categoriaID })
	if err != nil || len(porCategoria) != 1 || porCategoria[0].ID != classificadaID {
		t.Errorf("filtro por categoria = %d itens, %v", len(porCategoria), err)
	}

	porCompetencia, err := listar(func(e *portsin.ListarSolicitacoesInput) { e.Competencia = "2099-03" })
	if err != nil || len(porCompetencia) != 1 || porCompetencia[0].ID != classificadaID {
		t.Errorf("filtro por competência = %d itens, %v", len(porCompetencia), err)
	}

	vencidas, err := listar(func(e *portsin.ListarSolicitacoesInput) { e.Vencidas = true })
	if err != nil || len(vencidas) != 1 || vencidas[0].ID != semCategoriaID {
		t.Errorf("filtro de vencidas = %d itens, %v", len(vencidas), err)
	}

	manuais, err := listar(func(e *portsin.ListarSolicitacoesInput) { e.Origem = "manual" })
	if err != nil || len(manuais) != 2 {
		t.Errorf("filtro por origem = %d itens, %v", len(manuais), err)
	}

	de := time.Date(2099, time.March, 11, 0, 0, 0, 0, time.UTC)
	ate := time.Date(2099, time.March, 1, 0, 0, 0, 0, time.UTC)
	invalidos := map[string]func(*portsin.ListarSolicitacoesInput){
		"competência":         func(e *portsin.ListarSolicitacoesInput) { e.Competencia = "março" },
		"origem":              func(e *portsin.ListarSolicitacoesInput) { e.Origem = "pedido" },
		"intervalo invertido": func(e *portsin.ListarSolicitacoesInput) { e.VenceDe = &de; e.VenceAte = &ate },
	}
	for nome, ajustar := range invalidos {
		if _, err := listar(ajustar); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("%s = %v, esperado erro de validação", nome, err)
		}
	}
}
