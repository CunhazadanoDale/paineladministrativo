package solicitacao_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
)

func TestNovaObservacaoValida(t *testing.T) {
	observacao, err := domainsolicitacao.NovaObservacao("  material de alvenaria  ")
	if err != nil {
		t.Fatalf("observação válida falhou: %v", err)
	}
	if observacao.Texto() != "material de alvenaria" {
		t.Errorf("texto = %q, esperado sem espaços nas bordas", observacao.Texto())
	}
}

func TestNovaObservacaoRejeitaVazia(t *testing.T) {
	if _, err := domainsolicitacao.NovaObservacao("   "); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("observação vazia = %v, esperado erro de validação", err)
	}
}

func TestNovaObservacaoRejeitaAcimaDeMil(t *testing.T) {
	if _, err := domainsolicitacao.NovaObservacao(strings.Repeat("a", 1001)); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("observação longa = %v, esperado erro de validação", err)
	}
}

func TestNovoMotivoRejeicaoValida(t *testing.T) {
	motivo, err := domainsolicitacao.NovoMotivoRejeicao("  fora do orçamento  ")
	if err != nil {
		t.Fatalf("motivo válido falhou: %v", err)
	}
	if motivo.Texto() != "fora do orçamento" {
		t.Errorf("texto = %q", motivo.Texto())
	}
}

func TestNovoMotivoRejeicaoRejeitaInvalidos(t *testing.T) {
	casos := []string{"", "   ", strings.Repeat("x", 501)}

	for _, caso := range casos {
		if _, err := domainsolicitacao.NovoMotivoRejeicao(caso); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("motivo de %d caracteres = %v, esperado erro de validação", len(caso), err)
		}
	}
}
