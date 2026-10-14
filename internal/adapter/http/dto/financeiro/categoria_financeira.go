package financeiro

import (
	"time"

	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/financeiro"
	"github.com/google/uuid"
)

type DadosCategoriaFinanceiraRequest struct {
	Nome   string  `json:"nome"`
	Codigo *string `json:"codigo"`
	Ordem  int     `json:"ordem"`
}

func (r DadosCategoriaFinanceiraRequest) ParaDados() portsin.DadosCategoriaFinanceiraInput {
	return portsin.DadosCategoriaFinanceiraInput{
		Nome:   r.Nome,
		Codigo: r.Codigo,
		Ordem:  r.Ordem,
	}
}

type CriarCategoriaFinanceiraRequest struct {
	DadosCategoriaFinanceiraRequest
	CategoriaPaiID *uuid.UUID `json:"categoria_pai_id"`
	Natureza       *string    `json:"natureza"`
	LinhaDRE       *string    `json:"linha_dre"`
}

func (r CriarCategoriaFinanceiraRequest) ParaInput(usuarioID uuid.UUID) portsin.CriarCategoriaFinanceiraInput {
	return portsin.CriarCategoriaFinanceiraInput{
		UsuarioID:      usuarioID,
		CategoriaPaiID: r.CategoriaPaiID,
		Natureza:       r.Natureza,
		LinhaDRE:       r.LinhaDRE,
		Dados:          r.ParaDados(),
	}
}

type AlterarCategoriaFinanceiraRequest struct {
	DadosCategoriaFinanceiraRequest
	Versao int `json:"versao"`
}

type CategoriaFinanceiraResponse struct {
	ID             uuid.UUID  `json:"id"`
	Nome           string     `json:"nome"`
	CategoriaPaiID *uuid.UUID `json:"categoria_pai_id"`
	Natureza       string     `json:"natureza"`
	LinhaDRE       string     `json:"linha_dre"`
	Codigo         *string    `json:"codigo"`
	Ordem          int        `json:"ordem"`
	Ativa          bool       `json:"ativa"`
	Lancavel       bool       `json:"lancavel"`
	Versao         int        `json:"versao"`
	CriadoEm       time.Time  `json:"criado_em"`
	AtualizadoEm   time.Time  `json:"atualizado_em"`
}

func NovaCategoriaFinanceiraResponse(categoria *domainfinanceiro.CategoriaFinanceira) CategoriaFinanceiraResponse {
	return CategoriaFinanceiraResponse{
		ID:             categoria.ID,
		Nome:           categoria.Nome,
		CategoriaPaiID: categoria.CategoriaPaiID,
		Natureza:       categoria.Natureza.String(),
		LinhaDRE:       categoria.LinhaDRE.String(),
		Codigo:         categoria.Codigo,
		Ordem:          categoria.Ordem,
		Ativa:          categoria.Ativa,
		Lancavel:       categoria.Lancavel(),
		Versao:         categoria.Versao,
		CriadoEm:       categoria.CriadoEm,
		AtualizadoEm:   categoria.AtualizadoEm,
	}
}

func NovasCategoriaFinanceiraResponses(categorias []*domainfinanceiro.CategoriaFinanceira) []CategoriaFinanceiraResponse {
	respostas := make([]CategoriaFinanceiraResponse, 0, len(categorias))
	for _, categoria := range categorias {
		respostas = append(respostas, NovaCategoriaFinanceiraResponse(categoria))
	}

	return respostas
}

type RamoDoPlanoResponse struct {
	CategoriaFinanceiraResponse
	Subcategorias []CategoriaFinanceiraResponse `json:"subcategorias"`
}

func NovoPlanoResponse(plano []domainfinanceiro.RamoDoPlano) []RamoDoPlanoResponse {
	respostas := make([]RamoDoPlanoResponse, 0, len(plano))
	for _, ramo := range plano {
		respostas = append(respostas, RamoDoPlanoResponse{
			CategoriaFinanceiraResponse: NovaCategoriaFinanceiraResponse(ramo.Categoria),
			Subcategorias:               NovasCategoriaFinanceiraResponses(ramo.Subcategorias),
		})
	}

	return respostas
}
