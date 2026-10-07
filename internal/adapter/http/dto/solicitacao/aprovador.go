package solicitacao

import (
	"time"

	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

type DesignarAprovadorRequest struct {
	UsuarioID uuid.UUID `json:"usuario_id"`
}

type AprovadorResponse struct {
	ID        uuid.UUID `json:"id"`
	UsuarioID uuid.UUID `json:"usuario_id"`
	CriadoEm  time.Time `json:"criado_em"`
}

func NovoAprovadorResponse(item *domainsolicitacao.Aprovador) AprovadorResponse {
	return AprovadorResponse{
		ID:        item.ID,
		UsuarioID: item.UsuarioID,
		CriadoEm:  item.CriadoEm,
	}
}

func NovoAprovadorResponses(itens []*domainsolicitacao.Aprovador) []AprovadorResponse {
	respostas := make([]AprovadorResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovoAprovadorResponse(item))
	}

	return respostas
}
