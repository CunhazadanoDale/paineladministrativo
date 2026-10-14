package financeiro

import (
	"math"
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const (
	tamanhoMaximoNomeCategoria = 120
	tamanhoMaximoCodigo        = 20
)

type CategoriaFinanceira struct {
	ID             uuid.UUID
	Nome           string
	CategoriaPaiID *uuid.UUID
	Natureza       Natureza
	LinhaDRE       LinhaDRE
	Codigo         *string
	Ordem          int
	Ativa          bool
	Versao         int
	CriadoEm       time.Time
	AtualizadoEm   time.Time
}

type DadosCategoriaFinanceira struct {
	Nome   string
	Codigo *string
	Ordem  int
}

func NovaCategoriaFinanceira(dados DadosCategoriaFinanceira, classificacao *Classificacao, pai *CategoriaFinanceira) (*CategoriaFinanceira, error) {
	agora := time.Now().UTC()
	categoria := &CategoriaFinanceira{
		ID:       uuid.New(),
		Ativa:    true,
		Versao:   versaoInicial,
		CriadoEm: agora,
	}

	if pai == nil {
		if classificacao == nil || !classificacao.LinhaDRE().AceitaNatureza(classificacao.Natureza()) {
			return nil, domain.ErroValidacao("natureza e linha da DRE são obrigatórias na categoria raiz")
		}
		categoria.Natureza = classificacao.Natureza()
		categoria.LinhaDRE = classificacao.LinhaDRE()
	} else {
		if classificacao != nil {
			return nil, domain.ErroValidacao("subcategoria herda natureza e linha da DRE da categoria pai")
		}
		if pai.EhSubcategoria() || !pai.Ativa {
			return nil, ErroPaiDeveSerRaizAtiva()
		}
		paiID := pai.ID
		categoria.CategoriaPaiID = &paiID
		categoria.Natureza = pai.Natureza
		categoria.LinhaDRE = pai.LinhaDRE
	}

	if err := categoria.Alterar(dados, agora); err != nil {
		return nil, err
	}

	return categoria, nil
}

func ErroPaiDeveSerRaizAtiva() error {
	return domain.ErroValidacao("categoria pai deve ser uma categoria raiz ativa")
}

func ErroPaiInativo() error {
	return domain.ErroValidacao("categoria pai está inativa")
}

func ErroSubcategoriasAtivas() error {
	return domain.ErroConflito("categoria possui subcategorias ativas e não pode ser desativada")
}

func ErroCategoriaFinanceiraAlterada() error {
	return domain.ErroConflito("a categoria financeira foi alterada por outra pessoa; recarregue e tente de novo")
}

func (c *CategoriaFinanceira) EhSubcategoria() bool {
	return c != nil && c.CategoriaPaiID != nil
}

func (c *CategoriaFinanceira) Lancavel() bool {
	return c.EhSubcategoria() && c.Ativa
}

func (c *CategoriaFinanceira) ConferirVersao(informada int) error {
	return conferirVersao(informada, c.Versao, "da categoria financeira", ErroCategoriaFinanceiraAlterada())
}

func (c *CategoriaFinanceira) Alterar(dados DadosCategoriaFinanceira, agora time.Time) error {
	nome, err := textoObrigatorio(dados.Nome, "nome da categoria financeira", tamanhoMaximoNomeCategoria)
	if err != nil {
		return err
	}

	codigo, err := codigoContabil(dados.Codigo)
	if err != nil {
		return err
	}

	if dados.Ordem < 0 || dados.Ordem > math.MaxInt32 {
		return domain.ErroValidacao("ordem da categoria financeira deve estar entre 0 e 2147483647")
	}

	c.Nome = nome
	c.Codigo = codigo
	c.Ordem = dados.Ordem
	c.AtualizadoEm = agora

	return nil
}

func (c *CategoriaFinanceira) Ativar(pai *CategoriaFinanceira, agora time.Time) (bool, error) {
	if c.Ativa {
		return false, nil
	}
	if c.EhSubcategoria() {
		if pai == nil || pai.ID != *c.CategoriaPaiID {
			return false, domain.ErroValidacao("categoria pai da subcategoria não informada")
		}
		if !pai.Ativa {
			return false, ErroPaiInativo()
		}
	}

	c.Ativa = true
	c.AtualizadoEm = agora

	return true, nil
}

func (c *CategoriaFinanceira) Desativar(possuiSubcategoriasAtivas bool, agora time.Time) (bool, error) {
	if !c.Ativa {
		return false, nil
	}
	if possuiSubcategoriasAtivas {
		return false, ErroSubcategoriasAtivas()
	}

	c.Ativa = false
	c.AtualizadoEm = agora

	return true, nil
}

func codigoContabil(valor *string) (*string, error) {
	if valor == nil {
		return nil, nil
	}

	codigo := strings.TrimSpace(*valor)
	if codigo == "" {
		return nil, nil
	}
	if len(codigo) > tamanhoMaximoCodigo {
		return nil, domain.ErroValidacao("código da categoria financeira deve ter no máximo 20 caracteres")
	}

	for _, parte := range strings.Split(codigo, ".") {
		if parte == "" || strings.Trim(parte, "0123456789") != "" {
			return nil, domain.ErroValidacao("código da categoria financeira deve ter números separados por ponto, como 4.1.03")
		}
	}

	return &codigo, nil
}
