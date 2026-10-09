package estoque

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const tamanhoMaximoAltImagem = 160

type Imagem struct {
	ID        uuid.UUID
	ProdutoID uuid.UUID
	ArquivoID uuid.UUID
	Ordem     int
	Alt       string
	CriadoEm  time.Time
}

type ImagemInput struct {
	ProdutoID uuid.UUID
	ArquivoID uuid.UUID
	Ordem     int
	Alt       string
}

func NovaImagem(input ImagemInput) (*Imagem, error) {
	if err := validarImagemInput(input); err != nil {
		return nil, err
	}

	return &Imagem{
		ID:        uuid.New(),
		ProdutoID: input.ProdutoID,
		ArquivoID: input.ArquivoID,
		Ordem:     input.Ordem,
		Alt:       strings.TrimSpace(input.Alt),
		CriadoEm:  time.Now().UTC(),
	}, nil
}

func validarImagemInput(input ImagemInput) error {
	if input.ProdutoID == uuid.Nil {
		return domain.ErroValidacao("produto da imagem é obrigatório")
	}
	if input.ArquivoID == uuid.Nil {
		return domain.ErroValidacao("arquivo da imagem é obrigatório")
	}
	if input.Ordem < 0 {
		return domain.ErroValidacao("ordem da imagem não pode ser negativa")
	}
	if utf8.RuneCountInString(strings.TrimSpace(input.Alt)) > tamanhoMaximoAltImagem {
		return domain.ErroValidacao("texto alternativo da imagem deve ter no máximo 160 caracteres")
	}

	return nil
}
