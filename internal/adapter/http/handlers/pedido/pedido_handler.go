package pedido

import (
	"context"
	"net/http"

	pedidodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/pedido"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/pedido"
)

type PedidoHandler struct {
	usecase portsin.PedidoUseCase
}

func NewPedidoHandler(usecase portsin.PedidoUseCase) *PedidoHandler {
	return &PedidoHandler{usecase: usecase}
}

func (h *PedidoHandler) Criar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao pedidodto.CriarPedidoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	dados, err := requisicao.ParaDados()
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	pedido, err := h.usecase.Criar(r.Context(), portsin.CriarPedidoInput{UsuarioID: usuario.ID, EmpresaID: requisicao.EmpresaID, Dados: dados})
	escreverItem(w, http.StatusCreated, pedido, err, pedidodto.NovoPedidoResponse)
}

func (h *PedidoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.ListarPedidosInput{
		UsuarioID: usuario.ID,
		Status:    resposta.ConsultaTexto(r, "status"),
		Modelo:    resposta.ConsultaTexto(r, "modelo"),
		Filtro:    resposta.ConsultaPaginacao(r),
	}
	var err error
	if input.Empresa, err = resposta.ConsultaUUID(r, "empresa"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.ClienteID, err = resposta.ConsultaUUID(r, "cliente"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.RepresentanteID, err = resposta.ConsultaUUID(r, "representante"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.Numero, err = consultaInteiro(r, "numero"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	pedidos, err := h.usecase.Listar(r.Context(), input)
	escreverPagina(w, input.Filtro, pedidos, err, pedidodto.NovoPedidoResponses)
}

func (h *PedidoHandler) Obter(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	pedido, reservas, err := h.usecase.ObterComReservas(r.Context(), id, usuario.ID)
	escreverItem(w, http.StatusOK, pedido, err, func(item *domainpedido.Pedido) pedidodto.PedidoDetalheResponse {
		return pedidodto.NovoPedidoDetalheResponse(item, reservas)
	})
}

func (h *PedidoHandler) Alterar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao pedidodto.AlterarPedidoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	dados, err := requisicao.ParaDados()
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	pedido, err := h.usecase.Alterar(r.Context(), portsin.AlterarPedidoInput{
		UsuarioID: usuario.ID, PedidoID: id, Versao: requisicao.Versao, Dados: dados,
	})
	escreverItem(w, http.StatusOK, pedido, err, pedidodto.NovoPedidoResponse)
}

func (h *PedidoHandler) Confirmar(w http.ResponseWriter, r *http.Request) {
	h.transicionar(w, r, h.usecase.Confirmar)
}

func (h *PedidoHandler) ReservarPendencias(w http.ResponseWriter, r *http.Request) {
	h.transicionar(w, r, h.usecase.ReservarPendencias)
}

func (h *PedidoHandler) RegistrarEntrada(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao pedidodto.EntradaRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	pagoEm, err := pedidodto.ParaData(requisicao.PagoEm, "data do pagamento da entrada")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	pedido, err := h.usecase.RegistrarEntrada(r.Context(), portsin.RegistrarEntradaInput{
		UsuarioID: usuario.ID, PedidoID: id, Versao: requisicao.Versao, PagaEm: pagoEm, ValorCentavos: requisicao.ValorCentavos,
	})
	escreverItem(w, http.StatusOK, pedido, err, pedidodto.NovoPedidoResponse)
}

func (h *PedidoHandler) Cancelar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao pedidodto.CancelarPedidoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	pedido, err := h.usecase.Cancelar(r.Context(), portsin.CancelarPedidoInput{
		UsuarioID:              usuario.ID,
		PedidoID:               id,
		Versao:                 requisicao.Versao,
		Motivo:                 requisicao.Motivo,
		ValorADevolverCentavos: requisicao.ValorADevolverCentavos,
	})
	escreverItem(w, http.StatusOK, pedido, err, pedidodto.NovoPedidoResponse)
}

func (h *PedidoHandler) ListarHistorico(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	historico, err := h.usecase.ListarHistorico(r.Context(), id, usuario.ID)
	escreverLista(w, historico, err, pedidodto.NovoHistoricoResponses)
}

func (h *PedidoHandler) transicionar(
	w http.ResponseWriter,
	r *http.Request,
	mudar func(context.Context, portsin.TransicaoPedidoInput) (*domainpedido.Pedido, error),
) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao pedidodto.VersaoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	pedido, err := mudar(r.Context(), portsin.TransicaoPedidoInput{UsuarioID: usuario.ID, PedidoID: id, Versao: requisicao.Versao})
	escreverItem(w, http.StatusOK, pedido, err, pedidodto.NovoPedidoResponse)
}

func (h *PedidoHandler) Documento(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	documento, err := h.usecase.Documento(r.Context(), id, usuario.ID)
	escreverItem(w, http.StatusOK, documento, err, pedidodto.NovoDocumentoPropostaResponse)
}

func (h *PedidoHandler) ModelosDeVenda(w http.ResponseWriter, r *http.Request) {
	escreverLista(w, h.usecase.ModelosDeVenda(r.Context()), nil, pedidodto.NovoModeloDeVendaResponses)
}
