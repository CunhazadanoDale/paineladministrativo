package estoque

import (
	"time"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	"github.com/google/uuid"
)

type CriarCategoriaRequest struct {
	Nome           string     `json:"nome"`
	CategoriaPaiID *uuid.UUID `json:"categoria_pai_id"`
	Ordem          int        `json:"ordem"`
	Icone          string     `json:"icone"`
}

func (r CriarCategoriaRequest) ParaInput(usuarioID uuid.UUID) portsin.CriarCategoriaInput {
	return portsin.CriarCategoriaInput{
		UsuarioID:      usuarioID,
		Nome:           r.Nome,
		CategoriaPaiID: r.CategoriaPaiID,
		Ordem:          r.Ordem,
		Icone:          r.Icone,
	}
}

type AlterarCategoriaRequest struct {
	Nome  string `json:"nome"`
	Ordem int    `json:"ordem"`
	Icone string `json:"icone"`
}

func (r AlterarCategoriaRequest) ParaInput(usuarioID, categoriaID uuid.UUID) portsin.AlterarCategoriaInput {
	return portsin.AlterarCategoriaInput{
		UsuarioID:   usuarioID,
		CategoriaID: categoriaID,
		Nome:        r.Nome,
		Ordem:       r.Ordem,
		Icone:       r.Icone,
	}
}

type CategoriaResponse struct {
	ID             uuid.UUID  `json:"id"`
	Nome           string     `json:"nome"`
	CategoriaPaiID *uuid.UUID `json:"categoria_pai_id"`
	Slug           string     `json:"slug"`
	Ordem          int        `json:"ordem"`
	Icone          string     `json:"icone"`
	Ativo          bool       `json:"ativo"`
	CriadoEm       time.Time  `json:"criado_em"`
	AtualizadoEm   time.Time  `json:"atualizado_em"`
}

func NovaCategoriaResponse(item *domainestoque.Categoria) CategoriaResponse {
	return CategoriaResponse{
		ID:             item.ID,
		Nome:           item.Nome,
		CategoriaPaiID: item.CategoriaPaiID,
		Slug:           item.Slug.Valor(),
		Ordem:          item.Ordem,
		Icone:          item.Icone,
		Ativo:          item.Ativo,
		CriadoEm:       item.CriadoEm,
		AtualizadoEm:   item.AtualizadoEm,
	}
}

func NovaCategoriaResponses(itens []*domainestoque.Categoria) []CategoriaResponse {
	respostas := make([]CategoriaResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovaCategoriaResponse(item))
	}

	return respostas
}
