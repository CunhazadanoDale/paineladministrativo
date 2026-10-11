package estoque

import (
	"time"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	"github.com/google/uuid"
)

type ReservarEstoqueRequest struct {
	LocalID    uuid.UUID `json:"local_id"`
	Quantidade int       `json:"quantidade"`
	Observacao string    `json:"observacao"`
}

func (r ReservarEstoqueRequest) ParaInput(usuarioID, produtoID uuid.UUID) portsin.ReservarEstoqueInput {
	return portsin.ReservarEstoqueInput{
		UsuarioID:  usuarioID,
		ProdutoID:  produtoID,
		LocalID:    r.LocalID,
		Quantidade: r.Quantidade,
		Observacao: r.Observacao,
	}
}

type ReservaResponse struct {
	ID          uuid.UUID  `json:"id"`
	ProdutoID   uuid.UUID  `json:"produto_id"`
	LocalID     uuid.UUID  `json:"local_id"`
	Quantidade  int        `json:"quantidade"`
	Origem      string     `json:"origem"`
	OrigemID    *uuid.UUID `json:"origem_id"`
	Status      string     `json:"status"`
	UsuarioID   uuid.UUID  `json:"usuario_id"`
	Observacao  string     `json:"observacao"`
	CriadoEm    time.Time  `json:"criado_em"`
	EncerradaEm *time.Time `json:"encerrada_em"`
}

func NovaReservaResponse(item *domainestoque.Reserva) ReservaResponse {
	return ReservaResponse{
		ID:          item.ID,
		ProdutoID:   item.ProdutoID,
		LocalID:     item.LocalID,
		Quantidade:  item.Quantidade,
		Origem:      item.Origem.String(),
		OrigemID:    item.OrigemID,
		Status:      item.Status.String(),
		UsuarioID:   item.UsuarioID,
		Observacao:  item.Observacao,
		CriadoEm:    item.CriadoEm,
		EncerradaEm: item.EncerradaEm,
	}
}

func NovaReservaResponses(itens []*domainestoque.Reserva) []ReservaResponse {
	respostas := make([]ReservaResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovaReservaResponse(item))
	}

	return respostas
}
