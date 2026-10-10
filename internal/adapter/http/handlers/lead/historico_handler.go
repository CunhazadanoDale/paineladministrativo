package lead

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	leaddto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/lead"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/leads"
)

type LeadHistoryHandler struct {
	usecase portsin.LeadHistoryUseCase
}

func NewLeadHistoryHandler(usecase portsin.LeadHistoryUseCase) *LeadHistoryHandler {
	return &LeadHistoryHandler{usecase: usecase}
}

func (h *LeadHistoryHandler) Listar(w http.ResponseWriter, r *http.Request) {
	leadID, ok := resposta.ParametroUUID(w, r, "lead_id")
	if !ok {
		return
	}

	paginacao := resposta.ConsultaPaginacao(r)

	itens, err := h.usecase.ListByLead(r.Context(), leadID, paginacao)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[leaddto.LeadHistoricoResponse]{
		Dados:   leaddto.NovaLeadHistoricoResponses(itens),
		Pagina:  paginacao.Page,
		Tamanho: paginacao.Size,
	})
}
