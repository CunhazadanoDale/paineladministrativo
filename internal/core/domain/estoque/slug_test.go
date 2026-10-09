package estoque_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
)

func TestNovoSlugGeraFormatoPadrao(t *testing.T) {
	casos := []struct {
		texto    string
		esperado string
	}{
		{"Cimento CP II", "cimento-cp-ii"},
		{"  Argamassa AC-III  ", "argamassa-ac-iii"},
		{"Água Fria Mineral", "agua-fria-mineral"},
		{"Ferro 7/8''", "ferro-7-8"},
		{"Já É!", "ja-e"},
	}

	for _, caso := range casos {
		slug, err := domainestoque.NovoSlug(caso.texto)
		if err != nil {
			t.Errorf("NovoSlug(%q) falhou: %v", caso.texto, err)
			continue
		}
		if slug.Valor() != caso.esperado {
			t.Errorf("NovoSlug(%q) = %q, esperado %q", caso.texto, slug.Valor(), caso.esperado)
		}
	}
}

func TestNovoSlugRejeitaTextoSemCaracteresValidos(t *testing.T) {
	for _, texto := range []string{"", "   ", "!!!", "---"} {
		if _, err := domainestoque.NovoSlug(texto); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("NovoSlug(%q) = %v, esperado erro de validação", texto, err)
		}
	}
}

func TestNovoSlugRejeitaTextoGigante(t *testing.T) {
	if _, err := domainestoque.NovoSlug(strings.Repeat("a", 300)); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("slug longo = %v, esperado erro de validação", err)
	}
}
