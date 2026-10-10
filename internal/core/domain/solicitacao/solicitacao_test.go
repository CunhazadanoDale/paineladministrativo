package solicitacao_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

func novaSolicitacaoDeTeste(t *testing.T) *domainsolicitacao.Solicitacao {
	t.Helper()

	solicitanteID := uuid.New()
	valor, err := domainsolicitacao.NovoValor(150000)
	if err != nil {
		t.Fatalf("valor de teste inválido: %v", err)
	}
	prazo, err := domainsolicitacao.NovoPrazoPagamento(time.Now().AddDate(0, 0, 7))
	if err != nil {
		t.Fatalf("prazo de teste inválido: %v", err)
	}
	observacao, err := domainsolicitacao.NovaObservacao("compra de cimento")
	if err != nil {
		t.Fatalf("observação de teste inválida: %v", err)
	}
	forma, err := domainsolicitacao.NovaFormaPagamento("pix")
	if err != nil {
		t.Fatalf("forma de pagamento de teste inválida: %v", err)
	}

	solicitacao, err := domainsolicitacao.NovaSolicitacao(solicitanteID, valor, prazo, observacao, forma, nil)
	if err != nil {
		t.Fatalf("criação da solicitação falhou: %v", err)
	}

	return solicitacao
}

func TestNovaSolicitacaoNascePendente(t *testing.T) {
	solicitacao := novaSolicitacaoDeTeste(t)

	if solicitacao.ID == uuid.Nil {
		t.Error("solicitação criada sem id")
	}
	if solicitacao.Status != domainsolicitacao.StatusPendenteAprovacao {
		t.Errorf("status = %q, esperado %q", solicitacao.Status, domainsolicitacao.StatusPendenteAprovacao)
	}
	if solicitacao.SolicitanteID == uuid.Nil {
		t.Error("solicitação criada sem solicitante")
	}
}

func TestNovaSolicitacaoRejeitaSolicitanteAusente(t *testing.T) {
	solicitacaoValida := novaSolicitacaoDeTeste(t)

	if _, err := domainsolicitacao.NovaSolicitacao(
		uuid.Nil,
		solicitacaoValida.Valor,
		solicitacaoValida.Prazo,
		solicitacaoValida.Observacao,
		solicitacaoValida.FormaPagamento,
		nil,
	); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("sem solicitante = %v, esperado erro de validação", err)
	}
}

func TestSolicitacaoAprovadaSoAposTransicaoValida(t *testing.T) {
	solicitacao := novaSolicitacaoDeTeste(t)
	aprovadorID := uuid.New()
	agora := time.Now().UTC()

	if err := solicitacao.Aprovar(aprovadorID, agora); err != nil {
		t.Fatalf("aprovação falhou: %v", err)
	}
	if solicitacao.Status != domainsolicitacao.StatusAprovado {
		t.Errorf("status = %q, esperado %q", solicitacao.Status, domainsolicitacao.StatusAprovado)
	}
	if solicitacao.AprovadorID == nil || *solicitacao.AprovadorID != aprovadorID {
		t.Errorf("aprovador = %v, esperado %s", solicitacao.AprovadorID, aprovadorID)
	}
	if solicitacao.AprovadoEm == nil {
		t.Error("aprovação não registrou a data")
	}
	if err := solicitacao.Aprovar(aprovadorID, agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("reaprovacao = %v, esperado erro de conflito", err)
	}
}

func TestSolicitanteNaoAprovaAPropriaSolicitacao(t *testing.T) {
	solicitacao := novaSolicitacaoDeTeste(t)

	if err := solicitacao.Aprovar(solicitacao.SolicitanteID, time.Now().UTC()); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("autoaprovação = %v, esperado erro de permissão", err)
	}
	if solicitacao.Status != domainsolicitacao.StatusPendenteAprovacao {
		t.Errorf("status = %q, esperado %q", solicitacao.Status, domainsolicitacao.StatusPendenteAprovacao)
	}
}

func TestSolicitacaoRejeitadaExigeMotivoEConflitaDepois(t *testing.T) {
	solicitacao := novaSolicitacaoDeTeste(t)
	motivo, err := domainsolicitacao.NovoMotivoRejeicao("orçamento insuficiente")
	if err != nil {
		t.Fatalf("motivo de teste inválido: %v", err)
	}

	if err := solicitacao.Rejeitar(uuid.New(), motivo, time.Now().UTC()); err != nil {
		t.Fatalf("rejeição falhou: %v", err)
	}
	if solicitacao.Status != domainsolicitacao.StatusRejeitado {
		t.Errorf("status = %q, esperado %q", solicitacao.Status, domainsolicitacao.StatusRejeitado)
	}
	if solicitacao.MotivoRejeicao == nil || solicitacao.MotivoRejeicao.Texto() != "orçamento insuficiente" {
		t.Errorf("motivo = %v, esperado registrado", solicitacao.MotivoRejeicao)
	}
	if err := solicitacao.Cancelar(time.Now().UTC()); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("cancelar rejeitada = %v, esperado erro de conflito", err)
	}
}

func TestSolicitacaoSoPagaAposAprovacao(t *testing.T) {
	solicitacao := novaSolicitacaoDeTeste(t)

	if err := solicitacao.MarcarComoPago(time.Now().UTC()); !errors.Is(err, domain.ErrConflito) {
		t.Fatalf("pagar pendente = %v, esperado erro de conflito", err)
	}

	if err := solicitacao.Aprovar(uuid.New(), time.Now().UTC()); err != nil {
		t.Fatalf("aprovação falhou: %v", err)
	}
	if err := solicitacao.MarcarComoPago(time.Now().UTC()); err != nil {
		t.Fatalf("pagamento após aprovação falhou: %v", err)
	}
	if solicitacao.Status != domainsolicitacao.StatusPago {
		t.Errorf("status = %q, esperado %q", solicitacao.Status, domainsolicitacao.StatusPago)
	}
}

func TestCancelamentoDependeDoPerfil(t *testing.T) {
	solicitanteID := uuid.New()
	solicitacao := novaSolicitacaoDeTeste(t)
	solicitacao.SolicitanteID = solicitanteID

	if solicitacao.PodeSerCanceladoPor(uuid.New(), false) {
		t.Error("estranho não deveria cancelar")
	}
	if !solicitacao.PodeSerCanceladoPor(solicitanteID, false) {
		t.Error("solicitante deveria cancelar")
	}
	if !solicitacao.PodeSerCanceladoPor(uuid.New(), true) {
		t.Error("administrador deveria cancelar")
	}

	if err := solicitacao.Cancelar(time.Now().UTC()); err != nil {
		t.Fatalf("cancelamento falhou: %v", err)
	}
	if err := solicitacao.Cancelar(time.Now().UTC()); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("recancelamento = %v, esperado erro de conflito", err)
	}
}
