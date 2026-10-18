package postgres_test

import (
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portscomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/comissao"
	"github.com/google/uuid"
)

type cenarioBancoFechamentos struct {
	*cenarioBancoComissoes
	fornecedorID uuid.UUID
	despesaID    uuid.UUID
	anterior     domaincomissao.Competencia
}

func novoCenarioFechamentos(t *testing.T) *cenarioBancoFechamentos {
	t.Helper()

	c := &cenarioBancoFechamentos{cenarioBancoComissoes: novoCenarioComissoes(t)}
	c.fornecedorID = c.fornecedorNoBanco(t)
	c.despesaID = c.despesaNoBanco(t)
	atual := domaincomissao.CompetenciaDe(domaincomissao.Hoje(time.Now().UTC()))
	c.anterior = domaincomissao.CompetenciaDe(atual.Inicio().AddDate(0, -1, 0))

	return c
}

func (c *cenarioBancoFechamentos) fornecedorNoBanco(t *testing.T) uuid.UUID {
	t.Helper()

	documento, err := domaincliente.NovoDocumento("45.723.174/0001-10")
	if err != nil {
		t.Fatalf("documento de teste inválido: %v", err)
	}
	fornecedor, err := domainfornecedor.NovoFornecedor(domainfornecedor.PaisBrasil, documento, domainfornecedor.DadosFornecedor{
		Nome: "Carlos Representações", MoedaPadrao: domainfornecedor.MoedaReal,
	})
	if err != nil {
		t.Fatalf("fornecedor de teste inválido: %v", err)
	}
	if _, err := postgres.NewFornecedorRepository(c.banco).Criar(c.ctx, fornecedor); err != nil {
		t.Fatalf("criação do fornecedor falhou: %v", err)
	}

	return fornecedor.ID
}

func (c *cenarioBancoFechamentos) despesaNoBanco(t *testing.T) uuid.UUID {
	t.Helper()

	categorias := postgres.NewCategoriaFinanceiraRepository(c.banco)
	classificacao, err := domainfinanceiro.NovaClassificacao("despesa", "despesa_variavel")
	if err != nil {
		t.Fatalf("classificação de teste inválida: %v", err)
	}
	raiz, err := domainfinanceiro.NovaCategoriaFinanceira(domainfinanceiro.DadosCategoriaFinanceira{Nome: "Despesas de venda"}, &classificacao, nil)
	if err != nil {
		t.Fatalf("raiz de teste inválida: %v", err)
	}
	if err := categorias.Criar(c.ctx, raiz); err != nil {
		t.Fatalf("criação da raiz falhou: %v", err)
	}
	sub, err := domainfinanceiro.NovaCategoriaFinanceira(domainfinanceiro.DadosCategoriaFinanceira{Nome: "Comissões"}, nil, raiz)
	if err != nil {
		t.Fatalf("subcategoria de teste inválida: %v", err)
	}
	if err := categorias.Criar(c.ctx, sub); err != nil {
		t.Fatalf("criação da subcategoria falhou: %v", err)
	}

	return sub.ID
}

func (c *cenarioBancoFechamentos) comissaoNaCompetencia(t *testing.T, competencia domaincomissao.Competencia, quantidade int) *domaincomissao.Comissao {
	t.Helper()

	pedido := c.pedidoConfirmado(t, c.carlos)
	c.saida(t, pedido, quantidade)
	entregas := c.paraApurar(t, portscomissao.EntregaFiltro{})
	apuracao, err := domaincomissao.NovaApuracao(*entregas[0], c.carlos, c.usuarioID, time.Now().UTC())
	if err != nil {
		t.Fatalf("NovaApuracao: %v", err)
	}
	apuracao.Comissao.Competencia = competencia
	if criada, err := c.comissoes.GravarApuracao(c.ctx, apuracao); err != nil || !criada {
		t.Fatalf("GravarApuracao = %v, %v", criada, err)
	}

	return apuracao.Comissao
}

func (c *cenarioBancoFechamentos) ajusteNaCompetencia(t *testing.T, competencia domaincomissao.Competencia, valor int64) *domaincomissao.Ajuste {
	t.Helper()

	ajuste := c.novoAjuste(t, domaincomissao.DadosAjuste{
		EmpresaID: c.mia, RepresentanteID: c.carlos.ID, Competencia: competencia.String(), ValorCentavos: valor, Motivo: "devolução",
	})
	if err := c.comissoes.CriarAjuste(c.ctx, ajuste); err != nil {
		t.Fatalf("CriarAjuste: %v", err)
	}

	return ajuste
}

