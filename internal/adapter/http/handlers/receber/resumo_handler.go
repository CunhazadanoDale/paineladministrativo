package receber

import (
	"net/http"

	receberdto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/receber"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/receber"
)

type ResumoContasAReceberHandler struct {
	usecase portsin.ResumoContasAReceberUseCase
}

func NewResumoContasAReceberHandler(usecase portsin.ResumoContasAReceberUseCase) *ResumoContasAReceberHandler {
	return &ResumoContasAReceberHandler{usecase: usecase}
}

func (h *ResumoContasAReceberHandler) Consultar(w http.ResponseWriter, r *http.Request) {
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
	if input.De, err = consultaData(r, "de"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.Ate, err = consultaData(r, "ate"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	resumo, err := h.usecase.Consultar(r.Context(), input)
	escreverItem(w, http.StatusOK, resumo, err, receberdto.NovoResumoContasAReceberResponse)
}
