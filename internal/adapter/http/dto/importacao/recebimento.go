package importacao

import (
	"time"

	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/importacao"
	"github.com/google/uuid"
)

type AlocacaoRequest struct {
	ImportacaoItemID uuid.UUID `json:"importacao_item_id"`
	PedidoItemID     uuid.UUID `json:"pedido_item_id"`
	Quantidade       int       `json:"quantidade"`
}

type AlocacoesRequest struct {
	Versao    int               `json:"versao"`
	Alocacoes []AlocacaoRequest `json:"alocacoes"`
}

func (r AlocacoesRequest) ParaAlocacoes() []portsin.AlocacaoInput {
	alocacoes := make([]portsin.AlocacaoInput, 0, len(r.Alocacoes))
	for _, alocacao := range r.Alocacoes {
		alocacoes = append(alocacoes, portsin.AlocacaoInput{
			ImportacaoItemID: alocacao.ImportacaoItemID,
			PedidoItemID:     alocacao.PedidoItemID,
			Quantidade:       alocacao.Quantidade,
		})
	}

	return alocacoes
}

type ItemRecebidoRequest struct {
	ItemID             uuid.UUID `json:"item_id"`
	QuantidadeRecebida int       `json:"quantidade_recebida"`
}

type ReceberRequest struct {
	Versao      int                   `json:"versao"`
	RecebidaEm  string                `json:"recebida_em"`
	LocalID     uuid.UUID             `json:"local_id"`
	CambioMicro *int64                `json:"cambio_micro"`
	Itens       []ItemRecebidoRequest `json:"itens"`
}

func (r ReceberRequest) ParaRecebidas() (map[uuid.UUID]int, error) {
	recebidas := make(map[uuid.UUID]int, len(r.Itens))
	for _, item := range r.Itens {
		if _, repetido := recebidas[item.ItemID]; repetido {
			return nil, erroItemRecebidoRepetido()
		}
		recebidas[item.ItemID] = item.QuantidadeRecebida
	}

	return recebidas, nil
}

type AlocacaoResponse struct {
	ImportacaoItemID uuid.UUID  `json:"importacao_item_id"`
	PedidoItemID     uuid.UUID  `json:"pedido_item_id"`
	Quantidade       int        `json:"quantidade"`
	Reservada        *int       `json:"reservada"`
	ReservaID        *uuid.UUID `json:"reserva_id"`
	UsuarioID        uuid.UUID  `json:"usuario_id"`
	CriadoEm         time.Time  `json:"criado_em"`
	AtualizadoEm     time.Time  `json:"atualizado_em"`
}

func NovasAlocacoesResponse(alocacoes []*domainimportacao.Alocacao) []AlocacaoResponse {
	respostas := make([]AlocacaoResponse, 0, len(alocacoes))
	for _, alocacao := range alocacoes {
		respostas = append(respostas, AlocacaoResponse{
			ImportacaoItemID: alocacao.ImportacaoItemID,
			PedidoItemID:     alocacao.PedidoItemID,
			Quantidade:       alocacao.Quantidade,
			Reservada:        alocacao.Reservada,
			ReservaID:        alocacao.ReservaID,
			UsuarioID:        alocacao.UsuarioID,
			CriadoEm:         alocacao.CriadoEm,
			AtualizadoEm:     alocacao.AtualizadoEm,
		})
	}

	return respostas
}
