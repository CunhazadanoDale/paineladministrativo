package solicitacao_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
)

func TestNovoPrazoPagamentoRejeitaDataAnteriorAHoje(t *testing.T) {
	ontem := time.Now().AddDate(0, 0, -1)

	if _, err := domainsolicitacao.NovoPrazoPagamento(ontem); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("prazo ontem = %v, esperado erro de validação", err)
	}
}

func TestNovoPrazoPagamentoRejeitaDataZerada(t *testing.T) {
	if _, err := domainsolicitacao.NovoPrazoPagamento(time.Time{}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("prazo zerado = %v, esperado erro de validação", err)
	}
}

func TestNovoPrazoPagamentoAceitaDataFutura(t *testing.T) {
	hoje := time.Now().AddDate(0, 0, 1)

	prazo, err := domainsolicitacao.NovoPrazoPagamento(hoje)
	if err != nil {
		t.Fatalf("prazo futuro falhou: %v", err)
	}
	if prazo.Data().IsZero() {
		t.Error("prazo aceito devolveu data zerada")
	}
}
