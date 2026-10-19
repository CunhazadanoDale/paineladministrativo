package postgres_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	portscaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/caixa"
	"github.com/google/uuid"
)

func (c *cenarioBancoConciliacao) categoria(t *testing.T, id uuid.UUID) *domainfinanceiro.CategoriaFinanceira {
	t.Helper()

	categoria, err := postgres.NewCategoriaFinanceiraRepository(c.banco).Obter(c.ctx, id)
	if err != nil || categoria == nil {
		t.Fatalf("categoria %s = %+v, %v", id, categoria, err)
	}

	return categoria
}

func (c *cenarioBancoConciliacao) conta(t *testing.T, id uuid.UUID) *domainfinanceiro.ContaBancaria {
	t.Helper()

	conta, err := postgres.NewContaBancariaRepository(c.banco).Obter(c.ctx, id)
	if err != nil || conta == nil {
		t.Fatalf("conta %s = %+v, %v", id, conta, err)
	}

	return conta
}

func (c *cenarioBancoConciliacao) desfazer(t *testing.T, conciliacaoID uuid.UUID) {
	t.Helper()

	conciliacao, err := c.conciliacoes.Obter(c.ctx, conciliacaoID)
	if err != nil || conciliacao == nil {
		t.Fatalf("Obter = %+v, %v", conciliacao, err)
	}
	if err := conciliacao.Desfazer(c.solicitanteID, "lançado errado", time.Now().UTC()); err != nil {
		t.Fatalf("Desfazer no domínio = %v", err)
	}
	if err := c.conciliacoes.Desfazer(c.ctx, conciliacao); err != nil {
		t.Fatalf("Desfazer = %v", err)
	}
}

