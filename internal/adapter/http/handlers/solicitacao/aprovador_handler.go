package solicitacao

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	solicitacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
)

type AprovadorHandler struct {
	usecase portsin.AprovadorUseCase
}

func NewAprovadorHandler(usecase portsin.AprovadorUseCase) *AprovadorHandler {
	return &AprovadorHandler{usecase: usecase}
}

func (h *AprovadorHandler) Designar(w http.ResponseWriter, r *http.Request) {
	var requisicao solicitacaodto.DesignarAprovadorRequest
	if !corpoJSON(w, r, &requisicao) {
		return
	}

	id, err := h.usecase.Designar(r.Context(), requisicao.UsuarioID)
	if err != nil {
		responderErro(w, err)
		return
	}

	item, err := h.usecase.Obter(r.Context(), id)
	if err != nil {
		responderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusCreated, dto.Resposta[solicitacaodto.AprovadorResponse]{
		Dados: solicitacaodto.NovoAprovadorResponse(item),
	})
}

func (h *AprovadorHandler) Listar(w http.ResponseWriter, r *http.Request) {
	paginacao := consultaPaginacao(r)

	itens, err := h.usecase.Listar(r.Context(), paginacao)
	if err != nil {
		responderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[solicitacaodto.AprovadorResponse]{
		Dados:   solicitacaodto.NovoAprovadorResponses(itens),
		Pagina:  paginacao.Page,
		Tamanho: paginacao.Size,
	})
}

func (h *AprovadorHandler) Remover(w http.ResponseWriter, r *http.Request) {
	id, ok := parametroUUID(w, r, "id")
	if !ok {
		return
	}

	if err := h.usecase.Remover(r.Context(), id); err != nil {
		responderErro(w, err)
		return
	}

	dto.EscreverVazio(w, http.StatusNoContent)
}
