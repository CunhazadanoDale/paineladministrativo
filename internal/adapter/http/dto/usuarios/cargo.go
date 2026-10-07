package usuarios

import (
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	"github.com/google/uuid"
)

type CriarCargoRequest struct {
	Nome      string `json:"nome"`
	Descricao string `json:"descricao"`
}

type AtualizarCargoRequest struct {
	Nome      string `json:"nome"`
	Descricao string `json:"descricao"`
	Ativo     bool   `json:"ativo"`
}

type CargoResponse struct {
	ID        uuid.UUID `json:"id"`
	Nome      string    `json:"nome"`
	Descricao string    `json:"descricao"`
	Ativo     bool      `json:"ativo"`
}

func (r AtualizarCargoRequest) ParaCargo(id uuid.UUID) *domainusuarios.Cargo {
	return &domainusuarios.Cargo{
		ID:        id,
		Nome:      r.Nome,
		Descricao: r.Descricao,
		Ativo:     r.Ativo,
	}
}

func NovaCargoResponse(item *domainusuarios.Cargo) CargoResponse {
	return CargoResponse{
		ID:        item.ID,
		Nome:      item.Nome,
		Descricao: item.Descricao,
		Ativo:     item.Ativo,
	}
}

func NovaCargoResponses(itens []*domainusuarios.Cargo) []CargoResponse {
	respostas := make([]CargoResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovaCargoResponse(item))
	}

	return respostas
}
