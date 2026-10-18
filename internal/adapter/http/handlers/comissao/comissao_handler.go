package comissao

import (
	"net/http"

	comissaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/comissao"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/comissao"
)

type ComissaoHandler struct {
	usecase portsin.ComissaoUseCase
}

func NewComissaoHandler(usecase portsin.ComissaoUseCase) *ComissaoHandler {
	return &ComissaoHandler{usecase: usecase}
}

func (h *ComissaoHandler) Apurar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao comissaodto.ApurarRequest
	if _, ok := resposta.CorpoJSONOpcional(w, r, &requisicao); !ok {
		return
	}

	resultado, err := h.usecase.Apurar(r.Context(), portsin.ApurarInput{
		UsuarioID: usuario.ID, Empresa: requisicao.EmpresaID, Competencia: requisicao.Competencia,
	})
	escreverItem(w, http.StatusOK, resultado, err, comissaodto.NovoResultadoApuracaoResponse)
}

func (h *ComissaoHandler) Pendencias(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	empresa, err := resposta.ConsultaUUID(r, "empresa")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	input := portsin.PendenciasInput{
		UsuarioID:   usuario.ID,
		Empresa:     empresa,
		Competencia: resposta.ConsultaTexto(r, "competencia"),
		Filtro:      resposta.ConsultaPaginacao(r),
	}
	entregas, err := h.usecase.Pendencias(r.Context(), input)
	escreverPagina(w, input.Filtro, entregas, err, comissaodto.NovoEntregaPendenteResponses)
}

func (h *ComissaoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.ListarComissoesInput{
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

	comissoes, err := h.usecase.Listar(r.Context(), input)
	escreverPagina(w, input.Filtro, comissoes, err, comissaodto.NovoComissaoResponses)
}
