package caixa

import (
	"net/http"

	caixadto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/caixa"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
)

type SaldoHandler struct {
	usecase portsin.SaldoUseCase
}

func NewSaldoHandler(usecase portsin.SaldoUseCase) *SaldoHandler {
	return &SaldoHandler{usecase: usecase}
}

func (h *SaldoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	empresa, err := resposta.ConsultaUUID(r, "empresa")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	saldos, err := h.usecase.Listar(r.Context(), usuario.ID, empresa)
	escreverLista(w, saldos, err, caixadto.NovasSaldoContaResponses)
}
