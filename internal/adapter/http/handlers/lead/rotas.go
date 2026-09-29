package lead

import (
	"net/http"

	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/leads"
)

type Handlers struct {
	Leads      *LeadHandler
	Funis      *FunilHandler
	Etapas     *EtapaHandler
	Historicos *LeadHistoryHandler
}

func NovosHandlers(
	leads portsin.LeadUseCase,
	funis portsin.FunilUseCase,
	etapas portsin.EtapaUseCase,
	historicos portsin.LeadHistoryUseCase,
) *Handlers {
	return &Handlers{
		Leads:      NewLeadHandler(leads),
		Funis:      NewFunilHandler(funis),
		Etapas:     NewEtapaHandler(etapas),
		Historicos: NewLeadHistoryHandler(historicos),
	}
}

func (h *Handlers) RegistrarRotas(mux *http.ServeMux) {
	h.Leads.RegistrarRotas(mux)
	h.Funis.RegistrarRotas(mux)
	h.Etapas.RegistrarRotas(mux)
	h.Historicos.RegistrarRotas(mux)
}
