package estoque

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	estoquedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/estoque"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
)

type ResumoHandler struct {
	usecase portsin.ResumoUseCase
}

func NewResumoHandler(usecase portsin.ResumoUseCase) *ResumoHandler {
	return &ResumoHandler{usecase: usecase}
}

func (h *ResumoHandler) Consultar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	empresa, err := resposta.ConsultaUUID(r, "empresa")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	resumo, err := h.usecase.Resumo(r.Context(), portsin.ResumoInput{UsuarioID: usuario.ID, Empresa: empresa})
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[estoquedto.ResumoResponse]{
		Dados: estoquedto.NovoResumoResponse(resumo),
	})
}
