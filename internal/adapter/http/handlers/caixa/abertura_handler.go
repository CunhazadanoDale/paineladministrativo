package caixa

import (
	"net/http"

	caixadto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/caixa"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
)

type AberturaHandler struct {
	usecase portsin.AberturaUseCase
}

func NewAberturaHandler(usecase portsin.AberturaUseCase) *AberturaHandler {
	return &AberturaHandler{usecase: usecase}
}

func (h *AberturaHandler) Definir(w http.ResponseWriter, r *http.Request) {
	usuario, contaID, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao caixadto.DefinirAberturaRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	input, err := requisicao.ParaInput(usuario.ID, contaID)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	abertura, err := h.usecase.Definir(r.Context(), input)
	escreverItem(w, http.StatusOK, abertura, err, caixadto.NovaAberturaResponse)
}

func (h *AberturaHandler) Obter(w http.ResponseWriter, r *http.Request) {
	usuario, contaID, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	abertura, err := h.usecase.Obter(r.Context(), contaID, usuario.ID)
	escreverItem(w, http.StatusOK, abertura, err, caixadto.NovaAberturaResponse)
}
