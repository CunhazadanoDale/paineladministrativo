package comissao

import (
	"net/http"

	comissaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/comissao"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/comissao"
)

type PercentualHandler struct {
	usecase portsin.PercentualUseCase
}

func NewPercentualHandler(usecase portsin.PercentualUseCase) *PercentualHandler {
	return &PercentualHandler{usecase: usecase}
}

func (h *PercentualHandler) Obter(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	percentual, err := h.usecase.Obter(r.Context(), id, usuario.ID)
	escreverItem(w, http.StatusOK, percentual, err, comissaodto.NovoPercentualResponse)
}

func (h *PercentualHandler) Definir(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao comissaodto.DefinirPercentualRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	pontosBase, err := requisicao.PontosBase()
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	percentual, err := h.usecase.Definir(r.Context(), portsin.DefinirPercentualInput{
		UsuarioID: usuario.ID, PedidoID: id, PontosBase: pontosBase, Versao: requisicao.Versao,
	})
	escreverItem(w, http.StatusOK, percentual, err, comissaodto.NovoPercentualResponse)
}
