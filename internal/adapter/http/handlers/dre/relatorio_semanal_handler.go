package dre

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	dredto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/dre"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/dre"
)

type RelatorioSemanalHandler struct {
	usecase portsin.RelatorioSemanalUseCase
}

func NewRelatorioSemanalHandler(usecase portsin.RelatorioSemanalUseCase) *RelatorioSemanalHandler {
	return &RelatorioSemanalHandler{usecase: usecase}
}

func (h *RelatorioSemanalHandler) Consultar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.ConsultarSemanaInput{UsuarioID: usuario.ID}
	var err error
	if input.Empresa, err = resposta.ConsultaUUID(r, "empresa"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.Semana, err = dredto.DataDeConsulta(resposta.ConsultaTexto(r, "semana"), "semana"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	relatorio, err := h.usecase.Consultar(r.Context(), input)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[dredto.RelatorioSemanalResponse]{
		Dados: dredto.NovoRelatorioSemanalResponse(relatorio),
	})
}
