package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/solicitacao"
	"github.com/google/uuid"
)

type cenarioBancoClassificacao struct {
	*cenarioBancoSolicitacoes
	fornecedorID uuid.UUID
	despesaID    uuid.UUID
	receitaID    uuid.UUID
	raizID       uuid.UUID
}

func cenarioClassificacao(t *testing.T) *cenarioBancoClassificacao {
	t.Helper()

	base := cenarioSolicitacoes(t)
	c := &cenarioBancoClassificacao{cenarioBancoSolicitacoes: base}
	c.fornecedorID = c.inserirFornecedor(t, "11.444.777/0001-61", "Cimentos do Vale")

	raizDespesa := c.inserirCategoria(t, "Despesas administrativas", "despesa", "despesa_fixa", nil)
	c.raizID = raizDespesa.ID
	c.despesaID = c.inserirCategoria(t, "Aluguel", "", "", raizDespesa).ID
	raizReceita := c.inserirCategoria(t, "Outras receitas", "receita", "receita_operacional", nil)
	c.receitaID = c.inserirCategoria(t, "Aluguel recebido", "", "", raizReceita).ID

	return c
}

func (c *cenarioBancoClassificacao) inserirFornecedor(t *testing.T, documento, nome string) uuid.UUID {
	t.Helper()

	valor, err := domaincliente.NovoDocumento(documento)
	if err != nil {
		t.Fatalf("documento de teste inválido: %v", err)
	}
	fornecedor, err := domainfornecedor.NovoFornecedor(domainfornecedor.PaisBrasil, valor, domainfornecedor.DadosFornecedor{
		Nome:        nome,
		MoedaPadrao: domainfornecedor.MoedaReal,
	})
	if err != nil {
		t.Fatalf("fornecedor de teste inválido: %v", err)
	}
	if _, err := postgres.NewFornecedorRepository(c.banco).Criar(context.Background(), fornecedor); err != nil {
		t.Fatalf("criação do fornecedor falhou: %v", err)
	}

	return fornecedor.ID
}

func (c *cenarioBancoClassificacao) inserirCategoria(t *testing.T, nome, natureza, linha string, pai *domainfinanceiro.CategoriaFinanceira) *domainfinanceiro.CategoriaFinanceira {
	t.Helper()

	var classificacao *domainfinanceiro.Classificacao
	if pai == nil {
		valor, err := domainfinanceiro.NovaClassificacao(natureza, linha)
		if err != nil {
			t.Fatalf("classificação de teste inválida: %v", err)
		}
		classificacao = &valor
	}

	categoria, err := domainfinanceiro.NovaCategoriaFinanceira(domainfinanceiro.DadosCategoriaFinanceira{Nome: nome}, classificacao, pai)
	if err != nil {
		t.Fatalf("categoria de teste inválida: %v", err)
	}
	if err := postgres.NewCategoriaFinanceiraRepository(c.banco).Criar(context.Background(), categoria); err != nil {
		t.Fatalf("criação da categoria %q falhou: %v", nome, err)
	}

	return categoria
}

func (c *cenarioBancoClassificacao) gravar(t *testing.T, solicitacao *domainsolicitacao.Solicitacao) error {
	t.Helper()

	historico := domainsolicitacao.NovoHistorico(
		solicitacao.ID, solicitacao.SolicitanteID, nil, solicitacao.Status, domainsolicitacao.EventoCriada, "solicitação criada",
	)
	_, err := c.solicitacoes.Criar(context.Background(), solicitacao, nil, historico)

	return err
}

func (c *cenarioBancoClassificacao) aprovada(t *testing.T) *domainsolicitacao.Solicitacao {
	t.Helper()

	solicitacao := c.novaSolicitacao(t)
	if err := c.gravar(t, solicitacao); err != nil {
		t.Fatalf("criação falhou: %v", err)
	}

	statusAnterior := solicitacao.Status
	if err := solicitacao.Aprovar(c.terceiroID, time.Now().UTC()); err != nil {
		t.Fatalf("aprovação falhou: %v", err)
	}
	historico := domainsolicitacao.NovoHistorico(
		solicitacao.ID, c.terceiroID, &statusAnterior, solicitacao.Status, domainsolicitacao.EventoAprovada, "solicitação aprovada",
	)
	atualizado, err := c.solicitacoes.AtualizarStatus(context.Background(), solicitacao, historico)
	if err != nil || !atualizado {
		t.Fatalf("aprovação no banco = %v, %v", atualizado, err)
	}

	return solicitacao
}

