package estoque

import (
	"time"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	"github.com/google/uuid"
)

type CriarProdutoRequest struct {
	CategoriaID              uuid.UUID `json:"categoria_id"`
	Nome                     string    `json:"nome"`
	Descricao                string    `json:"descricao"`
	Codigo                   *string   `json:"codigo"`
	UnidadeMedida            string    `json:"unidade_medida"`
	PrecoCentavos            *int64    `json:"preco_centavos"`
	PrecoPromocionalCentavos *int64    `json:"preco_promocional_centavos"`
	EstoqueMinimo            *int      `json:"estoque_minimo"`
	PesoKg                   *float64  `json:"peso_kg"`
	Destaque                 bool      `json:"destaque"`
}

func (r CriarProdutoRequest) ParaInput(usuarioID uuid.UUID) portsin.CriarProdutoInput {
	return portsin.CriarProdutoInput{
		UsuarioID:                usuarioID,
		CategoriaID:              r.CategoriaID,
		Nome:                     r.Nome,
		Descricao:                r.Descricao,
		Codigo:                   r.Codigo,
		UnidadeMedida:            r.UnidadeMedida,
		PrecoCentavos:            r.PrecoCentavos,
		PrecoPromocionalCentavos: r.PrecoPromocionalCentavos,
		EstoqueMinimo:            r.EstoqueMinimo,
		PesoKg:                   r.PesoKg,
		Destaque:                 r.Destaque,
	}
}

type AlterarProdutoRequest struct {
	CategoriaID              uuid.UUID `json:"categoria_id"`
	Nome                     string    `json:"nome"`
	Descricao                string    `json:"descricao"`
	Codigo                   *string   `json:"codigo"`
	UnidadeMedida            string    `json:"unidade_medida"`
	PrecoCentavos            *int64    `json:"preco_centavos"`
	PrecoPromocionalCentavos *int64    `json:"preco_promocional_centavos"`
	EstoqueMinimo            *int      `json:"estoque_minimo"`
	PesoKg                   *float64  `json:"peso_kg"`
	Destaque                 bool      `json:"destaque"`
}

func (r AlterarProdutoRequest) ParaInput(usuarioID, produtoID uuid.UUID) portsin.AlterarProdutoInput {
	return portsin.AlterarProdutoInput{
		UsuarioID:                usuarioID,
		ProdutoID:                produtoID,
		CategoriaID:              r.CategoriaID,
		Nome:                     r.Nome,
		Descricao:                r.Descricao,
		Codigo:                   r.Codigo,
		UnidadeMedida:            r.UnidadeMedida,
		PrecoCentavos:            r.PrecoCentavos,
		PrecoPromocionalCentavos: r.PrecoPromocionalCentavos,
		EstoqueMinimo:            r.EstoqueMinimo,
		PesoKg:                   r.PesoKg,
		Destaque:                 r.Destaque,
	}
}

type AlternarDestaqueRequest struct {
	Destaque bool `json:"destaque"`
}

type ProdutoResponse struct {
	ID                       uuid.UUID        `json:"id"`
	CategoriaID              uuid.UUID        `json:"categoria_id"`
	Nome                     string           `json:"nome"`
	Slug                     string           `json:"slug"`
	Descricao                string           `json:"descricao"`
	Codigo                   *string          `json:"codigo"`
	UnidadeMedida            string           `json:"unidade_medida"`
	PrecoCentavos            *int64           `json:"preco_centavos"`
	PrecoPromocionalCentavos *int64           `json:"preco_promocional_centavos"`
	Saldo                    int              `json:"saldo"`
	EstoqueMinimo            *int             `json:"estoque_minimo"`
	EstoqueBaixo             bool             `json:"estoque_baixo"`
	PesoKg                   *float64         `json:"peso_kg"`
	Destaque                 bool             `json:"destaque"`
	Ativo                    bool             `json:"ativo"`
	CriadoEm                 time.Time        `json:"criado_em"`
	AtualizadoEm             time.Time        `json:"atualizado_em"`
	Imagens                  []ImagemResponse `json:"imagens"`
}

func NovaProdutoResponse(item *domainestoque.Produto) ProdutoResponse {
	resposta := ProdutoResponse{
		ID:            item.ID,
		CategoriaID:   item.CategoriaID,
		Nome:          item.Nome,
		Slug:          item.Slug.Valor(),
		Descricao:     item.Descricao,
		Codigo:        item.Codigo,
		UnidadeMedida: item.UnidadeMedida.String(),
		Saldo:         item.Saldo.Quantidade(),
		EstoqueBaixo:  item.EstoqueBaixo(),
		Destaque:      item.Destaque,
		Ativo:         item.Ativo,
		CriadoEm:      item.CriadoEm,
		AtualizadoEm:  item.AtualizadoEm,
		Imagens:       NovaImagemResponses(item.Imagens),
	}

	if item.Preco != nil {
		centavos := item.Preco.Centavos()
		resposta.PrecoCentavos = &centavos
	}
	if item.PrecoPromocional != nil {
		centavos := item.PrecoPromocional.Centavos()
		resposta.PrecoPromocionalCentavos = &centavos
	}
	if item.EstoqueMinimo != nil {
		minimo := item.EstoqueMinimo.Quantidade()
		resposta.EstoqueMinimo = &minimo
	}
	if item.Peso != nil {
		quilos := item.Peso.Quilos()
		resposta.PesoKg = &quilos
	}

	return resposta
}

func NovaProdutoResponses(itens []*domainestoque.Produto) []ProdutoResponse {
	respostas := make([]ProdutoResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovaProdutoResponse(item))
	}

	return respostas
}
