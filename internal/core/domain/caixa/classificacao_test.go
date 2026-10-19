package caixa_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	"github.com/google/uuid"
)

func subcategoria(natureza domainfinanceiro.Natureza) *domainfinanceiro.CategoriaFinanceira {
	pai := uuid.New()

	return &domainfinanceiro.CategoriaFinanceira{ID: uuid.New(), Nome: "Sub", CategoriaPaiID: &pai, Natureza: natureza, Ativa: true}
}

func contaDaEmpresa(empresaID uuid.UUID) *domainfinanceiro.ContaBancaria {
	return &domainfinanceiro.ContaBancaria{ID: uuid.New(), EmpresaID: empresaID, Ativa: true}
}

func TestClassificarLinhaCriaLancamentoConciliado(t *testing.T) {
	empresa := uuid.New()
	linha := lancamentoDe(uuid.New(), data(2026, 9, 30), -1290)
	linha.Descricao = "TARIFA PACOTE"
	categoria := subcategoria(domainfinanceiro.NaturezaDespesa)

	classificacao, err := domaincaixa.NovaClassificacao(domaincaixa.DadosClassificacao{
		Lancamento: linha, EmpresaID: empresa, Categoria: categoria, UsuarioID: uuid.New(), Agora: agoraDoTeste,
	})
	if err != nil {
		t.Fatalf("NovaClassificacao = %v", err)
	}

	lancamento := classificacao.Lancamento
	if lancamento.ValorCentavos != -1290 || lancamento.ContaBancariaID != linha.ContaBancariaID || lancamento.EmpresaID != empresa ||
		lancamento.CategoriaFinanceiraID != categoria.ID || lancamento.Descricao != "TARIFA PACOTE" || !lancamento.Data.Equal(linha.Data) {
		t.Errorf("lançamento de caixa = %+v", lancamento)
	}
	if !lancamento.Competencia.Equal(data(2026, 9, 1)) {
		t.Errorf("competência = %v", lancamento.Competencia)
	}

	conciliacao := classificacao.Conciliacao
	if conciliacao.Modo != domaincaixa.ConciliacaoClassificacao || !conciliacao.Modo.CriaOMovimento() || len(conciliacao.Itens) != 1 ||
		conciliacao.Itens[0].Tipo != domaincaixa.MovimentoLancamentoCaixa || conciliacao.Itens[0].AlvoID != lancamento.ID {
		t.Errorf("conciliação = %+v", conciliacao)
	}

	rendimento, err := domaincaixa.NovaClassificacao(domaincaixa.DadosClassificacao{
		Lancamento: lancamentoDe(uuid.New(), data(2026, 10, 1), 350), EmpresaID: empresa,
		Categoria: subcategoria(domainfinanceiro.NaturezaReceita), Descricao: "  Rendimento   CDB ", UsuarioID: uuid.New(), Agora: agoraDoTeste,
	})
	if err != nil || rendimento.Lancamento.Descricao != "Rendimento CDB" {
		t.Errorf("rendimento = %+v, %v", rendimento, err)
	}
}

func TestClassificarRecusaCategoriaIncompativel(t *testing.T) {
	inativa := subcategoria(domainfinanceiro.NaturezaDespesa)
	inativa.Ativa = false
	raiz := &domainfinanceiro.CategoriaFinanceira{ID: uuid.New(), Natureza: domainfinanceiro.NaturezaDespesa, Ativa: true}
	conciliada := lancamentoDe(uuid.New(), data(2026, 10, 1), -10)
	outra := uuid.New()
	conciliada.ConciliacaoID = &outra

	casos := []struct {
		nome      string
		linha     *domaincaixa.LancamentoExtrato
		categoria *domainfinanceiro.CategoriaFinanceira
		sentinela error
		trecho    string
	}{
		{"sem categoria", lancamentoDe(uuid.New(), data(2026, 10, 1), -10), nil, domain.ErrValidacao, "não encontrada"},
		{"inativa", lancamentoDe(uuid.New(), data(2026, 10, 1), -10), inativa, domain.ErrValidacao, "subcategoria ativa"},
		{"raiz", lancamentoDe(uuid.New(), data(2026, 10, 1), -10), raiz, domain.ErrValidacao, "subcategoria ativa"},
		{"receita em débito", lancamentoDe(uuid.New(), data(2026, 10, 1), -10), subcategoria(domainfinanceiro.NaturezaReceita), domain.ErrValidacao, "linha de débito só recebe categoria de despesa"},
		{"despesa em crédito", lancamentoDe(uuid.New(), data(2026, 10, 1), 10), subcategoria(domainfinanceiro.NaturezaDespesa), domain.ErrValidacao, "linha de crédito só recebe categoria de receita"},
		{"transferência", lancamentoDe(uuid.New(), data(2026, 10, 1), 10), subcategoria(domainfinanceiro.NaturezaTransferencia), domain.ErrValidacao, "rota própria"},
		{"linha conciliada", conciliada, subcategoria(domainfinanceiro.NaturezaDespesa), domain.ErrConflito, "já conciliado"},
	}

	for _, caso := range casos {
		_, err := domaincaixa.NovaClassificacao(domaincaixa.DadosClassificacao{
			Lancamento: caso.linha, EmpresaID: uuid.New(), Categoria: caso.categoria, UsuarioID: uuid.New(), Agora: agoraDoTeste,
		})
		if !errors.Is(err, caso.sentinela) || !strings.Contains(err.Error(), caso.trecho) {
			t.Errorf("%s: erro = %v, esperado %v com %q", caso.nome, err, caso.sentinela, caso.trecho)
		}
	}
}

