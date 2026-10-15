package pedido

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	pedidodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/pedido"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/pedido"
)

type ConsultaHandler struct {
	usecase portsin.ConsultaUseCase
}

func NewConsultaHandler(usecase portsin.ConsultaUseCase) *ConsultaHandler {
	return &ConsultaHandler{usecase: usecase}
}

func (h *ConsultaHandler) Pendencias(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.ConsultarPendenciasInput{UsuarioID: usuario.ID, Filtro: resposta.ConsultaPaginacao(r)}
	var err error
	if input.Empresa, err = resposta.ConsultaUUID(r, "empresa"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.ClienteID, err = resposta.ConsultaUUID(r, "cliente"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.ProdutoID, err = resposta.ConsultaUUID(r, "produto"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	pendencias, err := h.usecase.Pendencias(r.Context(), input)
	escreverPagina(w, input.Filtro, pendencias, err, pedidodto.NovaPendenciaResponses)
}

func (h *ConsultaHandler) Prazos(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.ConsultarPrazosInput{
		UsuarioID: usuario.ID,
		Situacao:  resposta.ConsultaTexto(r, "situacao"),
		Filtro:    resposta.ConsultaPaginacao(r),
	}
	var err error
	if input.Empresa, err = resposta.ConsultaUUID(r, "empresa"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.Dias, err = consultaDias(r); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	pedidos, err := h.usecase.Prazos(r.Context(), input)
	escreverPagina(w, input.Filtro, pedidos, err, func(itens []*domainpedido.Pedido) []pedidodto.PedidoNoPrazoResponse {
		return pedidodto.NovoPedidoNoPrazoResponses(itens, time.Now().UTC())
	})
}

func (h *ConsultaHandler) Cancelamentos(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.ConsultarCancelamentosInput{UsuarioID: usuario.ID, Filtro: resposta.ConsultaPaginacao(r)}
	var err error
	if input.Empresa, err = resposta.ConsultaUUID(r, "empresa"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	comDevolucao, err := resposta.ConsultaBooleanaOpcional(r, "com_devolucao")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	input.ComDevolucao = comDevolucao != nil && *comDevolucao
	if input.De, err = consultaData(r, "de"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.Ate, err = consultaData(r, "ate"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	pedidos, err := h.usecase.Cancelamentos(r.Context(), input)
	escreverPagina(w, input.Filtro, pedidos, err, pedidodto.NovoPedidoResponses)
}

func consultaDias(r *http.Request) (*int, error) {
	valor := strings.TrimSpace(r.URL.Query().Get("dias"))
	if valor == "" {
		return nil, nil
	}

	dias, err := strconv.Atoi(valor)
	if err != nil {
		return nil, domain.ErroValidacao("parâmetro dias deve ser um número inteiro")
	}

	return &dias, nil
}

func consultaData(r *http.Request, nome string) (*time.Time, error) {
	valor := resposta.ConsultaTexto(r, nome)
	if valor == "" {
		return nil, nil
	}

	data, err := pedidodto.ParaData(valor, "parâmetro "+nome)
	if err != nil {
		return nil, err
	}

	return &data, nil
}
