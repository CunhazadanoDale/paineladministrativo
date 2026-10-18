package receber

import (
	"net/http"

	receberdto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/receber"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/receber"
)

type EntradaDoPedidoHandler struct {
	usecase portsin.EntradaDoPedidoUseCase
}

func NewEntradaDoPedidoHandler(usecase portsin.EntradaDoPedidoUseCase) *EntradaDoPedidoHandler {
	return &EntradaDoPedidoHandler{usecase: usecase}
}

func (h *EntradaDoPedidoHandler) GerarTitulo(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao receberdto.TituloDeEntradaRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	vencimento, err := receberdto.ParaData(requisicao.Vencimento, "vencimento")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	titulo, criado, err := h.usecase.GerarTituloDeEntrada(r.Context(), portsin.GerarTituloDeEntradaInput{
		UsuarioID: usuario.ID, PedidoID: id, Vencimento: vencimento, ContaBancariaID: requisicao.ContaBancariaID,
	})
	status := http.StatusOK
	if criado {
		status = http.StatusCreated
	}
	escreverItem(w, status, titulo, err, receberdto.NovoTituloResponse)
}
