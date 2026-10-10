package lead

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	leaddto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/lead"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	domainlead "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/leads"
)

type LeadHandler struct {
	usecase portsin.LeadUseCase
}

func NewLeadHandler(usecase portsin.LeadUseCase) *LeadHandler {
	return &LeadHandler{usecase: usecase}
}

func (h *LeadHandler) Criar(w http.ResponseWriter, r *http.Request) {
	var requisicao leaddto.CriarLeadRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	id, err := h.usecase.Criar(r.Context(), requisicao.ParaLead())
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	item, err := h.usecase.Obter(r.Context(), id)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusCreated, dto.Resposta[leaddto.LeadResponse]{
		Dados: leaddto.NovaLeadResponse(item),
	})
}

func (h *LeadHandler) Listar(w http.ResponseWriter, r *http.Request) {
	paginacao := resposta.ConsultaPaginacao(r)
	busca := resposta.ConsultaTexto(r, "q")

	var (
		itens []*domainlead.Lead
		err   error
	)

	if busca != "" {
		itens, err = h.usecase.Buscar(r.Context(), busca, paginacao)
	} else {
		itens, err = h.usecase.ListarAtivos(r.Context(), paginacao)
	}

	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[leaddto.LeadResponse]{
		Dados:   leaddto.NovaLeadResponses(itens),
		Pagina:  paginacao.Page,
		Tamanho: paginacao.Size,
	})
}

func (h *LeadHandler) Obter(w http.ResponseWriter, r *http.Request) {
	id, ok := resposta.ParametroUUID(w, r, "id")
	if !ok {
		return
	}

	item, err := h.usecase.Obter(r.Context(), id)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[leaddto.LeadResponse]{
		Dados: leaddto.NovaLeadResponse(item),
	})
}

func (h *LeadHandler) Atualizar(w http.ResponseWriter, r *http.Request) {
	id, ok := resposta.ParametroUUID(w, r, "id")
	if !ok {
		return
	}

	var requisicao leaddto.AtualizarLeadRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	lead, err := requisicao.ParaLead(id)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	if err := h.usecase.Atualizar(r.Context(), lead); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	item, err := h.usecase.Obter(r.Context(), id)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[leaddto.LeadResponse]{
		Dados: leaddto.NovaLeadResponse(item),
	})
}

func (h *LeadHandler) Remover(w http.ResponseWriter, r *http.Request) {
	id, ok := resposta.ParametroUUID(w, r, "id")
	if !ok {
		return
	}

	if err := h.usecase.Remover(r.Context(), id); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverVazio(w, http.StatusNoContent)
}

func (h *LeadHandler) MoverEtapa(w http.ResponseWriter, r *http.Request) {
	id, ok := resposta.ParametroUUID(w, r, "id")
	if !ok {
		return
	}

	var requisicao leaddto.MoverLeadRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	if err := h.usecase.MoverEtapa(r.Context(), id, requisicao.EtapaID); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	item, err := h.usecase.Obter(r.Context(), id)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[leaddto.LeadResponse]{
		Dados: leaddto.NovaLeadResponse(item),
	})
}

func (h *LeadHandler) ListarPorFunil(w http.ResponseWriter, r *http.Request) {
	funilID, ok := resposta.ParametroUUID(w, r, "funil_id")
	if !ok {
		return
	}

	itens, err := h.usecase.ListarPorFunil(r.Context(), funilID)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[[]leaddto.LeadResponse]{
		Dados: leaddto.NovaLeadResponses(itens),
	})
}

func (h *LeadHandler) ListarPorEtapa(w http.ResponseWriter, r *http.Request) {
	etapaID, ok := resposta.ParametroUUID(w, r, "etapa_id")
	if !ok {
		return
	}

	itens, err := h.usecase.ListarPorEtapa(r.Context(), etapaID)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[[]leaddto.LeadResponse]{
		Dados: leaddto.NovaLeadResponses(itens),
	})
}

func (h *LeadHandler) ContarPorFunil(w http.ResponseWriter, r *http.Request) {
	funilID, ok := resposta.ParametroUUID(w, r, "funil_id")
	if !ok {
		return
	}

	total, err := h.usecase.ContarPorFunil(r.Context(), funilID)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[leaddto.ContagemResponse]{
		Dados: leaddto.ContagemResponse{Total: total},
	})
}

func (h *LeadHandler) ContarPorEtapa(w http.ResponseWriter, r *http.Request) {
	etapaID, ok := resposta.ParametroUUID(w, r, "etapa_id")
	if !ok {
		return
	}

	total, err := h.usecase.ContarPorEtapa(r.Context(), etapaID)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[leaddto.ContagemResponse]{
		Dados: leaddto.ContagemResponse{Total: total},
	})
}
