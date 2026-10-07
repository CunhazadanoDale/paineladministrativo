package solicitacao_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
)

func TestNovaFormaPagamentoAceitaFormasValidas(t *testing.T) {
	for _, valor := range []string{"pix", "cartao", "boleto", "  PIX  "} {
		forma, err := domainsolicitacao.NovaFormaPagamento(valor)
		if err != nil {
			t.Errorf("NovaFormaPagamento(%q) falhou: %v", valor, err)
			continue
		}
		if err := forma.Validado(); err != nil {
			t.Errorf("Validado() de %q falhou: %v", valor, err)
		}
	}
}

func TestNovaFormaPagamentoRejeitaFormasInvalidas(t *testing.T) {
	for _, valor := range []string{"", "dinheiro", "transferencia"} {
		if _, err := domainsolicitacao.NovaFormaPagamento(valor); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("NovaFormaPagamento(%q) = %v, esperado erro de validação", valor, err)
		}
	}
}
