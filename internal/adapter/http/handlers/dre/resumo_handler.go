package dre

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	dredto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/dre"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/dre"
)

type ResumoHandler struct {
	usecase portsin.ResumoDREUseCase
}

func NewResumoHandler(usecase portsin.ResumoDREUseCase) *ResumoHandler {
	return &ResumoHandler{usecase: usecase}
}

func (h *ResumoHandler) Consultar(w http.ResponseWriter, r *http.Request) {
	input, ok := consultaDoPeriodo(w, r)
	if !ok {
		return
	}

	resumo, err := h.usecase.Consultar(r.Context(), input)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[dredto.ResumoDREResponse]{
		Dados: dredto.NovoResumoDREResponse(resumo),
	})
}