func (c *cenarioBancoFechamentos) gravacao(t *testing.T, saldo domaincomissao.SaldoAberto) *portscomissao.GravacaoFechamento {
	t.Helper()

	agora := time.Now().UTC()
	fechamento, err := domaincomissao.NovoFechamento(c.mia, c.anterior, saldo, c.despesaID, c.usuarioID, agora)
	if err != nil {
		t.Fatalf("NovoFechamento: %v", err)
	}
	competencia := domainsolicitacao.CompetenciaDe(c.anterior.Inicio())
	solicitacao, historico, err := domainsolicitacao.NovaSolicitacaoDeOrigem(domainsolicitacao.DadosSolicitacaoDeOrigem{
		EmpresaID: c.mia, SolicitanteID: c.usuarioID, FornecedorID: &c.fornecedorID, CategoriaFinanceiraID: c.despesaID,
		Origem: domainsolicitacao.OrigemComissao, OrigemID: fechamento.ID, ValorCentavos: fechamento.TotalCentavos,
		Vencimento: fechamento.Vencimento, Competencia: &competencia, Observacao: "comissão de Carlos",
	}, agora)
	if err != nil {
		t.Fatalf("NovaSolicitacaoDeOrigem: %v", err)
	}
	fechamento.SolicitacaoID = solicitacao.ID

	return &portscomissao.GravacaoFechamento{Fechamento: fechamento, Solicitacao: solicitacao, Historico: historico}
}

func (c *cenarioBancoFechamentos) saldos(t *testing.T) []domaincomissao.SaldoAberto {
	t.Helper()

	saldos, err := c.comissoes.SaldosAbertos(c.ctx, portscomissao.SaldoFiltro{EmpresaID: c.mia, Ate: c.anterior})
	if err != nil {
		t.Fatalf("SaldosAbertos: %v", err)
	}

	return saldos
}

func TestComissaoRepositoryFechaOSaldoAbertoComAContaAPagar(t *testing.T) {
	c := novoCenarioFechamentos(t)
	retrasado := domaincomissao.CompetenciaDe(c.anterior.Inicio().AddDate(0, -1, 0))
	atual := c.anterior.Seguinte()
	antiga := c.comissaoNaCompetencia(t, retrasado, 4)
	recente := c.comissaoNaCompetencia(t, c.anterior, 2)
	futura := c.comissaoNaCompetencia(t, atual, 1)
	ajuste := c.ajusteNaCompetencia(t, c.anterior, -100)

	saldos := c.saldos(t)
	esperado := antiga.ValorCentavos + recente.ValorCentavos
	if len(saldos) != 1 || saldos[0].RepresentanteID != c.carlos.ID || saldos[0].ComissoesCentavos != esperado || saldos[0].AjustesCentavos != -100 {
		t.Fatalf("saldos = %+v, esperado comissões %d", saldos, esperado)
	}

	gravacao := c.gravacao(t, saldos[0])
	if err := c.comissoes.Fechar(c.ctx, gravacao); err != nil {
		t.Fatalf("Fechar: %v", err)
	}
	if restantes := c.saldos(t); len(restantes) != 0 {
		t.Fatalf("nada deveria ficar aberto até a competência: %+v", restantes)
	}

	fechamento, err := c.comissoes.ObterFechamento(c.ctx, gravacao.Fechamento.ID)
	if err != nil || fechamento == nil || fechamento.TotalCentavos != esperado-100 || fechamento.StatusContaAPagar != "pendente_aprovacao" ||
		fechamento.SolicitacaoID != gravacao.Solicitacao.ID || !fechamento.Vencimento.Equal(domaincomissao.VencimentoDoFechamento(c.anterior)) {
		t.Fatalf("fechamento lido = %+v, %v", fechamento, err)
	}

	solicitacao, err := postgres.NewSolicitacaoRepository(c.banco).Obter(c.ctx, gravacao.Solicitacao.ID)
	if err != nil || solicitacao == nil || solicitacao.Origem != domainsolicitacao.OrigemComissao || *solicitacao.OrigemID != fechamento.ID ||
		*solicitacao.FornecedorID != c.fornecedorID || solicitacao.Valor.Centavos() != esperado-100 {
		t.Fatalf("conta a pagar = %+v, %v", solicitacao, err)
	}

	comissoes, ajustes, err := c.comissoes.ItensDoFechamento(c.ctx, fechamento.ID)
	if err != nil || len(comissoes) != 2 || len(ajustes) != 1 || ajustes[0].ID != ajuste.ID || *comissoes[0].FechamentoID != fechamento.ID {
		t.Fatalf("itens = %+v, %+v, %v", comissoes, ajustes, err)
	}

	aberta := true
	abertas, err := c.comissoes.Listar(c.ctx, portscomissao.ComissaoFiltro{Aberta: &aberta})
	if err != nil || len(abertas) != 1 || abertas[0].ID != futura.ID || abertas[0].FechamentoID != nil {
		t.Fatalf("comissões abertas = %+v, %v", abertas, err)
	}
	fechada := false
	if fechadas, err := c.comissoes.ListarAjustes(c.ctx, portscomissao.AjusteFiltro{Aberto: &fechada}); err != nil || len(fechadas) != 1 {
		t.Fatalf("ajustes fechados = %+v, %v", fechadas, err)
	}

	listados, err := c.comissoes.ListarFechamentos(c.ctx, portscomissao.FechamentoFiltro{RepresentanteID: &c.carlos.ID, Competencia: &c.anterior})
	if err != nil || len(listados) != 1 || listados[0].ID != fechamento.ID {
		t.Fatalf("fechamentos = %+v, %v", listados, err)
	}
	if daRed, err := c.comissoes.ListarFechamentos(c.ctx, portscomissao.FechamentoFiltro{EmpresaIDs: []uuid.UUID{c.red}}); err != nil || len(daRed) != 0 {
		t.Fatalf("fechamentos da Red = %+v, %v", daRed, err)
	}
}

