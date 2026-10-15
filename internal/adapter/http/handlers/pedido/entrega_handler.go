package pedido

import (
	"net/http"

	pedidodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/pedido"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/pedido"
)

type EntregaHandler struct {
	usecase portsin.EntregaUseCase
}

func NewEntregaHandler(usecase portsin.EntregaUseCase) *EntregaHandler {
	return &EntregaHandler{usecase: usecase}
}

func (h *EntregaHandler) Criar(w http.ResponseWriter, r *http.Request) {
	usuario, pedidoID, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao pedidodto.CriarEntregaRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	input, err := requisicao.ParaInput(usuario.ID, pedidoID)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	entrega, err := h.usecase.Criar(r.Context(), input)
	escreverItem(w, http.StatusCreated, entrega, err, pedidodto.NovaEntregaResponse)
}

func (h *EntregaHandler) ListarDoPedido(w http.ResponseWriter, r *http.Request) {
	usuario, pedidoID, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	entregas, err := h.usecase.ListarDoPedido(r.Context(), pedidoID, usuario.ID)
	escreverLista(w, entregas, err, pedidodto.NovaEntregaResponses)
}

func (h *EntregaHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.ListarEntregasInput{
		UsuarioID: usuario.ID,
		Status:    resposta.ConsultaTexto(r, "status"),
		Filtro:    resposta.ConsultaPaginacao(r),
	}
	var err error
	if input.Empresa, err = resposta.ConsultaUUID(r, "empresa"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if previsao := resposta.ConsultaTexto(r, "previsao_ate"); previsao != "" {
		data, err := pedidodto.ParaData(previsao, "previsao_ate")
		if err != nil {
			resposta.ResponderErro(w, err)
			return
		}
		input.PrevisaoAte = &data
	}

	entregas, err := h.usecase.Listar(r.Context(), input)
	escreverPagina(w, input.Filtro, entregas, err, pedidodto.NovaEntregaResponses)
}

func (h *EntregaHandler) Obter(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	entrega, err := h.usecase.Obter(r.Context(), id, usuario.ID)
	escreverItem(w, http.StatusOK, entrega, err, pedidodto.NovaEntregaResponse)
}

func (h *EntregaHandler) RegistrarSaida(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao pedidodto.SaidaRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	input, err := requisicao.ParaInput(usuario.ID, id)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	entrega, err := h.usecase.RegistrarSaida(r.Context(), input)
	escreverItem(w, http.StatusOK, entrega, err, pedidodto.NovaEntregaResponse)
}

func (h *EntregaHandler) Concluir(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao pedidodto.ConclusaoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	input, err := requisicao.ParaInput(usuario.ID, id)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	entrega, err := h.usecase.Concluir(r.Context(), input)
	escreverItem(w, http.StatusOK, entrega, err, pedidodto.NovaEntregaResponse)
}

func (h *EntregaHandler) Cancelar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao pedidodto.VersaoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	entrega, err := h.usecase.Cancelar(r.Context(), portsin.TransicaoEntregaInput{UsuarioID: usuario.ID, EntregaID: id, Versao: requisicao.Versao})
	escreverItem(w, http.StatusOK, entrega, err, pedidodto.NovaEntregaResponse)
}
