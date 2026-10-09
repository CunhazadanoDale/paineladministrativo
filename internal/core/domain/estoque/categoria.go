package estoque

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const (
	tamanhoMaximoNomeCategoria  = 120
	tamanhoMaximoIconeCategoria = 60
)

type Categoria struct {
	ID             uuid.UUID
	Nome           string
	CategoriaPaiID *uuid.UUID
	Slug           Slug
	Ordem          int
	Icone          string
	Ativo          bool
	CriadoEm       time.Time
	AtualizadoEm   time.Time
}

func NovaCategoria(nome string, slug Slug, categoriaPai *Categoria, ordem int, icone string) (*Categoria, error) {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return nil, domain.ErroValidacao("nome da categoria é obrigatório")
	}
	if utf8.RuneCountInString(nome) > tamanhoMaximoNomeCategoria {
		return nil, domain.ErroValidacao("nome da categoria deve ter no máximo 120 caracteres")
	}
	if slug.Vazio() {
		return nil, domain.ErroValidacao("slug da categoria é obrigatório")
	}
	if ordem < 0 {
		return nil, domain.ErroValidacao("ordem da categoria não pode ser negativa")
	}
	if utf8.RuneCountInString(icone) > tamanhoMaximoIconeCategoria {
		return nil, domain.ErroValidacao("icone da categoria deve ter no máximo 60 caracteres")
	}

	var categoriaPaiID *uuid.UUID
	if categoriaPai != nil {
		if categoriaPai.CategoriaPaiID != nil {
			return nil, domain.ErroValidacao("categoria não pode ter mais de dois níveis")
		}
		if !categoriaPai.Ativo {
			return nil, domain.ErroValidacao("categoria pai deve estar ativa")
		}
		categoriaPaiID = &categoriaPai.ID
	}

	agora := time.Now().UTC()

	return &Categoria{
		ID:             uuid.New(),
		Nome:           nome,
		CategoriaPaiID: categoriaPaiID,
		Slug:           slug,
		Ordem:          ordem,
		Icone:          strings.TrimSpace(icone),
		Ativo:          true,
		CriadoEm:       agora,
		AtualizadoEm:   agora,
	}, nil
}

func (c *Categoria) EhSubcategoria() bool {
	return c != nil && c.CategoriaPaiID != nil
}

func (c *Categoria) AlterarNome(nome string, agora time.Time) error {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return domain.ErroValidacao("nome da categoria é obrigatório")
	}
	if utf8.RuneCountInString(nome) > tamanhoMaximoNomeCategoria {
		return domain.ErroValidacao("nome da categoria deve ter no máximo 120 caracteres")
	}

	c.Nome = nome
	c.AtualizadoEm = agora

	return nil
}

func (c *Categoria) AlterarDados(nome string, ordem int, icone string, agora time.Time) error {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return domain.ErroValidacao("nome da categoria é obrigatório")
	}
	if utf8.RuneCountInString(nome) > tamanhoMaximoNomeCategoria {
		return domain.ErroValidacao("nome da categoria deve ter no máximo 120 caracteres")
	}
	if ordem < 0 {
		return domain.ErroValidacao("ordem da categoria não pode ser negativa")
	}
	if utf8.RuneCountInString(icone) > tamanhoMaximoIconeCategoria {
		return domain.ErroValidacao("icone da categoria deve ter no máximo 60 caracteres")
	}

	c.Nome = nome
	c.Ordem = ordem
	c.Icone = strings.TrimSpace(icone)
	c.AtualizadoEm = agora

	return nil
}

func (c *Categoria) AlternarAtivo(ativo bool, possuiSubcategorias bool, agora time.Time) error {
	if !ativo && possuiSubcategorias {
		return domain.ErroConflito("categoria possui subcategorias e não pode ser inativada")
	}

	c.Ativo = ativo
	c.AtualizadoEm = agora

	return nil
}
