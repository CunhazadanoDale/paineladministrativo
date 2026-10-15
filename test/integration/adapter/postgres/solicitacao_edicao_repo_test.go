package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
)

func (c *cenarioBancoClassificacao) pendente(t *testing.T) *domainsolicitacao.Solicitacao {
	t.Helper()

	solicitacao := c.novaSolicitacao(t)
	if err := c.gravar(t, solicitacao); err != nil {
		t.Fatalf("criação falhou: %v", err)
	}

	return solicitacao
}

func historicoDoEvento(solicitacao *domainsolicitacao.Solicitacao, evento domainsolicitacao.Evento, descricao string) *domainsolicitacao.Historico {
	status := solicitacao.Status

	return domainsolicitacao.NovoHistorico(solicitacao.ID, solicitacao.SolicitanteID, &status, status, evento, descricao)
}

func dadosEditados(t *testing.T, c *cenarioBancoClassificacao, solicitacao *domainsolicitacao.Solicitacao) domainsolicitacao.DadosEdicao {
	t.Helper()

	valor, err := domainsolicitacao.NovoValor(77700)
	if err != nil {
		t.Fatalf("valor de teste inválido: %v", err)
	}
	observacao, err := domainsolicitacao.NovaObservacao("conta de energia de setembro")
	if err != nil {
		t.Fatalf("observação de teste inválida: %v", err)
	}

	return domainsolicitacao.DadosEdicao{
		Valor:          valor,
		Prazo:          domainsolicitacao.PrazoPagamentoDe(solicitacao.Prazo.Data().AddDate(0, 0, 3)),
		Observacao:     observacao,
		FormaPagamento: domainsolicitacao.FormaPagamentoTransferencia,
		Classificacao:  c.classificacao(t, "2026-09"),
		Estimado:       true,
	}
}

func TestEditarGravaOsCamposAvancaVersaoEGravaEvento(t *testing.T) {
	c := cenarioClassificacao(t)
	ctx := context.Background()
	solicitacao := c.pendente(t)

	versaoLida := solicitacao.Versao
	if _, err := solicitacao.Editar(dadosEditados(t, c, solicitacao), time.Now().UTC()); err != nil {
		t.Fatalf("edição no domínio falhou: %v", err)
	}
	if err := c.solicitacoes.Editar(ctx, solicitacao, versaoLida,
		historicoDoEvento(solicitacao, domainsolicitacao.EventoEditada, "solicitação editada: valor")); err != nil {
		t.Fatalf("edição falhou: %v", err)
	}

	gravada := c.obter(t, solicitacao.ID)
	if gravada.Versao != 2 || gravada.Valor.Centavos() != 77700 || gravada.Observacao.Texto() != "conta de energia de setembro" {
		t.Errorf("gravada = versão %d, valor %d, observação %q", gravada.Versao, gravada.Valor.Centavos(), gravada.Observacao.Texto())
	}
	if gravada.Prazo.String() != solicitacao.Prazo.String() {
		t.Errorf("vencimento = %s, esperado %s", gravada.Prazo, solicitacao.Prazo)
	}
	if gravada.FormaPagamento != domainsolicitacao.FormaPagamentoTransferencia || !gravada.Estimado || gravada.Competencia.String() != "2026-09" {
		t.Errorf("forma = %q, estimado = %v, competência = %s", gravada.FormaPagamento, gravada.Estimado, gravada.Competencia)
	}
	if gravada.FornecedorID == nil || *gravada.FornecedorID != c.fornecedorID || gravada.CategoriaFinanceiraID == nil || *gravada.CategoriaFinanceiraID != c.despesaID {
		t.Errorf("classificação = %v/%v", gravada.FornecedorID, gravada.CategoriaFinanceiraID)
	}

	eventos, err := c.solicitacoes.ListarHistorico(ctx, solicitacao.ID)
	if err != nil || len(eventos) != 2 || eventos[0].Evento != domainsolicitacao.EventoEditada {
		t.Errorf("histórico = %+v, %v; esperado editada no topo", eventos, err)
	}
}

func TestEditarRecusaVersaoVelhaStatusErradoEClassificacaoInvalida(t *testing.T) {
	c := cenarioClassificacao(t)
	ctx := context.Background()

	velha := c.pendente(t)
	if _, err := velha.Editar(dadosEditados(t, c, velha), time.Now().UTC()); err != nil {
		t.Fatalf("edição no domínio falhou: %v", err)
	}
	if err := c.solicitacoes.Editar(ctx, velha, 2, historicoDoEvento(velha, domainsolicitacao.EventoEditada, "x")); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("versão velha = %v, esperado conflito", err)
	}

	aprovada := c.aprovada(t)
	aprovada.Valor = domainsolicitacao.ValorDe(1)
	if err := c.solicitacoes.Editar(ctx, aprovada, aprovada.Versao, historicoDoEvento(aprovada, domainsolicitacao.EventoEditada, "x")); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("editar aprovada = %v, esperado conflito", err)
	}
	if gravada := c.obter(t, aprovada.ID); gravada.Valor.Centavos() == 1 {
		t.Error("edição recusada gravou o valor")
	}

	receita := c.pendente(t)
	versaoLida := receita.Versao
	dados := dadosEditados(t, c, receita)
	dados.Classificacao.CategoriaFinanceiraID = &c.receitaID
	if _, err := receita.Editar(dados, time.Now().UTC()); err != nil {
		t.Fatalf("edição no domínio falhou: %v", err)
	}
	if err := c.solicitacoes.Editar(ctx, receita, versaoLida, historicoDoEvento(receita, domainsolicitacao.EventoEditada, "x")); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("categoria de receita = %v, esperado erro de validação", err)
	}
}

func TestAdiarGravaVencimentoSoEmAprovada(t *testing.T) {
	c := cenarioClassificacao(t)
	ctx := context.Background()

	aprovada := c.aprovada(t)
	competencia := aprovada.Competencia
	versaoLida := aprovada.Versao
	novo := domainsolicitacao.PrazoPagamentoDe(aprovada.Prazo.Data().AddDate(0, 0, 20))
	if _, err := aprovada.Adiar(novo, time.Now().UTC()); err != nil {
		t.Fatalf("adiamento no domínio falhou: %v", err)
	}
	if err := c.solicitacoes.Adiar(ctx, aprovada, versaoLida,
		historicoDoEvento(aprovada, domainsolicitacao.EventoAdiada, "vencimento adiado")); err != nil {
		t.Fatalf("adiamento falhou: %v", err)
	}

	gravada := c.obter(t, aprovada.ID)
	if gravada.Prazo.String() != novo.String() || gravada.Versao != versaoLida+1 || gravada.Competencia != competencia {
		t.Errorf("gravada = vencimento %s, versão %d, competência %s", gravada.Prazo, gravada.Versao, gravada.Competencia)
	}

	eventos, err := c.solicitacoes.ListarHistorico(ctx, aprovada.ID)
	if err != nil || eventos[0].Evento != domainsolicitacao.EventoAdiada || eventos[0].ParaStatus != domainsolicitacao.StatusAprovado {
		t.Errorf("histórico = %+v, %v; esperado adiada no topo", eventos, err)
	}

	pendente := c.pendente(t)
	pendente.Prazo = novo
	if err := c.solicitacoes.Adiar(ctx, pendente, pendente.Versao, historicoDoEvento(pendente, domainsolicitacao.EventoAdiada, "x")); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("adiar pendente = %v, esperado conflito", err)
	}
}
