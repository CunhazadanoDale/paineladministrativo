package estoque

import (
	"errors"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

func TestNovaImagemValidaEConstrói(t *testing.T) {
	imagem, err := NovaImagem(ImagemInput{
		ProdutoID: uuid.New(),
		ArquivoID: uuid.New(),
		Ordem:     2,
		Alt:       "  Saco de cimento  ",
	})
	if err != nil {
		t.Fatalf("criação falhou: %v", err)
	}

	if imagem.ID == uuid.Nil {
		t.Error("imagem nasceu sem identificador")
	}
	if imagem.Ordem != 2 {
		t.Errorf("ordem = %d, esperado 2", imagem.Ordem)
	}
	if imagem.Alt != "Saco de cimento" {
		t.Errorf("texto alternativo = %q, esperado sem espaços nas bordas", imagem.Alt)
	}
	if imagem.CriadoEm.IsZero() {
		t.Error("imagem nasceu sem data de criação")
	}
}

func TestNovaImagemValidaCamposObrigatorios(t *testing.T) {
	casos := []struct {
		nome  string
		input ImagemInput
	}{
		{"sem produto", ImagemInput{ArquivoID: uuid.New()}},
		{"sem arquivo", ImagemInput{ProdutoID: uuid.New()}},
		{"ordem negativa", ImagemInput{ProdutoID: uuid.New(), ArquivoID: uuid.New(), Ordem: -1}},
	}

	for _, caso := range casos {
		if _, err := NovaImagem(caso.input); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("%s = %v, esperado erro de validação", caso.nome, err)
		}
	}
}

func TestNovaImagemValidaTextoAlternativoLongo(t *testing.T) {
	_, err := NovaImagem(ImagemInput{
		ProdutoID: uuid.New(),
		ArquivoID: uuid.New(),
		Alt:       strings.Repeat("a", 161),
	})
	if !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("texto alternativo com 161 caracteres = %v, esperado erro de validação", err)
	}

	if _, err := NovaImagem(ImagemInput{
		ProdutoID: uuid.New(),
		ArquivoID: uuid.New(),
		Alt:       strings.Repeat("a", 160),
	}); err != nil {
		t.Errorf("texto alternativo com 160 caracteres = %v, esperado aceito", err)
	}
}
