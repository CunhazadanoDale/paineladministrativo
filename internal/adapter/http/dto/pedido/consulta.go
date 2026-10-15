package pedido

import (
	"time"

	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	"github.com/google/uuid"
)

type PendenciaResponse struct {
	PedidoID            uuid.UUID  `json:"pedido_id"`
	Numero              int        `json:"numero"`
	EmpresaID           uuid.UUID  `json:"empresa_id"`
	ClienteID           uuid.UUID  `json:"cliente_id"`
	RepresentanteID     *uuid.UUID `json:"representante_id"`
	Status              string     `json:"status"`
	PrazoLimite         *time.Time `json:"prazo_limite"`
	PedidoItemID        uuid.UUID  `json:"pedido_item_id"`
	ProdutoID           uuid.UUID  `json:"produto_id"`
	ProdutoNome         string     `json:"produto_nome"`
	Vendido             int        `json:"vendido"`
	Entregue            int        `json:"entregue"`
	Restante            int        `json:"restante"`
	Programado          int        `json:"programado"`
	Reservado           int        `json:"reservado"`
	FaltaCobrir         int        `json:"falta_cobrir"`
	DisponivelNaEmpresa int        `json:"disponivel_na_empresa"`
	PodeCobrirAgora     int        `json:"pode_cobrir_agora"`
}

type PedidoNoPrazoResponse struct {
	PedidoResponse
	DiasParaOPrazo *int `json:"dias_para_o_prazo"`
}

func NovaPendenciaResponses(itens []*domainpedido.Pendencia) []PendenciaResponse {
	respostas := make([]PendenciaResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, PendenciaResponse{
			PedidoID:            item.PedidoID,
			Numero:              item.Numero,
			EmpresaID:           item.EmpresaID,
			ClienteID:           item.ClienteID,
			RepresentanteID:     item.RepresentanteID,
			Status:              item.Status.String(),
			PrazoLimite:         item.PrazoLimite,
			PedidoItemID:        item.ItemID,
			ProdutoID:           item.ProdutoID,
			ProdutoNome:         item.ProdutoNome,
			Vendido:             item.Vendido,
			Entregue:            item.Entregue,
			Restante:            item.Restante(),
			Programado:          item.Programado,
			Reservado:           item.Reservado,
			FaltaCobrir:         item.FaltaCobrir(),
			DisponivelNaEmpresa: item.DisponivelNaEmpresa,
			PodeCobrirAgora:     item.PodeCobrirAgora(),
		})
	}

	return respostas
}

func NovoPedidoNoPrazoResponses(itens []*domainpedido.Pedido, agora time.Time) []PedidoNoPrazoResponse {
	respostas := make([]PedidoNoPrazoResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, PedidoNoPrazoResponse{
			PedidoResponse: NovoPedidoResponse(item),
			DiasParaOPrazo: item.DiasParaOPrazo(agora),
		})
	}

	return respostas
}
