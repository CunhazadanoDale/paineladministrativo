package notafiscal

import (
	"fmt"
	"net/http"

	notafiscaldto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/notafiscal"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/middleware"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/notafiscal"
)

type EventoHandler struct {
	usecase portsin.EventoUseCase
}

func NewEventoHandler(usecase portsin.EventoUseCase) *EventoHandler {
	return &EventoHandler{usecase: usecase}
}

func (h *EventoHandler) Importar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao notafiscaldto.ImportarEventoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	importacao, err := h.usecase.Importar(r.Context(), requisicao.ParaInput(usuario.ID))
	if err == nil && importacao.ErroDosTitulos != nil {
		middleware.AnotarErro(w, fmt.Errorf("cancelamento dos títulos da nota fiscal: %w", importacao.ErroDosTitulos))
	}
	escreverItem(w, http.StatusCreated, importacao, err, notafiscaldto.NovaImportacaoEventoResponse)
}

func (h *EventoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	eventos, err := h.usecase.Listar(r.Context(), id, usuario.ID)
	escreverLista(w, eventos, err, notafiscaldto.NovosEventosResponse)
}
