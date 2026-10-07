package solicitacao_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
)

func TestNovoValorRejeitaValoresInvalidos(t *testing.T) {
	for _, centavos := range []int64{0, -1, -100} {
		if _, err := domainsolicitacao.NovoValor(centavos); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("NovoValor(%d) = %v, esperado erro de validação", centavos, err)
		}
	}
}

func TestNovoValorAceitaAcimaDeZero(t *testing.T) {
	valor, err := domainsolicitacao.NovoValor(150000)
	if err != nil {
		t.Fatalf("NovoValor(150000) falhou: %v", err)
	}
	if valor.Centavos() != 150000 {
		t.Errorf("centavos = %d, esperado 150000", valor.Centavos())
	}
}
