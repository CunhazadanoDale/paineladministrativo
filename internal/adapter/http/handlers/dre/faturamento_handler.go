package dre

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	dredto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/dre"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/dre"
)

type FaturamentoHandler struct {
	usecase portsin.FaturamentoUseCase
}

func NewFaturamentoHandler(usecase portsin.FaturamentoUseCase) *FaturamentoHandler {
	return &FaturamentoHandler{usecase: usecase}
}

func (h *FaturamentoHandler) Consultar(w http.ResponseWriter, r *http.Request) {
	input, ok := consultaDoPeriodo(w, r)
	if !ok {
		return
	}

	resumo, err := h.usecase.Consultar(r.Context(), input)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[dredto.ResumoFaturamentoResponse]{
		Dados: dredto.NovoResumoFaturamentoResponse(resumo),
	})
}