func TestComissaoRepositoryRecusaFechamentoComSaldoDesatualizado(t *testing.T) {
	c := novoCenarioFechamentos(t)
	c.comissaoNaCompetencia(t, c.anterior, 2)
	saldo := c.saldos(t)[0]
	c.ajusteNaCompetencia(t, c.anterior, -50)

	gravacao := c.gravacao(t, saldo)
	esperarErro(t, c.comissoes.Fechar(c.ctx, gravacao), domain.ErrConflito, "mudaram durante o fechamento")

	if fechamento, err := c.comissoes.ObterFechamento(c.ctx, gravacao.Fechamento.ID); err != nil || fechamento != nil {
		t.Fatalf("nada deveria ser gravado: %+v, %v", fechamento, err)
	}
	if solicitacao, err := postgres.NewSolicitacaoRepository(c.banco).Obter(c.ctx, gravacao.Solicitacao.ID); err != nil || solicitacao != nil {
		t.Fatalf("a conta a pagar não deveria existir: %+v, %v", solicitacao, err)
	}
}

func TestComissaoRepositoryRecusaSegundoFechamentoNaMesmaCompetencia(t *testing.T) {
	c := novoCenarioFechamentos(t)
	c.comissaoNaCompetencia(t, c.anterior, 2)
	if err := c.comissoes.Fechar(c.ctx, c.gravacao(t, c.saldos(t)[0])); err != nil {
		t.Fatalf("primeiro Fechar: %v", err)
	}

	c.ajusteNaCompetencia(t, c.anterior, 500)
	segundo := c.gravacao(t, c.saldos(t)[0])
	esperarErro(t, c.comissoes.Fechar(c.ctx, segundo), domain.ErrConflito, "já existe fechamento")
}

func TestRepresentanteRepositoryGravaOFornecedor(t *testing.T) {
	c := novoCenarioFechamentos(t)
	representantes := postgres.NewRepresentanteRepository(c.banco)

	c.carlos.DefinirFornecedor(&c.fornecedorID, time.Now().UTC())
	if atualizado, err := representantes.Atualizar(c.ctx, c.carlos); err != nil || !atualizado {
		t.Fatalf("Atualizar = %v, %v", atualizado, err)
	}
	lido, err := representantes.Obter(c.ctx, c.carlos.ID)
	if err != nil || lido.FornecedorID == nil || *lido.FornecedorID != c.fornecedorID {
		t.Fatalf("representante lido = %+v, %v", lido, err)
	}

	inexistente := uuid.New()
	lido.DefinirFornecedor(&inexistente, time.Now().UTC())
	_, err = representantes.Atualizar(c.ctx, lido)
	esperarErro(t, err, domain.ErrValidacao, "registro relacionado não encontrado")
}
