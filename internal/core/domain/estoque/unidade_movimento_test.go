package estoque_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
)

func TestNovaUnidadeMedidaAceitaUnidadesValidas(t *testing.T) {
	for _, valor := range []string{"un", "M2", "  kg ", "pct"} {
		unidade, err := domainestoque.NovaUnidadeMedida(valor)
		if err != nil {
			t.Errorf("NovaUnidadeMedida(%q) falhou: %v", valor, err)
			continue
		}
		if unidade.String() == "" {
			t.Errorf("NovaUnidadeMedida(%q) retornou vazio", valor)
		}
	}
}

func TestNovaUnidadeMedidaRejeitaValorInvalido(t *testing.T) {
	if _, err := domainestoque.NovaUnidadeMedida("pacote"); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("unidade inválida = %v, esperado erro de validação", err)
	}
}

func TestNovoTipoMovimentoValidaEfeitos(t *testing.T) {
	casos := []struct {
		tipo       domainestoque.TipoMovimento
		quantidade int
		efeito     int
	}{
		{domainestoque.TipoMovimentoEntrada, 10, 10},
		{domainestoque.TipoMovimentoDevolucao, 4, 4},
		{domainestoque.TipoMovimentoSaida, 7, -7},
		{domainestoque.TipoMovimentoPerda, 2, -2},
		{domainestoque.TipoMovimentoAjuste, -3, -3},
	}

	for _, caso := range casos {
		quantidade := domainestoque.QuantidadeDe(caso.quantidade)
		if efeito := caso.tipo.Efeito(quantidade); efeito != caso.efeito {
			t.Errorf("Efeito(%s, %d) = %d, esperado %d", caso.tipo, caso.quantidade, efeito, caso.efeito)
		}
	}
}

func TestNovoTipoMovimentoRejeitaValorInvalido(t *testing.T) {
	if _, err := domainestoque.NovoTipoMovimento("transferencia"); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("tipo inválido = %v, esperado erro de validação", err)
	}
}