func (c *cenarioBancoClassificacao) classificacao(t *testing.T, competencia string) domainsolicitacao.Classificacao {
	t.Helper()

	valor, err := domainsolicitacao.NovaCompetencia(competencia)
	if err != nil {
		t.Fatalf("competência de teste inválida: %v", err)
	}

	return domainsolicitacao.Classificacao{
		FornecedorID:          &c.fornecedorID,
		CategoriaFinanceiraID: &c.despesaID,
		Competencia:           &valor,
	}
}

func (c *cenarioBancoClassificacao) obter(t *testing.T, id uuid.UUID) *domainsolicitacao.Solicitacao {
	t.Helper()

	solicitacao, err := c.solicitacoes.Obter(context.Background(), id)
	if err != nil || solicitacao == nil {
		t.Fatalf("obter = %v, %v", solicitacao, err)
	}

	return solicitacao
}

func TestSolicitacaoCriadaGravaClassificacaoOrigemEVersao(t *testing.T) {
	c := cenarioClassificacao(t)
	base := c.novaSolicitacao(t)

	solicitacao, err := domainsolicitacao.NovaSolicitacao(
		base.SolicitanteID, base.EmpresaID, base.Valor, base.Prazo, base.Observacao,
		domainsolicitacao.FormaPagamentoDebitoAutomatico, c.classificacao(t, "2026-09"), true, nil,
	)
	if err != nil {
		t.Fatalf("solicitação de teste inválida: %v", err)
	}
	if err := c.gravar(t, solicitacao); err != nil {
		t.Fatalf("criação falhou: %v", err)
	}

	gravada := c.obter(t, solicitacao.ID)
	if gravada.FornecedorID == nil || *gravada.FornecedorID != c.fornecedorID {
		t.Errorf("fornecedor = %v, esperado %s", gravada.FornecedorID, c.fornecedorID)
	}
	if gravada.CategoriaFinanceiraID == nil || *gravada.CategoriaFinanceiraID != c.despesaID {
		t.Errorf("categoria = %v, esperado %s", gravada.CategoriaFinanceiraID, c.despesaID)
	}
	if gravada.Competencia.String() != "2026-09" || !gravada.Estimado {
		t.Errorf("competência = %s, estimado = %v", gravada.Competencia, gravada.Estimado)
	}
	if gravada.FormaPagamento != domainsolicitacao.FormaPagamentoDebitoAutomatico {
		t.Errorf("forma = %q, esperado debito_automatico", gravada.FormaPagamento)
	}
	if gravada.Origem != domainsolicitacao.OrigemManual || gravada.OrigemID != nil || gravada.Versao != 1 {
		t.Errorf("origem = %q/%v, versão = %d", gravada.Origem, gravada.OrigemID, gravada.Versao)
	}

	historico, err := c.solicitacoes.ListarHistorico(context.Background(), solicitacao.ID)
	if err != nil || len(historico) != 1 || historico[0].Evento != domainsolicitacao.EventoCriada {
		t.Errorf("histórico = %+v, %v; esperado um evento criada", historico, err)
	}
}

func TestSolicitacaoSemClassificacaoGuardaCompetenciaDoVencimento(t *testing.T) {
	c := cenarioClassificacao(t)
	solicitacao := c.novaSolicitacao(t)
	if err := c.gravar(t, solicitacao); err != nil {
		t.Fatalf("criação falhou: %v", err)
	}

	gravada := c.obter(t, solicitacao.ID)
	if gravada.FornecedorID != nil || gravada.CategoriaFinanceiraID != nil {
		t.Error("solicitação sem classificação voltou classificada")
	}
	if gravada.Competencia != domainsolicitacao.CompetenciaDe(solicitacao.Prazo.Data()) {
		t.Errorf("competência = %s, esperado o mês do vencimento", gravada.Competencia)
	}
}

