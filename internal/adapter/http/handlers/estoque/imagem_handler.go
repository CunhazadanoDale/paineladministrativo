package estoque

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	estoquedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/estoque"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
)

type ImagemHandler struct {
	usecase portsin.ImagemUseCase
}

func NewImagemHandler(usecase portsin.ImagemUseCase) *ImagemHandler {
	return &ImagemHandler{usecase: usecase}
}

func (h *ImagemHandler) Anexar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao estoquedto.AnexarImagemRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	imagem, err := h.usecase.Anexar(r.Context(), requisicao.ParaInput(usuario.ID, id))
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusCreated, dto.Resposta[estoquedto.ImagemResponse]{
		Dados: estoquedto.NovaImagemResponse(imagem),
	})
}

func (h *ImagemHandler) Remover(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	produtoID, ok := resposta.ParametroUUID(w, r, "id")
	if !ok {
		return
	}

	imagemID, ok := resposta.ParametroUUID(w, r, "imagemId")
	if !ok {
		return
	}

	if err := h.usecase.Remover(r.Context(), usuario.ID, produtoID, imagemID); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverVazio(w, http.StatusNoContent)
}
