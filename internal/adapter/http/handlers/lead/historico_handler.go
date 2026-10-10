package lead

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	leaddto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/lead"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/leads"
)

type LeadHistoricoHandler struct {
	usecase portsin.LeadHistoricoUseCase
}

func NewLeadHistoricoHandler(usecase portsin.LeadHistoricoUseCase) *LeadHistoricoHandler {
	return &LeadHistoricoHandler{usecase: usecase}
}

func (h *LeadHistoricoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	leadID, ok := resposta.ParametroUUID(w, r, "lead_id")
	if !ok {
		return
	}

	paginacao := resposta.ConsultaPaginacao(r)

	itens, err := h.usecase.ListarPorLead(r.Context(), leadID, paginacao)
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
