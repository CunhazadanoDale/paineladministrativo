package estoque

import (
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type CategoriaPublicaResponse struct {
	ID     uuid.UUID                  `json:"id"`
	Nome   string                     `json:"nome"`
	Slug   string                     `json:"slug"`
	Ordem  int                        `json:"ordem"`
	Icone  string                     `json:"icone"`
	Filhas []CategoriaPublicaResponse `json:"filhas"`
}

func NovaCategoriaPublicaResponses(itens []*domainestoque.Categoria) []CategoriaPublicaResponse {
	porID := make(map[uuid.UUID]int, len(itens))
	respostas := make([]CategoriaPublicaResponse, 0, len(itens))
	raizes := make([]int, 0, len(itens))

	for indice, item := range itens {
		porID[item.ID] = indice
		respostas = append(respostas, CategoriaPublicaResponse{
			ID:     item.ID,
			Nome:   item.Nome,
			Slug:   item.Slug.Valor(),
			Ordem:  item.Ordem,
			Icone:  item.Icone,
			Filhas: make([]CategoriaPublicaResponse, 0),
		})
	}

	for indice, item := range itens {
		if item.CategoriaPaiID == nil {
			raizes = append(raizes, indice)
			continue
		}

		pai, ok := porID[*item.CategoriaPaiID]
		if !ok {
			raizes = append(raizes, indice)
			continue
		}

		respostas[pai].Filhas = append(respostas[pai].Filhas, respostas[indice])
	}

	arvore := make([]CategoriaPublicaResponse, 0, len(raizes))
	for _, indice := range raizes {
		arvore = append(arvore, respostas[indice])
	}

	return arvore
}

type ImagemPublicaResponse struct {
	ID    uuid.UUID `json:"id"`
	Ordem int       `json:"ordem"`
	Alt   string    `json:"alt"`
}

func NovaImagemPublicaResponses(itens []*domainestoque.Imagem) []ImagemPublicaResponse {
	respostas := make([]ImagemPublicaResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, ImagemPublicaResponse{
			ID:    item.ID,
			Ordem: item.Ordem,
			Alt:   item.Alt,
		})
	}

	return respostas
}

type ProdutoPublicoResponse struct {
	ID                       uuid.UUID               `json:"id"`
	CategoriaID              uuid.UUID               `json:"categoria_id"`
	Nome                     string                  `json:"nome"`
	Slug                     string                  `json:"slug"`
	Descricao                string                  `json:"descricao"`
	UnidadeMedida            string                  `json:"unidade_medida"`
	PrecoCentavos            *int64                  `json:"preco_centavos"`
	PrecoPromocionalCentavos *int64                  `json:"preco_promocional_centavos"`
	Destaque                 bool                    `json:"destaque"`
	Imagens                  []ImagemPublicaResponse `json:"imagens"`
}

func NovaProdutoPublicoResponse(item *domainestoque.Produto) ProdutoPublicoResponse {
	resposta := ProdutoPublicoResponse{
		ID:            item.ID,
		CategoriaID:   item.CategoriaID,
		Nome:          item.Nome,
		Slug:          item.Slug.Valor(),
		Descricao:     item.Descricao,
		UnidadeMedida: item.UnidadeMedida.String(),
		Destaque:      item.Destaque,
		Imagens:       NovaImagemPublicaResponses(item.Imagens),
	}

	if item.Preco != nil {
		centavos := item.Preco.Centavos()
		resposta.PrecoCentavos = &centavos
	}
	if item.PrecoPromocional != nil {
		centavos := item.PrecoPromocional.Centavos()
		resposta.PrecoPromocionalCentavos = &centavos
	}

	return resposta
}

func NovaProdutoPublicoResponses(itens []*domainestoque.Produto) []ProdutoPublicoResponse {
	respostas := make([]ProdutoPublicoResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovaProdutoPublicoResponse(item))
	}

	return respostas
}
