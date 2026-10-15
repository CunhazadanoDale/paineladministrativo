package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
)

func TestPagarDeNovoComOMesmoComprovanteDepoisDoEstorno(t *testing.T) {
	c := cenarioClassificacao(t)
	contaID := c.inserirConta(t, c.empresaID, "54321", "Movimento")
	solicitacao := c.aprovadaClassificada(t)
	arquivo := c.criarArquivo(t, "comprovante.pdf", "application/pdf")

	primeiro, err := c.pagar(t, solicitacao, contaID, &arquivo.ID)
	if err != nil {
		t.Fatalf("pagamento falhou: %v", err)
	}
	if err := c.estornar(t, solicitacao, primeiro); err != nil {
		t.Fatalf("estorno falhou: %v", err)
	}

	if _, err := c.pagar(t, solicitacao, contaID, &arquivo.ID); err != nil {
		t.Errorf("pagar de novo com o mesmo comprovante = %v, esperado permitido", err)
	}

	outra := c.aprovadaClassificada(t)
	if _, err := c.pagar(t, outra, contaID, &arquivo.ID); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("mesmo comprovante em pagamento vigente de outra solicitação = %v, esperado conflito", err)
	}
}

func TestCriarPagamentoRecusaCategoriaDesativadaNaTransacao(t *testing.T) {
	c := cenarioClassificacao(t)
	ctx := context.Background()
	contaID := c.inserirConta(t, c.empresaID, "54321", "Movimento")
	solicitacao := c.aprovadaClassificada(t)

	if _, err := c.banco.ExecContext(ctx, `UPDATE categoria_financeira SET ativa = false WHERE id = $1`, c.despesaID); err != nil {
		t.Fatalf("desativação falhou: %v", err)
	}

	if _, err := c.pagar(t, solicitacao, contaID, nil); err == nil || err.Error() != domainsolicitacao.ErroCategoriaFinanceiraNaoLancavel().Error() {
		t.Errorf("pagar com categoria desativada = %v", err)
	}
}

func TestTransicaoComVersaoAntigaNaoGrava(t *testing.T) {
	c := cenarioClassificacao(t)
	ctx := context.Background()
	solicitacao := c.pendente(t)

	editada := *solicitacao
	if _, err := editada.Editar(dadosEditados(t, c, &editada), time.Now().UTC()); err != nil {
		t.Fatalf("edição no domínio falhou: %v", err)
	}
	if err := c.solicitacoes.Editar(ctx, &editada, 1, historicoDoEvento(&editada, domainsolicitacao.EventoEditada, "x")); err != nil {
		t.Fatalf("edição falhou: %v", err)
	}

	statusAnterior := solicitacao.Status
	if err := solicitacao.Aprovar(c.terceiroID, time.Now().UTC()); err != nil {
		t.Fatalf("aprovação no domínio falhou: %v", err)
	}
	historico := domainsolicitacao.NovoHistorico(
		solicitacao.ID, c.terceiroID, &statusAnterior, solicitacao.Status, domainsolicitacao.EventoAprovada, "x",
	)
	atualizado, err := c.solicitacoes.AtualizarStatus(ctx, solicitacao, historico)
	if err != nil || atualizado {
		t.Errorf("aprovar com a leitura anterior à edição = %v, %v; esperado não gravar", atualizado, err)
	}
	if gravada := c.obter(t, solicitacao.ID); gravada.Status != domainsolicitacao.StatusPendenteAprovacao || gravada.Versao != 2 {
		t.Errorf("gravada = status %q, versão %d", gravada.Status, gravada.Versao)
	}
}

func TestGerarDeRecorrenciaJaGeradaNaoValidaDeNovo(t *testing.T) {
	c := cenarioRecorrencias(t)
	ctx := context.Background()
	despesa := c.novaRecorrencia(t, c.empresaID, "Aluguel", "2026-01", nil)

	if _, criada, err := c.gerar(t, despesa, "2026-03"); err != nil || !criada {
		t.Fatalf("primeira geração = %v, %v", criada, err)
	}
	if _, err := c.banco.ExecContext(ctx, `UPDATE categoria_financeira SET ativa = false WHERE id = $1`, c.despesaID); err != nil {
		t.Fatalf("desativação falhou: %v", err)
	}

	if _, criada, err := c.gerar(t, despesa, "2026-03"); err != nil || criada {
		t.Errorf("regerar competência já gerada = %v, %v; esperado ignorada sem erro", criada, err)
	}
}
