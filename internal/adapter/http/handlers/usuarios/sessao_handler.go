package usuarios

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	usuariosdto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/usuarios"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/usuarios"
)

type SessaoHandler struct {
	usecase portsin.UsuarioUseCase
}

func NewSessaoHandler(usecase portsin.UsuarioUseCase) *SessaoHandler {
	return &SessaoHandler{usecase: usecase}
}

func (h *SessaoHandler) Obter(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	item, err := h.usecase.GetByID(r.Context(), usuario.ID)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[usuariosdto.UsuarioResponse]{
		Dados: usuariosdto.NovaUsuarioResponse(item),
	})
}

func (h *SessaoHandler) TrocarSenha(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao usuariosdto.TrocarSenhaRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	if err := h.usecase.UpdateSenha(r.Context(), usuario.ID, requisicao.NovaSenha); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverVazio(w, http.StatusNoContent)
}
