package estoque

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	estoquedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/estoque"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
)

type ReservaHandler struct {
	usecase portsin.ReservaUseCase
}

func NewReservaHandler(usecase portsin.ReservaUseCase) *ReservaHandler {
	return &ReservaHandler{usecase: usecase}
}

func (h *ReservaHandler) Reservar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao estoquedto.ReservarEstoqueRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	reserva, err := h.usecase.Reservar(r.Context(), requisicao.ParaInput(usuario.ID, id))
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusCreated, dto.Resposta[estoquedto.ReservaResponse]{
		Dados: estoquedto.NovaReservaResponse(reserva),
	})
}

func (h *ReservaHandler) Liberar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	reserva, err := h.usecase.Liberar(r.Context(), id, usuario.ID)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[estoquedto.ReservaResponse]{
		Dados: estoquedto.NovaReservaResponse(reserva),
	})
}

func (h *ReservaHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	local, err := resposta.ConsultaUUID(r, "local")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	paginacao := resposta.ConsultaPaginacao(r)

	itens, err := h.usecase.Listar(r.Context(), portsin.ListarReservasInput{
		UsuarioID: usuario.ID,
		ProdutoID: id,
		LocalID:   local,
		Status:    resposta.ConsultaTexto(r, "status"),
		Filtro:    paginacao,
	})
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[estoquedto.ReservaResponse]{
		Dados:   estoquedto.NovaReservaResponses(itens),
		Pagina:  paginacao.Page,
		Tamanho: paginacao.Size,
	})
}
