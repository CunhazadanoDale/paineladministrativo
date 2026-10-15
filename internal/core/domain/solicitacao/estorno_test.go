package solicitacao_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

func pagamentoDeTeste(t *testing.T, pagoEm time.Time) *domainsolicitacao.Pagamento {
	t.Helper()

	valor, _ := domainsolicitacao.NovoValor(10000)
	pagamento, err := domainsolicitacao.NovoPagamento(uuid.New(), uuid.New(), uuid.New(), valor, nil, pagoEm, time.Now().UTC())
	if err != nil {
		t.Fatalf("pagamento de teste inválido: %v", err)
	}

	return pagamento
}

func TestNovoPagamentoExigeContaBancariaEGuardaQuemRegistrou(t *testing.T) {
	valor, _ := domainsolicitacao.NovoValor(10000)
	contaID := uuid.New()
	usuarioID := uuid.New()

	if _, err := domainsolicitacao.NovoPagamento(uuid.New(), uuid.Nil, usuarioID, valor, nil, time.Time{}, time.Now().UTC()); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("sem conta = %v, esperado erro de validação", err)
	}

	pagamento, err := domainsolicitacao.NovoPagamento(uuid.New(), contaID, usuarioID, valor, nil, time.Time{}, time.Now().UTC())
	if err != nil {
		t.Fatalf("pagamento falhou: %v", err)
	}
	if *pagamento.ContaBancariaID != contaID || *pagamento.UsuarioID != usuarioID || pagamento.Estornado {
		t.Errorf("pagamento = %+v", pagamento)
	}
}

func TestPagarExigeCategoriaNaAprovada(t *testing.T) {
	solicitacao := novaSolicitacaoDeTeste(t)
	if err := solicitacao.Aprovar(uuid.New(), time.Now().UTC()); err != nil {
		t.Fatalf("aprovação falhou: %v", err)
	}

	if err := solicitacao.MarcarComoPago(time.Now().UTC()); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("pagar sem categoria = %v, esperado erro de validação", err)
	}
	if solicitacao.Status != domainsolicitacao.StatusAprovado {
		t.Errorf("status = %q, esperado aprovado", solicitacao.Status)
	}
}

func TestEstornarPagamentoValidaADataEMarcaEstornado(t *testing.T) {
	pagoEm := time.Now().UTC().Add(-48 * time.Hour)
	motivo, err := domainsolicitacao.NovoMotivoEstorno("valor pago em duplicidade")
	if err != nil {
		t.Fatalf("motivo de teste inválido: %v", err)
	}
	usuarioID := uuid.New()

	casos := map[string]time.Time{
		"no futuro":          time.Now().UTC().Add(time.Hour),
		"antes do pagamento": pagoEm.Add(-time.Hour),
	}
	for nome, data := range casos {
		if _, err := pagamentoDeTeste(t, pagoEm).Estornar(motivo, usuarioID, data, time.Now().UTC()); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("%s = %v, esperado erro de validação", nome, err)
		}
	}

	pagamento := pagamentoDeTeste(t, pagoEm)
	estorno, err := pagamento.Estornar(motivo, usuarioID, time.Time{}, time.Now().UTC())
	if err != nil {
		t.Fatalf("estorno falhou: %v", err)
	}
	if !pagamento.Estornado || pagamento.Estorno != estorno || estorno.PagamentoID != pagamento.ID || estorno.UsuarioID != usuarioID {
		t.Errorf("estorno = %+v, pagamento = %+v", estorno, pagamento)
	}
	if estorno.EstornadoEm.IsZero() || estorno.Motivo.Texto() != "valor pago em duplicidade" {
		t.Errorf("estorno = %+v", estorno)
	}

	if _, err := pagamento.Estornar(motivo, usuarioID, time.Time{}, time.Now().UTC()); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("estornar de novo = %v, esperado conflito", err)
	}
}

func TestEstornarSolicitacaoVoltaPagaParaAprovada(t *testing.T) {
	solicitacao := novaSolicitacaoClassificada(t)

	if err := solicitacao.Estornar(time.Now().UTC()); !errors.Is(err, domain.ErrConflito) {
		t.Fatalf("estornar pendente = %v, esperado conflito", err)
	}

	if err := solicitacao.Aprovar(uuid.New(), time.Now().UTC()); err != nil {
		t.Fatalf("aprovação falhou: %v", err)
	}
	if err := solicitacao.MarcarComoPago(time.Now().UTC()); err != nil {
		t.Fatalf("pagamento falhou: %v", err)
	}
	versaoPaga := solicitacao.Versao

	if err := solicitacao.Estornar(time.Now().UTC()); err != nil {
		t.Fatalf("estorno falhou: %v", err)
	}
	if solicitacao.Status != domainsolicitacao.StatusAprovado || solicitacao.Versao != versaoPaga+1 {
		t.Errorf("status = %q, versão = %d", solicitacao.Status, solicitacao.Versao)
	}
	if err := solicitacao.MarcarComoPago(time.Now().UTC()); err != nil {
		t.Errorf("pagar de novo depois do estorno = %v, esperado permitido", err)
	}
}

func TestAprovarNaoReabrePaga(t *testing.T) {
	solicitacao := novaSolicitacaoClassificada(t)
	if err := solicitacao.Aprovar(uuid.New(), time.Now().UTC()); err != nil {
		t.Fatalf("aprovação falhou: %v", err)
	}
	if err := solicitacao.MarcarComoPago(time.Now().UTC()); err != nil {
		t.Fatalf("pagamento falhou: %v", err)
	}

	if err := solicitacao.Aprovar(uuid.New(), time.Now().UTC()); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("aprovar paga = %v, esperado conflito", err)
	}
}

func TestNovoMotivoEstorno(t *testing.T) {
	for _, texto := range []string{"", "  ", strings.Repeat("x", 501)} {
		if _, err := domainsolicitacao.NovoMotivoEstorno(texto); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("motivo de %d caracteres = %v, esperado erro de validação", len(texto), err)
		}
	}
}
