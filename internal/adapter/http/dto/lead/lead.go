package lead

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainlead "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	"github.com/google/uuid"
)

type CriarLeadRequest struct {
	Nome     string    `json:"nome"`
	Email    string    `json:"email"`
	Telefone string    `json:"telefone"`
	Origem   string    `json:"origem"`
	EtapaID  uuid.UUID `json:"etapa_id"`
}

type AtualizarLeadRequest struct {
	Nome     string    `json:"nome"`
	Email    string    `json:"email"`
	Telefone string    `json:"telefone"`
	Origem   string    `json:"origem"`
	Ativo    *bool     `json:"ativo"`
	EtapaID  uuid.UUID `json:"etapa_id"`
}

type MoverLeadRequest struct {
	EtapaID uuid.UUID `json:"etapa_id"`
}

type LeadResponse struct {
	ID           uuid.UUID `json:"id"`
	Nome         string    `json:"nome"`
	Email        string    `json:"email"`
	Telefone     string    `json:"telefone"`
	Ativo        bool      `json:"ativo"`
	Origem       string    `json:"origem"`
	CriadoEm     time.Time `json:"criado_em"`
	AtualizadoEm time.Time `json:"atualizado_em"`
	EtapaID      uuid.UUID `json:"etapa_id"`
}

type ContagemResponse struct {
	Total int `json:"total"`
}

func (r CriarLeadRequest) ParaLead() *domainlead.Lead {
	return &domainlead.Lead{
		Nome:     r.Nome,
		Email:    r.Email,
		Telefone: r.Telefone,
		Origem:   r.Origem,
		EtapaID:  r.EtapaID,
	}
}

func (r AtualizarLeadRequest) ParaLead(id uuid.UUID) (*domainlead.Lead, error) {
	if r.Ativo == nil {
		return nil, domain.ErroValidacao("campo ativo é obrigatório")
	}

	return &domainlead.Lead{
		ID:       id,
		Nome:     r.Nome,
		Email:    r.Email,
		Telefone: r.Telefone,
		Origem:   r.Origem,
		Ativo:    *r.Ativo,
		EtapaID:  r.EtapaID,
	}, nil
}

func NovaLeadResponse(item *domainlead.Lead) LeadResponse {
	return LeadResponse{
		ID:           item.ID,
		Nome:         item.Nome,
		Email:        item.Email,
		Telefone:     item.Telefone,
		Ativo:        item.Ativo,
		Origem:       item.Origem,
		CriadoEm:     item.CriadoEm,
		AtualizadoEm: item.AtualizadoEm,
		EtapaID:      item.EtapaID,
	}
}

func NovaLeadResponses(itens []*domainlead.Lead) []LeadResponse {
	respostas := make([]LeadResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovaLeadResponse(item))
	}

	return respostas
}
