package comissao

import (
	"net/http"

	comissaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/comissao"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/comissao"
)

type AjusteHandler struct {
	usecase portsin.AjusteUseCase
}

func NewAjusteHandler(usecase portsin.AjusteUseCase) *AjusteHandler {
	return &AjusteHandler{usecase: usecase}
}

func (h *AjusteHandler) Criar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao comissaodto.CriarAjusteRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	ajuste, err := h.usecase.Criar(r.Context(), portsin.CriarAjusteInput{UsuarioID: usuario.ID, Dados: requisicao.ParaDados()})
	escreverItem(w, http.StatusCreated, ajuste, err, comissaodto.NovoAjusteResponse)
}

func (h *AjusteHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.ListarAjustesInput{
		UsuarioID:   usuario.ID,
		Competencia: resposta.ConsultaTexto(r, "competencia"),
		Filtro:      resposta.ConsultaPaginacao(r),
	}
	var err error
	if input.Empresa, err = resposta.ConsultaUUID(r, "empresa"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.RepresentanteID, err = resposta.ConsultaUUID(r, "representante"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.PedidoID, err = resposta.ConsultaUUID(r, "pedido"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.Aberto, err = resposta.ConsultaBooleanaOpcional(r, "aberto"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	ajustes, err := h.usecase.Listar(r.Context(), input)
	escreverPagina(w, input.Filtro, ajustes, err, comissaodto.NovoAjusteResponses)
}
