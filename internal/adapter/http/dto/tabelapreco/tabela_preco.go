package tabelapreco

import (
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaintabelapreco "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/tabelapreco"
	"github.com/google/uuid"
)

const formatoData = "2006-01-02"

type CriarItemRequest struct {
	ProdutoID     uuid.UUID `json:"produto_id"`
	Modelo        string    `json:"modelo"`
	PrecoCentavos int64     `json:"preco_centavos"`
	VigenteDe     string    `json:"vigente_de"`
	Observacao    string    `json:"observacao"`
}

func (r CriarItemRequest) ParaDados() (domaintabelapreco.DadosItem, error) {
	vigenteDe, err := ParaData(r.VigenteDe, "vigente_de")
	if err != nil {
		return domaintabelapreco.DadosItem{}, err
	}

	return domaintabelapreco.DadosItem{
		ProdutoID:     r.ProdutoID,
		Modelo:        r.Modelo,
		PrecoCentavos: r.PrecoCentavos,
		VigenteDe:     vigenteDe,
		Observacao:    r.Observacao,
	}, nil
}

func ParaData(valor, campo string) (time.Time, error) {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return time.Time{}, nil
	}
	if data, err := time.Parse(formatoData, valor); err == nil {
		return data, nil
	}

	return time.Time{}, domain.ErroValidacao(campo + " deve estar no formato AAAA-MM-DD")
}

type ItemResponse struct {
	ID            uuid.UUID `json:"id"`
	ProdutoID     uuid.UUID `json:"produto_id"`
	Modelo        string    `json:"modelo"`
	PrecoCentavos int64     `json:"preco_centavos"`
	VigenteDe     string    `json:"vigente_de"`
	Observacao    string    `json:"observacao"`
	UsuarioID     uuid.UUID `json:"usuario_id"`
	CriadoEm      time.Time `json:"criado_em"`
}

type PrecoVigenteResponse struct {
	ProdutoID     uuid.UUID  `json:"produto_id"`
	ProdutoNome   string     `json:"produto_nome"`
	ProdutoCodigo *string    `json:"produto_codigo"`
	Unidade       string     `json:"unidade_medida"`
	Modelo        string     `json:"modelo"`
	Data          string     `json:"data"`
	PrecoCentavos *int64     `json:"preco_centavos"`
	Origem        string     `json:"origem"`
	ItemID        *uuid.UUID `json:"item_id"`
	VigenteDe     *string    `json:"vigente_de"`
}

func NovoItemResponse(item *domaintabelapreco.Item) ItemResponse {
	return ItemResponse{
		ID:            item.ID,
		ProdutoID:     item.ProdutoID,
		Modelo:        item.Modelo.String(),
		PrecoCentavos: item.PrecoCentavos,
		VigenteDe:     item.VigenteDe.Format(formatoData),
		Observacao:    item.Observacao,
		UsuarioID:     item.UsuarioID,
		CriadoEm:      item.CriadoEm,
	}
}

func NovoItemResponses(itens []*domaintabelapreco.Item) []ItemResponse {
	respostas := make([]ItemResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovoItemResponse(item))
	}

	return respostas
}

func NovoPrecoVigenteResponses(itens []*domaintabelapreco.PrecoVigente) []PrecoVigenteResponse {
	respostas := make([]PrecoVigenteResponse, 0, len(itens))
	for _, item := range itens {
		resposta := PrecoVigenteResponse{
			ProdutoID:     item.ProdutoID,
			ProdutoNome:   item.ProdutoNome,
			ProdutoCodigo: item.ProdutoCodigo,
			Unidade:       item.Unidade,
			Modelo:        item.Modelo.String(),
			Data:          item.Data.Format(formatoData),
			PrecoCentavos: item.PrecoCentavos,
			Origem:        item.Origem.String(),
			ItemID:        item.ItemID,
		}
		if item.VigenteDe != nil {
			vigenteDe := item.VigenteDe.Format(formatoData)
			resposta.VigenteDe = &vigenteDe
		}
		respostas = append(respostas, resposta)
	}

	return respostas
}
