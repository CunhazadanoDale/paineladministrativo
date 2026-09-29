package lead

import (
	domainlead "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	"github.com/google/uuid"
)

type CriarFunilRequest struct {
	Nome string `json:"nome"`
}

type AtualizarFunilRequest struct {
	Nome  string `json:"nome"`
	Ativo bool   `json:"ativo"`
}

type FunilResponse struct {
	FunilID uuid.UUID `json:"funil_id"`
	Nome    string    `json:"nome"`
	Ativo   bool      `json:"ativo"`
}

type CriarEtapaRequest struct {
	Nome    string    `json:"nome"`
	Ordem   int       `json:"ordem"`
	FunilID uuid.UUID `json:"funil_id"`
}

type AtualizarEtapaRequest struct {
	Nome    string    `json:"nome"`
	Ordem   int       `json:"ordem"`
	FunilID uuid.UUID `json:"funil_id"`
	Ativo   bool      `json:"ativo"`
}

type EtapaResponse struct {
	EtapaID uuid.UUID `json:"etapa_id"`
	Nome    string    `json:"nome"`
	Ordem   int       `json:"ordem"`
	FunilID uuid.UUID `json:"funil_id"`
	Ativo   bool      `json:"ativo"`
}

type ReordenarEtapasRequest struct {
	Etapas []uuid.UUID `json:"etapas"`
}

func (r CriarFunilRequest) ParaFunil() *domainlead.Funil {
	return &domainlead.Funil{Nome: r.Nome}
}

func (r AtualizarFunilRequest) ParaFunil(funilID uuid.UUID) *domainlead.Funil {
	return &domainlead.Funil{FunilID: funilID, Nome: r.Nome, Ativo: r.Ativo}
}

func NovaFunilResponse(item *domainlead.Funil) FunilResponse {
	return FunilResponse{FunilID: item.FunilID, Nome: item.Nome, Ativo: item.Ativo}
}

func NovaFunilResponses(itens []domainlead.Funil) []FunilResponse {
	respostas := make([]FunilResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovaFunilResponse(&item))
	}

	return respostas
}

func (r CriarEtapaRequest) ParaEtapa() *domainlead.Etapa {
	return &domainlead.Etapa{Nome: r.Nome, Ordem: r.Ordem, FunilID: r.FunilID}
}

func (r AtualizarEtapaRequest) ParaEtapa(etapaID uuid.UUID) *domainlead.Etapa {
	return &domainlead.Etapa{
		EtapaID: etapaID,
		Nome:    r.Nome,
		Ordem:   r.Ordem,
		FunilID: r.FunilID,
		Ativo:   r.Ativo,
	}
}

func NovaEtapaResponse(item *domainlead.Etapa) EtapaResponse {
	return EtapaResponse{
		EtapaID: item.EtapaID,
		Nome:    item.Nome,
		Ordem:   item.Ordem,
		FunilID: item.FunilID,
		Ativo:   item.Ativo,
	}
}

func NovaEtapaResponses(itens []*domainlead.Etapa) []EtapaResponse {
	respostas := make([]EtapaResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovaEtapaResponse(item))
	}

	return respostas
}