func TestTransferenciaPelaPontaDeSaidaEDeEntrada(t *testing.T) {
	mia, red := uuid.New(), uuid.New()
	sicredi, itau, aplicacao := contaDaEmpresa(mia), contaDaEmpresa(red), contaDaEmpresa(mia)
	categoria := subcategoria(domainfinanceiro.NaturezaTransferencia)

	saida, err := domaincaixa.NovaTransferencia(domaincaixa.DadosTransferencia{
		Lancamento: lancamentoDe(sicredi.ID, data(2026, 10, 5), -500000), Conta: sicredi, Contraparte: aplicacao,
		Categoria: categoria, UsuarioID: uuid.New(), Agora: agoraDoTeste,
	})
	if err != nil {
		t.Fatalf("NovaTransferencia de saída = %v", err)
	}
	transferencia := saida.Transferencia
	if transferencia.ContaOrigemID != sicredi.ID || transferencia.ContaDestinoID != aplicacao.ID || transferencia.ValorCentavos != 500000 ||
		transferencia.EntreEmpresas || !transferencia.Data.Equal(data(2026, 10, 5)) {
		t.Errorf("transferência de saída = %+v", transferencia)
	}
	item := saida.Conciliacao.Itens[0]
	if saida.Conciliacao.Modo != domaincaixa.ConciliacaoTransferencia || item.Tipo != domaincaixa.MovimentoTransferenciaSaida ||
		item.AlvoID != transferencia.ID || item.ValorCentavos != -500000 {
		t.Errorf("conciliação da saída = %+v", saida.Conciliacao)
	}

	entrada, err := domaincaixa.NovaTransferencia(domaincaixa.DadosTransferencia{
		Lancamento: lancamentoDe(sicredi.ID, data(2026, 10, 6), 300000), Conta: sicredi, Contraparte: itau,
		Categoria: categoria, UsuarioID: uuid.New(), Agora: agoraDoTeste,
	})
	if err != nil {
		t.Fatalf("NovaTransferencia de entrada = %v", err)
	}
	if entrada.Transferencia.ContaOrigemID != itau.ID || entrada.Transferencia.ContaDestinoID != sicredi.ID ||
		entrada.Transferencia.ValorCentavos != 300000 || !entrada.Transferencia.EntreEmpresas ||
		entrada.Conciliacao.Itens[0].Tipo != domaincaixa.MovimentoTransferenciaEntrada {
		t.Errorf("transferência de entrada = %+v", entrada.Transferencia)
	}
}

func TestTransferenciaRecusaContraparteECategoriaInvalidas(t *testing.T) {
	empresa := uuid.New()
	conta, outra := contaDaEmpresa(empresa), contaDaEmpresa(empresa)
	inativa := contaDaEmpresa(empresa)
	inativa.Ativa = false

	casos := map[string]domaincaixa.DadosTransferencia{
		"mesma conta":         {Contraparte: conta, Categoria: subcategoria(domainfinanceiro.NaturezaTransferencia)},
		"contraparte inativa": {Contraparte: inativa, Categoria: subcategoria(domainfinanceiro.NaturezaTransferencia)},
		"sem contraparte":     {Categoria: subcategoria(domainfinanceiro.NaturezaTransferencia)},
		"categoria despesa":   {Contraparte: outra, Categoria: subcategoria(domainfinanceiro.NaturezaDespesa)},
	}
	for nome, dados := range casos {
		dados.Lancamento = lancamentoDe(conta.ID, data(2026, 10, 5), -10)
		dados.Conta = conta
		dados.UsuarioID = uuid.New()
		dados.Agora = agoraDoTeste
		if _, err := domaincaixa.NovaTransferencia(dados); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("%s: erro = %v, esperado erro de validação", nome, err)
		}
	}
}

func TestTiposDeMovimentoConciliaveisAMao(t *testing.T) {
	for _, valor := range []string{"pagamento", "recebimento", "antecipacao", "transferencia_saida", " TRANSFERENCIA_ENTRADA "} {
		if _, err := domaincaixa.NovoTipoMovimento(valor); err != nil {
			t.Errorf("%q = %v", valor, err)
		}
	}
	for _, valor := range []string{"lancamento_caixa", "transferencia", ""} {
		if _, err := domaincaixa.NovoTipoMovimento(valor); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("%q = %v, esperado erro de validação", valor, err)
		}
	}
}
