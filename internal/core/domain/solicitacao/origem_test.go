package solicitacao_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
)

func TestNovaOrigemAceitaSoAsConhecidas(t *testing.T) {
	for _, valor := range []string{"manual", "recorrencia", " MANUAL "} {
		if _, err := domainsolicitacao.NovaOrigem(valor); err != nil {
			t.Errorf("NovaOrigem(%q) falhou: %v", valor, err)
		}
	}
	for _, valor := range []string{"", "pedido", "importacao"} {
		if _, err := domainsolicitacao.NovaOrigem(valor); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("NovaOrigem(%q) = %v, esperado erro de validação", valor, err)
		}
	}
}