func TestSolicitacaoCriarRecusaClassificacaoEEmpresaInvalidas(t *testing.T) {
	c := cenarioClassificacao(t)
	ctx := context.Background()

	inativo := c.inserirFornecedor(t, "11.222.333/0001-81", "Fornecedor parado")
	if _, err := c.banco.ExecContext(ctx, `UPDATE fornecedor SET ativo = false WHERE id = $1`, inativo); err != nil {
		t.Fatalf("desativação falhou: %v", err)
	}
	empresaInativa := inserirEmpresa(t, c.banco, "11.444.777/0001-61", "Inativa")
	if _, err := c.banco.ExecContext(ctx, `UPDATE empresa SET ativa = false WHERE id = $1`, empresaInativa); err != nil {
		t.Fatalf("desativação falhou: %v", err)
	}
	inexistente := uuid.New()

	casos := map[string]struct {
		empresaID uuid.UUID
		ajustar   func(*domainsolicitacao.Solicitacao)
		mensagem  string
	}{
		"fornecedor inativo":     {c.empresaID, func(s *domainsolicitacao.Solicitacao) { s.FornecedorID = &inativo }, domainsolicitacao.ErroFornecedorInativo().Error()},
		"fornecedor inexistente": {c.empresaID, func(s *domainsolicitacao.Solicitacao) { s.FornecedorID = &inexistente }, domainsolicitacao.ErroFornecedorNaoEncontrado().Error()},
		"categoria raiz":         {c.empresaID, func(s *domainsolicitacao.Solicitacao) { s.CategoriaFinanceiraID = &c.raizID }, domainsolicitacao.ErroCategoriaFinanceiraNaoLancavel().Error()},
		"categoria de receita":   {c.empresaID, func(s *domainsolicitacao.Solicitacao) { s.CategoriaFinanceiraID = &c.receitaID }, domainsolicitacao.ErroCategoriaFinanceiraNaoEhDespesa().Error()},
		"categoria inexistente":  {c.empresaID, func(s *domainsolicitacao.Solicitacao) { s.CategoriaFinanceiraID = &inexistente }, domainsolicitacao.ErroCategoriaFinanceiraNaoEncontrada().Error()},
		"empresa inativa":        {empresaInativa, func(*domainsolicitacao.Solicitacao) {}, domainsolicitacao.ErroEmpresaInativa().Error()},
	}

	for nome, caso := range casos {
		solicitacao := c.novaSolicitacaoNaEmpresa(t, caso.empresaID)
		caso.ajustar(solicitacao)

		err := c.gravar(t, solicitacao)
		if !errors.Is(err, domain.ErrValidacao) || err.Error() != caso.mensagem {
			t.Errorf("%s = %v, esperado %q", nome, err, caso.mensagem)
		}
	}
}

func TestClassificarAvancaVersaoEGravaEvento(t *testing.T) {
	c := cenarioClassificacao(t)
	ctx := context.Background()
	solicitacao := c.aprovada(t)

	if lida := c.obter(t, solicitacao.ID); lida.Versao != 2 {
		t.Fatalf("versão depois da aprovação = %d, esperado 2", lida.Versao)
	}

	versaoLida := solicitacao.Versao
	if err := solicitacao.Classificar(c.classificacao(t, "2026-07"), time.Now().UTC()); err != nil {
		t.Fatalf("classificação no domínio falhou: %v", err)
	}
	historico := domainsolicitacao.NovoHistorico(
		solicitacao.ID, c.terceiroID, &solicitacao.Status, solicitacao.Status, domainsolicitacao.EventoClassificada, "solicitação classificada",
	)
	if err := c.solicitacoes.Classificar(ctx, solicitacao, versaoLida, historico); err != nil {
		t.Fatalf("classificação falhou: %v", err)
	}

	gravada := c.obter(t, solicitacao.ID)
	if gravada.Versao != 3 || gravada.Competencia.String() != "2026-07" || *gravada.CategoriaFinanceiraID != c.despesaID {
		t.Errorf("gravada = versão %d, competência %s, categoria %v", gravada.Versao, gravada.Competencia, gravada.CategoriaFinanceiraID)
	}

	eventos, err := c.solicitacoes.ListarHistorico(ctx, solicitacao.ID)
	if err != nil || len(eventos) != 3 || eventos[0].Evento != domainsolicitacao.EventoClassificada {
		t.Fatalf("histórico = %+v, %v; esperado classificada no topo", eventos, err)
	}
	if eventos[0].DeStatus == nil || *eventos[0].DeStatus != domainsolicitacao.StatusAprovado || eventos[0].ParaStatus != domainsolicitacao.StatusAprovado {
		t.Errorf("classificação mudou o status no histórico: %+v", eventos[0])
	}
}

func TestClassificarComVersaoVelhaOuInexistenteNaoGrava(t *testing.T) {
	c := cenarioClassificacao(t)
	ctx := context.Background()
	solicitacao := c.aprovada(t)

	if err := solicitacao.Classificar(c.classificacao(t, "2026-07"), time.Now().UTC()); err != nil {
		t.Fatalf("classificação no domínio falhou: %v", err)
	}
	historico := domainsolicitacao.NovoHistorico(
		solicitacao.ID, c.terceiroID, &solicitacao.Status, solicitacao.Status, domainsolicitacao.EventoClassificada, "solicitação classificada",
	)

	if err := c.solicitacoes.Classificar(ctx, solicitacao, 1, historico); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("versão velha = %v, esperado conflito", err)
	}

	outra := *solicitacao
	outra.ID = uuid.New()
	if err := c.solicitacoes.Classificar(ctx, &outra, 2, historico); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("inexistente = %v, esperado não encontrado", err)
	}

	if gravada := c.obter(t, solicitacao.ID); gravada.CategoriaFinanceiraID != nil || gravada.Versao != 2 {
		t.Errorf("classificação recusada gravou: categoria %v, versão %d", gravada.CategoriaFinanceiraID, gravada.Versao)
	}
}

