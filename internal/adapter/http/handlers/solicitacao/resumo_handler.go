package solicitacao

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	solicitacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/solicitacao"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
)

type ResumoContasAPagarHandler struct {
	usecase portsin.ResumoContasAPagarUseCase
}

func NewResumoContasAPagarHandler(usecase portsin.ResumoContasAPagarUseCase) *ResumoContasAPagarHandler {
	return &ResumoContasAPagarHandler{usecase: usecase}
}

func (h *ResumoContasAPagarHandler) Consultar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.ConsultarResumoInput{UsuarioID: usuario.ID}
	var err error
	if input.Empresa, err = resposta.ConsultaUUID(r, "empresa"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.De, err = solicitacaodto.DataDeConsulta(resposta.ConsultaTexto(r, "de"), "de"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.Ate, err = solicitacaodto.DataDeConsulta(resposta.ConsultaTexto(r, "ate"), "ate"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	resumo, err := h.usecase.Consultar(r.Context(), input)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[solicitacaodto.ResumoContasAPagarResponse]{
		Dados: solicitacaodto.NovoResumoContasAPagarResponse(resumo),
	})
}