func TestClassificacaoGravaLancamentoDeCaixaEDesfazEstorna(t *testing.T) {
	c := novoCenarioConciliacao(t)
	preparado := c.importarLinhas(t, transacaoNoDia("TAR", diaDoTitulo(10, 10), -1290))
	linha := c.lancamento(t, preparado.Lancamentos[0].ID)
	lancamentos := postgres.NewLancamentoCaixaRepository(c.banco)

	classificacao, err := domaincaixa.NovaClassificacao(domaincaixa.DadosClassificacao{
		Lancamento: linha, EmpresaID: c.empresaID, Categoria: c.categoria(t, c.despesaID), UsuarioID: c.solicitanteID, Agora: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("NovaClassificacao = %v", err)
	}
	if err := lancamentos.Classificar(c.ctx, classificacao); err != nil {
		t.Fatalf("Classificar = %v", err)
	}

	if classificada := c.lancamento(t, linha.ID); classificada.Situacao != domaincaixa.LancamentoConciliado {
		t.Errorf("linha classificada = %+v", classificada)
	}
	salvos, err := lancamentos.Listar(c.ctx, portscaixa.LancamentoCaixaFiltro{ContaBancariaID: &c.contaID, Limite: 10})
	if err != nil || len(salvos) != 1 {
		t.Fatalf("Listar = %+v, %v", salvos, err)
	}
	salvo := salvos[0]
	if salvo.ValorCentavos != -1290 || !salvo.Competencia.Equal(diaDoTitulo(10, 1)) || !salvo.Data.Equal(diaDoTitulo(10, 10)) ||
		salvo.CategoriaFinanceiraID != c.despesaID || salvo.Estornado || salvo.Descricao != "TAR" {
		t.Errorf("lançamento de caixa = %+v", salvo)
	}

	repetida, err := domaincaixa.NovaClassificacao(domaincaixa.DadosClassificacao{
		Lancamento: linha, EmpresaID: c.empresaID, Categoria: c.categoria(t, c.despesaID), UsuarioID: c.solicitanteID, Agora: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("NovaClassificacao repetida = %v", err)
	}
	if err := lancamentos.Classificar(c.ctx, repetida); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("classificar de novo = %v", err)
	}

	c.desfazer(t, classificacao.Conciliacao.ID)
	estornados, _ := lancamentos.Listar(c.ctx, portscaixa.LancamentoCaixaFiltro{EmpresaIDs: []uuid.UUID{c.empresaID}, Limite: 10})
	if len(estornados) != 1 || !estornados[0].Estornado {
		t.Errorf("lançamento depois de desfazer = %+v", estornados)
	}
	if pendente := c.lancamento(t, linha.ID); pendente.Situacao != domaincaixa.LancamentoPendente {
		t.Errorf("linha depois de desfazer = %+v", pendente)
	}

	comCategoriaErrada, err := domaincaixa.NovaClassificacao(domaincaixa.DadosClassificacao{
		Lancamento: c.lancamento(t, linha.ID), EmpresaID: c.empresaID, Categoria: c.categoria(t, c.despesaID),
		UsuarioID: c.solicitanteID, Agora: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("NovaClassificacao = %v", err)
	}
	comCategoriaErrada.Lancamento.CategoriaFinanceiraID = c.receitaID
	if err := lancamentos.Classificar(c.ctx, comCategoriaErrada); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("categoria de receita conferida na transação = %v", err)
	}
}

func TestTransferenciaEntreContasConciliaAsDuasPontas(t *testing.T) {
	c := novoCenarioConciliacao(t)
	lancamentos := postgres.NewLancamentoCaixaRepository(c.banco)
	raiz := c.inserirCategoria(t, "Movimentação entre contas", "transferencia", "entre_contas", nil)
	entreContas := c.inserirCategoria(t, "Aplicação", "", "", raiz)

	aplicacaoID := c.inserirConta(t, c.empresaID, "55555", "Aplicação")
	aberturaDaAplicacao, err := domaincaixa.NovaAbertura(domaincaixa.DadosAbertura{
		ContaBancariaID: aplicacaoID, DataCorte: diaDoTitulo(10, 1), UsuarioID: c.solicitanteID,
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("NovaAbertura = %v", err)
	}
	if err := postgres.NewCaixaAberturaRepository(c.banco).Criar(c.ctx, aberturaDaAplicacao); err != nil {
		t.Fatalf("abertura da aplicação = %v", err)
	}

	saida := c.importarLinhas(t, transacaoNoDia("APL", diaDoTitulo(10, 5), -500000))
	linhaDeSaida := c.lancamento(t, saida.Lancamentos[0].ID)
	registrada, err := domaincaixa.NovaTransferencia(domaincaixa.DadosTransferencia{
		Lancamento: linhaDeSaida, Conta: c.conta(t, c.contaID), Contraparte: c.conta(t, aplicacaoID),
		Categoria: entreContas, UsuarioID: c.solicitanteID, Agora: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("NovaTransferencia = %v", err)
	}
	if err := lancamentos.Transferir(c.ctx, registrada); err != nil {
		t.Fatalf("Transferir = %v", err)
	}

	pendentesNaAplicacao, err := c.conciliacoes.Movimentos(c.ctx, portscaixa.MovimentoFiltro{ContaBancariaID: aplicacaoID, SoPendentes: true})
	if err != nil || len(pendentesNaAplicacao) != 1 {
		t.Fatalf("pendências da aplicação = %+v, %v", pendentesNaAplicacao, err)
	}
	entrada := pendentesNaAplicacao[0]
	if entrada.Tipo != domaincaixa.MovimentoTransferenciaEntrada || entrada.ValorCentavos != 500000 || entrada.ID != registrada.Transferencia.ID ||
		entrada.Descricao != "Transferência de Cobrança - APL" {
		t.Errorf("ponta de entrada = %+v", entrada)
	}
	if daOrigem := c.movimentos(t, portscaixa.MovimentoFiltro{SoPendentes: true}); len(daOrigem) != 0 {
		t.Errorf("a ponta de saída já nasce conciliada: %+v", daOrigem)
	}

	credito, err := domaincaixa.PrepararImportacao(domaincaixa.DadosImportacao{
		Extrato:         &domaincaixa.ExtratoOFX{Transacoes: []domaincaixa.TransacaoOFX{transacaoNoDia("RES", diaDoTitulo(10, 6), 500000)}},
		ContaBancariaID: aplicacaoID, ArquivoID: c.novoArquivoDoCaixa(t), UsuarioID: c.solicitanteID, Abertura: aberturaDaAplicacao,
		Agora: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("PrepararImportacao = %v", err)
	}
	if err := c.extratos.Importar(c.ctx, credito, aberturaDaAplicacao.Versao); err != nil {
		t.Fatalf("Importar na aplicação = %v", err)
	}
	linhaDeEntrada := c.lancamento(t, credito.Lancamentos[0].ID)
	outraPonta := c.conciliar(t, linhaDeEntrada, entrada)
	if err := c.conciliacoes.Criar(c.ctx, outraPonta); err != nil {
		t.Fatalf("conciliar a outra ponta = %v", err)
	}

	transferencias, err := lancamentos.ListarTransferencias(c.ctx, portscaixa.LancamentoCaixaFiltro{ContaBancariaID: &aplicacaoID, Limite: 10})
	if err != nil || len(transferencias) != 1 || transferencias[0].ContaOrigemID != c.contaID || transferencias[0].EntreEmpresas {
		t.Errorf("transferências = %+v, %v", transferencias, err)
	}

	c.desfazer(t, registrada.Conciliacao.ID)
	if invalidada := c.lancamento(t, linhaDeEntrada.ID); invalidada.Situacao != domaincaixa.LancamentoInvalidado {
		t.Errorf("outra ponta depois de desfazer a transferência = %+v", invalidada)
	}
	estornada, _ := lancamentos.ListarTransferencias(c.ctx, portscaixa.LancamentoCaixaFiltro{Limite: 10})
	if len(estornada) != 1 || !estornada[0].Estornado {
		t.Errorf("transferência depois de desfazer = %+v", estornada)
	}

	if _, err := c.banco.ExecContext(c.ctx, `UPDATE conta_bancaria SET ativa = false WHERE id = $1`, aplicacaoID); err != nil {
		t.Fatalf("desativar a aplicação falhou: %v", err)
	}
	contraparte := c.conta(t, aplicacaoID)
	contraparte.Ativa = true
	paraInativa, err := domaincaixa.NovaTransferencia(domaincaixa.DadosTransferencia{
		Lancamento: c.lancamento(t, linhaDeSaida.ID), Conta: c.conta(t, c.contaID), Contraparte: contraparte,
		Categoria: entreContas, UsuarioID: c.solicitanteID, Agora: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("NovaTransferencia = %v", err)
	}
	if err := lancamentos.Transferir(c.ctx, paraInativa); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("contraparte inativa conferida na transação = %v", err)
	}
}