func TestClassificarRecusaCategoriaDesativadaNaTransacao(t *testing.T) {
	c := cenarioClassificacao(t)
	ctx := context.Background()
	solicitacao := c.aprovada(t)

	if _, err := c.banco.ExecContext(ctx, `UPDATE categoria_financeira SET ativa = false WHERE id = $1`, c.despesaID); err != nil {
		t.Fatalf("desativação falhou: %v", err)
	}

	versaoLida := solicitacao.Versao
	if err := solicitacao.Classificar(c.classificacao(t, "2026-07"), time.Now().UTC()); err != nil {
		t.Fatalf("classificação no domínio falhou: %v", err)
	}
	historico := domainsolicitacao.NovoHistorico(
		solicitacao.ID, c.terceiroID, &solicitacao.Status, solicitacao.Status, domainsolicitacao.EventoClassificada, "solicitação classificada",
	)

	if err := c.solicitacoes.Classificar(ctx, solicitacao, versaoLida, historico); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("categoria inativa = %v, esperado erro de validação", err)
	}
}

func TestListarSolicitacoesFiltraPorClassificacaoOrigemEVencimento(t *testing.T) {
	c := cenarioClassificacao(t)
	ctx := context.Background()
	hoje := time.Now().UTC().Truncate(24 * time.Hour)

	classificada := c.novaSolicitacao(t)
	classificada.FornecedorID = &c.fornecedorID
	classificada.CategoriaFinanceiraID = &c.despesaID
	classificada.Competencia = domainsolicitacao.CompetenciaDe(time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC))
	vencida := c.novaSolicitacao(t)
	vencida.Prazo = domainsolicitacao.PrazoPagamentoDe(hoje.AddDate(0, 0, -3))
	paga := c.novaSolicitacao(t)
	paga.Prazo = domainsolicitacao.PrazoPagamentoDe(hoje.AddDate(0, 0, -3))
	paga.Status = domainsolicitacao.StatusPago

	for _, solicitacao := range []*domainsolicitacao.Solicitacao{classificada, vencida, paga} {
		if err := c.gravar(t, solicitacao); err != nil {
			t.Fatalf("criação falhou: %v", err)
		}
	}

	competencia := domainsolicitacao.CompetenciaDe(time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC))
	ate := hoje.AddDate(0, 0, -1)
	casos := map[string]struct {
		filtro    portsout.SolicitacaoFiltro
		esperadas []uuid.UUID
	}{
		"fornecedor":  {portsout.SolicitacaoFiltro{Fornecedor: &c.fornecedorID}, []uuid.UUID{classificada.ID}},
		"categoria":   {portsout.SolicitacaoFiltro{Categoria: &c.despesaID}, []uuid.UUID{classificada.ID}},
		"competência": {portsout.SolicitacaoFiltro{Competencia: &competencia}, []uuid.UUID{classificada.ID}},
		"origem":      {portsout.SolicitacaoFiltro{Origem: domainsolicitacao.OrigemRecorrencia}, nil},
		"vence até":   {portsout.SolicitacaoFiltro{VenceAte: &ate}, []uuid.UUID{vencida.ID, paga.ID}},
		"vence de":    {portsout.SolicitacaoFiltro{VenceDe: &hoje}, []uuid.UUID{classificada.ID}},
		"vencidas":    {portsout.SolicitacaoFiltro{VencidasEm: &hoje}, []uuid.UUID{vencida.ID}},
	}

	for nome, caso := range casos {
		caso.filtro.PaginacaoFiltro = todasAsPaginas
		itens, err := c.solicitacoes.Listar(ctx, caso.filtro)
		if err != nil {
			t.Fatalf("%s: listagem falhou: %v", nome, err)
		}
		if !mesmosIDs(itens, caso.esperadas) {
			t.Errorf("%s = %d itens, esperado %v", nome, len(itens), caso.esperadas)
		}
	}
}

func mesmosIDs(itens []*domainsolicitacao.Solicitacao, esperadas []uuid.UUID) bool {
	if len(itens) != len(esperadas) {
		return false
	}

	restantes := map[uuid.UUID]bool{}
	for _, id := range esperadas {
		restantes[id] = true
	}
	for _, item := range itens {
		if !restantes[item.ID] {
			return false
		}
		delete(restantes, item.ID)
	}

	return true
}
