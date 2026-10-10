package usuarios

import (
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	"github.com/google/uuid"
)

type CriarCargoRequest struct {
	Nome          string `json:"nome"`
	Descricao     string `json:"descricao"`
	Administrador bool   `json:"administrador"`
	Financeiro    bool   `json:"financeiro"`
	Comercial     bool   `json:"comercial"`
}

type AtualizarCargoRequest struct {
	Nome          string `json:"nome"`
	Descricao     string `json:"descricao"`
	Ativo         bool   `json:"ativo"`
	Administrador bool   `json:"administrador"`
	Financeiro    bool   `json:"financeiro"`
	Comercial     bool   `json:"comercial"`
}

type CargoResponse struct {
	ID            uuid.UUID `json:"id"`
	Nome          string    `json:"nome"`
	Descricao     string    `json:"descricao"`
	Ativo         bool      `json:"ativo"`
	Administrador bool      `json:"administrador"`
	Financeiro    bool      `json:"financeiro"`
	Comercial     bool      `json:"comercial"`
}

func (r CriarCargoRequest) ParaCargo() *domainusuarios.Cargo {
	return &domainusuarios.Cargo{
		Nome:          r.Nome,
		Descricao:     r.Descricao,
		Administrador: r.Administrador,
		Financeiro:    r.Financeiro,
		Comercial:     r.Comercial,
	}
}

func (r AtualizarCargoRequest) ParaCargo(id uuid.UUID) *domainusuarios.Cargo {
	return &domainusuarios.Cargo{
		ID:            id,
		Nome:          r.Nome,
		Descricao:     r.Descricao,
		Ativo:         r.Ativo,
		Administrador: r.Administrador,
		Financeiro:    r.Financeiro,
		Comercial:     r.Comercial,
	}
}

func NovaCargoResponse(item *domainusuarios.Cargo) CargoResponse {
	return CargoResponse{
		ID:            item.ID,
		Nome:          item.Nome,
		Descricao:     item.Descricao,
		Ativo:         item.Ativo,
		Administrador: item.Administrador,
		Financeiro:    item.Financeiro,
		Comercial:     item.Comercial,
	}
}

func NovaCargoResponses(itens []*domainusuarios.Cargo) []CargoResponse {
	respostas := make([]CargoResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovaCargoResponse(item))
	}

	return respostas
}
