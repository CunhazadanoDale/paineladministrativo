package comissao

import (
	"net/http"

	comissaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/comissao"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/comissao"
)

type FechamentoHandler struct {
	usecase portsin.FechamentoUseCase
}

func NewFechamentoHandler(usecase portsin.FechamentoUseCase) *FechamentoHandler {
	return &FechamentoHandler{usecase: usecase}
}

func (h *FechamentoHandler) Fechar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao comissaodto.FecharRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	resultado, err := h.usecase.Fechar(r.Context(), portsin.FecharInput{
		UsuarioID:             usuario.ID,
		EmpresaID:             requisicao.EmpresaID,
		Competencia:           requisicao.Competencia,
		CategoriaFinanceiraID: requisicao.CategoriaFinanceiraID,
		RepresentanteID:       requisicao.RepresentanteID,
	})
	escreverItem(w, http.StatusOK, resultado, err, comissaodto.NovoResultadoFechamentoResponse)
}

func (h *FechamentoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.ListarFechamentosInput{
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

	fechamentos, err := h.usecase.Listar(r.Context(), input)
	escreverPagina(w, input.Filtro, fechamentos, err, comissaodto.NovoFechamentoResponses)
}

func (h *FechamentoHandler) Obter(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	detalhe, err := h.usecase.Obter(r.Context(), id, usuario.ID)
	escreverItem(w, http.StatusOK, detalhe, err, comissaodto.NovoFechamentoDetalheResponse)
}
