package pedido

import (
	"time"

	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/pedido"
	"github.com/google/uuid"
)

type EntregaItemRequest struct {
	PedidoItemID uuid.UUID `json:"pedido_item_id"`
	LocalID      uuid.UUID `json:"local_id"`
	Quantidade   int       `json:"quantidade"`
}

type CriarEntregaRequest struct {
	Transportadora *string              `json:"transportadora"`
	FreteTipo      string               `json:"frete_tipo"`
	FreteCentavos  *int64               `json:"frete_centavos"`
	Previsao       *string              `json:"previsao"`
	Itens          []EntregaItemRequest `json:"itens"`
}

func (r CriarEntregaRequest) ParaInput(usuarioID, pedidoID uuid.UUID) (portsin.CriarEntregaInput, error) {
	previsao, err := paraDataOpcional(r.Previsao, "previsão")
	if err != nil {
		return portsin.CriarEntregaInput{}, err
	}

	itens := make([]portsin.EntregaItemInput, 0, len(r.Itens))
	for _, item := range r.Itens {
		itens = append(itens, portsin.EntregaItemInput{
			PedidoItemID: item.PedidoItemID,
			LocalID:      item.LocalID,
			Quantidade:   item.Quantidade,
		})
	}

	return portsin.CriarEntregaInput{
		UsuarioID:      usuarioID,
		PedidoID:       pedidoID,
		Transportadora: r.Transportadora,
		FreteTipo:      r.FreteTipo,
		FreteCentavos:  r.FreteCentavos,
		Previsao:       previsao,
		Itens:          itens,
	}, nil
}

type SaidaRequest struct {
	Versao int     `json:"versao"`
	SaiuEm *string `json:"saiu_em"`
}

func (r SaidaRequest) ParaInput(usuarioID, entregaID uuid.UUID) (portsin.TransicaoEntregaInput, error) {
	data, err := paraDataOuZero(r.SaiuEm, "data de saída")

	return portsin.TransicaoEntregaInput{UsuarioID: usuarioID, EntregaID: entregaID, Versao: r.Versao, Data: data}, err
}

type ConclusaoRequest struct {
	Versao     int     `json:"versao"`
	EntregueEm *string `json:"entregue_em"`
}

func (r ConclusaoRequest) ParaInput(usuarioID, entregaID uuid.UUID) (portsin.TransicaoEntregaInput, error) {
	data, err := paraDataOuZero(r.EntregueEm, "data da entrega")

	return portsin.TransicaoEntregaInput{UsuarioID: usuarioID, EntregaID: entregaID, Versao: r.Versao, Data: data}, err
}

type EntregaItemResponse struct {
	PedidoItemID uuid.UUID  `json:"pedido_item_id"`
	LocalID      uuid.UUID  `json:"local_id"`
	Quantidade   int        `json:"quantidade"`
	MovimentoID  *uuid.UUID `json:"movimento_id"`
}

type EntregaResponse struct {
	ID             uuid.UUID             `json:"id"`
	PedidoID       uuid.UUID             `json:"pedido_id"`
	Status         string                `json:"status"`
	Transportadora *string               `json:"transportadora"`
	FreteTipo      string                `json:"frete_tipo"`
	FreteCentavos  *int64                `json:"frete_centavos"`
	Previsao       *time.Time            `json:"previsao"`
	SaiuEm         *time.Time            `json:"saiu_em"`
	EntregueEm     *time.Time            `json:"entregue_em"`
	UsuarioID      uuid.UUID             `json:"usuario_id"`
	Versao         int                   `json:"versao"`
	CriadoEm       time.Time             `json:"criado_em"`
	AtualizadoEm   time.Time             `json:"atualizado_em"`
	Itens          []EntregaItemResponse `json:"itens"`
}

func NovaEntregaResponse(item *domainpedido.Entrega) EntregaResponse {
	resposta := EntregaResponse{
		ID:             item.ID,
		PedidoID:       item.PedidoID,
		Status:         item.Status.String(),
		Transportadora: item.Transportadora,
		FreteTipo:      item.Frete.Tipo().String(),
		FreteCentavos:  item.Frete.Centavos(),
		Previsao:       item.Previsao,
		SaiuEm:         item.SaiuEm,
		EntregueEm:     item.EntregueEm,
		UsuarioID:      item.UsuarioID,
		Versao:         item.Versao,
		CriadoEm:       item.CriadoEm,
		AtualizadoEm:   item.AtualizadoEm,
		Itens:          make([]EntregaItemResponse, 0, len(item.Itens)),
	}
	for _, entregaItem := range item.Itens {
		resposta.Itens = append(resposta.Itens, EntregaItemResponse{
			PedidoItemID: entregaItem.PedidoItemID,
			LocalID:      entregaItem.LocalID,
			Quantidade:   entregaItem.Quantidade,
			MovimentoID:  entregaItem.MovimentoID,
		})
	}

	return resposta
}

func NovaEntregaResponses(itens []*domainpedido.Entrega) []EntregaResponse {
	respostas := make([]EntregaResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovaEntregaResponse(item))
	}

	return respostas
}
