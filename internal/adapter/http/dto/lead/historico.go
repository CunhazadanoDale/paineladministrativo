package lead

import (
	"time"

	domainlead "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	"github.com/google/uuid"
)

type RegistrarMovimentacaoRequest struct {
	EtapaAnteriorID uuid.UUID `json:"etapa_anterior_id"`
	EtapaAtualID    uuid.UUID `json:"etapa_atual_id"`
}

type LeadHistoricoResponse struct {
	ID              uuid.UUID `json:"id"`
	LeadID          uuid.UUID `json:"lead_id"`
	EtapaAnteriorID uuid.UUID `json:"etapa_anterior_id"`
	EtapaAtualID    uuid.UUID `json:"etapa_atual_id"`
	MovidoEm        time.Time `json:"movido_em"`
}

func NovaLeadHistoricoResponse(item domainlead.LeadHistorico) LeadHistoricoResponse {
	return LeadHistoricoResponse{
		ID:              item.ID,
		LeadID:          item.LeadID,
		EtapaAnteriorID: item.EtapaAnteriorID,
		EtapaAtualID:    item.EtapaAtualID,
		MovidoEm:        item.MovidoEm,
	}
}

func NovaLeadHistoricoResponses(itens []domainlead.LeadHistorico) []LeadHistoricoResponse {
	respostas := make([]LeadHistoricoResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, NovaLeadHistoricoResponse(item))
	}

	return respostas
}
