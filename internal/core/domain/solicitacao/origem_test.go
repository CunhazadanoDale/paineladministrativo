package solicitacao_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
)

func TestNovaOrigemAceitaSoAsConhecidas(t *testing.T) {
	for _, valor := range []string{"manual", "recorrencia", " MANUAL ", "recompra", "repasse_fomento", "comissao", "importacao_parcela", "importacao_custo", "divida_amortizacao", "divida_juros"} {
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

func TestOrigemSistemica(t *testing.T) {
	if !domainsolicitacao.OrigemRecompra.Sistemica() || !domainsolicitacao.OrigemRepasseFomento.Sistemica() ||
		!domainsolicitacao.OrigemComissao.Sistemica() || !domainsolicitacao.OrigemImportacaoParcela.Sistemica() ||
		!domainsolicitacao.OrigemImportacaoCusto.Sistemica() || !domainsolicitacao.OrigemDividaAmortizacao.Sistemica() ||
		!domainsolicitacao.OrigemDividaJuros.Sistemica() {
		t.Error("recompra, repasse, comissão, importação e dívida são origens sistêmicas")
	}
	if domainsolicitacao.OrigemManual.Sistemica() || domainsolicitacao.OrigemRecorrencia.Sistemica() {
		t.Error("manual e recorrência têm geração própria")
	}
}
