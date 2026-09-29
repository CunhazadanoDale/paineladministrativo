package lead

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	leaddto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/lead"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainlead "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/leads"
	"github.com/google/uuid"
)

type EtapaHandler struct {
	usecase portsin.EtapaUseCase
}

func NewEtapaHandler(usecase portsin.EtapaUseCase) *EtapaHandler {
	return &EtapaHandler{usecase: usecase}
}

func (h *EtapaHandler) RegistrarRotas(mux *http.ServeMux) {
	mux.HandleFunc("POST /etapas", h.Criar)
	mux.HandleFunc("GET /funils/{funil_id}/etapas", h.ListarPorFunil)
	mux.HandleFunc("PUT /funils/{funil_id}/etapas/ordem", h.Reordenar)
	mux.HandleFunc("GET /etapas/{etapa_id}", h.Obter)
	mux.HandleFunc("PUT /etapas/{etapa_id}", h.Atualizar)
	mux.HandleFunc("DELETE /etapas/{etapa_id}", h.Remover)
	mux.HandleFunc("GET /etapas/{etapa_id}/proxima", h.Proxima)
	mux.HandleFunc("GET /etapas/{etapa_id}/anterior", h.Anterior)
}

func (h *EtapaHandler) Criar(w http.ResponseWriter, r *http.Request) {
	var requisicao leaddto.CriarEtapaRequest
	if !corpoJSON(w, r, &requisicao) {
		return
	}

	id, err := h.usecase.Create(r.Context(), requisicao.ParaEtapa())
	if err != nil {
		responderErro(w, err)
		return
	}

	item, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		responderErro(w, err)
		return
	}

	responderJSON(w, http.StatusCreated, dto.Resposta[leaddto.EtapaResponse]{
		Dados: leaddto.NovaEtapaResponse(item),
	})
}

func (h *EtapaHandler) ListarPorFunil(w http.ResponseWriter, r *http.Request) {
	funilID, ok := parametroUUID(w, r, "funil_id")
	if !ok {
		return
	}

	itens, err := h.usecase.ListByFunilOrdenado(r.Context(), funilID)
	if err != nil {
		responderErro(w, err)
		return
	}

	responderJSON(w, http.StatusOK, dto.Resposta[[]leaddto.EtapaResponse]{
		Dados: leaddto.NovaEtapaResponses(itens),
	})
}

func (h *EtapaHandler) Obter(w http.ResponseWriter, r *http.Request) {
	etapaID, ok := parametroUUID(w, r, "etapa_id")
	if !ok {
		return
	}

	item, err := h.usecase.GetByID(r.Context(), etapaID)
	if err != nil {
		responderErro(w, err)
		return
	}

	responderJSON(w, http.StatusOK, dto.Resposta[leaddto.EtapaResponse]{
		Dados: leaddto.NovaEtapaResponse(item),
	})
}

func (h *EtapaHandler) Atualizar(w http.ResponseWriter, r *http.Request) {
	etapaID, ok := parametroUUID(w, r, "etapa_id")
	if !ok {
		return
	}

	var requisicao leaddto.AtualizarEtapaRequest
	if !corpoJSON(w, r, &requisicao) {
		return
	}

	if err := h.usecase.Update(r.Context(), requisicao.ParaEtapa(etapaID)); err != nil {
		responderErro(w, err)
		return
	}

	item, err := h.usecase.GetByID(r.Context(), etapaID)
	if err != nil {
		responderErro(w, err)
		return
	}

	responderJSON(w, http.StatusOK, dto.Resposta[leaddto.EtapaResponse]{
		Dados: leaddto.NovaEtapaResponse(item),
	})
}

func (h *EtapaHandler) Remover(w http.ResponseWriter, r *http.Request) {
	etapaID, ok := parametroUUID(w, r, "etapa_id")
	if !ok {
		return
	}

	if err := h.usecase.Delete(r.Context(), etapaID); err != nil {
		responderErro(w, err)
		return
	}

	responderVazio(w, http.StatusNoContent)
}

func (h *EtapaHandler) Reordenar(w http.ResponseWriter, r *http.Request) {
	funilID, ok := parametroUUID(w, r, "funil_id")
	if !ok {
		return
	}

	var requisicao leaddto.ReordenarEtapasRequest
	if !corpoJSON(w, r, &requisicao) {
		return
	}

	if len(requisicao.Etapas) == 0 {
		responderErro(w, domain.ErroValidacao("nenhuma etapa informada para reordenar"))
		return
	}

	etapas := make([]*domainlead.Etapa, 0, len(requisicao.Etapas))
	consultadas := make(map[uuid.UUID]bool, len(requisicao.Etapas))

	for _, etapaID := range requisicao.Etapas {
		if etapaID == uuid.Nil {
			responderErro(w, domain.ErroValidacao("etapa inválida na reordenação"))
			return
		}
		if consultadas[etapaID] {
			responderErro(w, domain.ErroValidacao("etapa repetida na reordenação"))
			return
		}
		consultadas[etapaID] = true

		etapa, err := h.usecase.GetByID(r.Context(), etapaID)
		if err != nil {
			responderErro(w, err)
			return
		}

		etapas = append(etapas, etapa)
	}

	if err := h.usecase.Reordenar(r.Context(), funilID, etapas); err != nil {
		responderErro(w, err)
		return
	}

	ordenadas, err := h.usecase.ListByFunilOrdenado(r.Context(), funilID)
	if err != nil {
		responderErro(w, err)
		return
	}

	responderJSON(w, http.StatusOK, dto.Resposta[[]leaddto.EtapaResponse]{
		Dados: leaddto.NovaEtapaResponses(ordenadas),
	})
}

func (h *EtapaHandler) Proxima(w http.ResponseWriter, r *http.Request) {
	h.seguinte(w, r, true)
}

func (h *EtapaHandler) Anterior(w http.ResponseWriter, r *http.Request) {
	h.seguinte(w, r, false)
}

func (h *EtapaHandler) seguinte(w http.ResponseWriter, r *http.Request, proxima bool) {
	etapaID, ok := parametroUUID(w, r, "etapa_id")
	if !ok {
		return
	}

	var (
		item *domainlead.Etapa
		err  error
	)

	if proxima {
		item, err = h.usecase.GetNextEtapa(r.Context(), etapaID)
	} else {
		item, err = h.usecase.GetPreviousEtapa(r.Context(), etapaID)
	}

	if err != nil {
		responderErro(w, err)
		return
	}
	if item == nil {
		if proxima {
			responderErro(w, domain.ErroNaoEncontrado("não há etapa seguinte"))
		} else {
			responderErro(w, domain.ErroNaoEncontrado("não há etapa anterior"))
		}
		return
	}

	responderJSON(w, http.StatusOK, dto.Resposta[leaddto.EtapaResponse]{
		Dados: leaddto.NovaEtapaResponse(item),
	})
}
