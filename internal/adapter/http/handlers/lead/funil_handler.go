package lead

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	leaddto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/lead"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	domainlead "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/leads"
)

type FunilHandler struct {
	usecase portsin.FunilUseCase
}

func NewFunilHandler(usecase portsin.FunilUseCase) *FunilHandler {
	return &FunilHandler{usecase: usecase}
}

func (h *FunilHandler) Criar(w http.ResponseWriter, r *http.Request) {
	var requisicao leaddto.CriarFunilRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	id, err := h.usecase.Create(r.Context(), requisicao.ParaFunil())
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	item, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusCreated, dto.Resposta[leaddto.FunilResponse]{
		Dados: leaddto.NovaFunilResponse(item),
	})
}

func (h *FunilHandler) Listar(w http.ResponseWriter, r *http.Request) {
	paginacao := resposta.ConsultaPaginacao(r)

	var (
		itens []domainlead.Funil
		err   error
	)

	if resposta.ConsultaBooleana(r, "ativos") {
		itens, err = h.usecase.ListAtivos(r.Context(), paginacao)
	} else {
		itens, err = h.usecase.List(r.Context(), paginacao)
	}

	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[leaddto.FunilResponse]{
		Dados:   leaddto.NovaFunilResponses(itens),
		Pagina:  paginacao.Page,
		Tamanho: paginacao.Size,
	})
}

func (h *FunilHandler) Obter(w http.ResponseWriter, r *http.Request) {
	funilID, ok := resposta.ParametroUUID(w, r, "funil_id")
	if !ok {
		return
	}

	item, err := h.usecase.GetByID(r.Context(), funilID)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[leaddto.FunilResponse]{
		Dados: leaddto.NovaFunilResponse(item),
	})
}

func (h *FunilHandler) Atualizar(w http.ResponseWriter, r *http.Request) {
	funilID, ok := resposta.ParametroUUID(w, r, "funil_id")
	if !ok {
		return
	}

	var requisicao leaddto.AtualizarFunilRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	if err := h.usecase.Update(r.Context(), requisicao.ParaFunil(funilID)); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	item, err := h.usecase.GetByID(r.Context(), funilID)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[leaddto.FunilResponse]{
		Dados: leaddto.NovaFunilResponse(item),
	})
}

func (h *FunilHandler) Remover(w http.ResponseWriter, r *http.Request) {
	funilID, ok := resposta.ParametroUUID(w, r, "funil_id")
	if !ok {
		return
	}

	if err := h.usecase.Delete(r.Context(), funilID); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverVazio(w, http.StatusNoContent)
}
