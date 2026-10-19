package caixa

import (
	"net/http"

	caixadto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/caixa"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
)

type FechamentoHandler struct {
	usecase portsin.FechamentoUseCase
}

func NewFechamentoHandler(usecase portsin.FechamentoUseCase) *FechamentoHandler {
	return &FechamentoHandler{usecase: usecase}
}

func (h *FechamentoHandler) Previa(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	conta, err := resposta.ConsultaUUID(r, "conta")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if conta == nil {
		resposta.ResponderErro(w, domain.ErroValidacao("parâmetro conta é obrigatório"))
		return
	}
	competencia, err := caixadto.ParaCompetencia(resposta.ConsultaTexto(r, "competencia"), "parâmetro competencia")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	previa, err := h.usecase.Previa(r.Context(), portsin.PreviaInput{UsuarioID: usuario.ID, ContaBancariaID: *conta, Competencia: competencia})
	escreverItem(w, http.StatusOK, previa, err, caixadto.NovaPreviaResponse)
}

func (h *FechamentoHandler) Fechar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao caixadto.FecharRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	input, err := requisicao.ParaInput(usuario.ID)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	detalhe, err := h.usecase.Fechar(r.Context(), input)
	escreverItem(w, http.StatusCreated, detalhe, err, caixadto.NovoDetalheFechamentoResponse)
}

func (h *FechamentoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.ListarFechamentosInput{UsuarioID: usuario.ID, Filtro: resposta.ConsultaPaginacao(r)}

	var err error
	if input.Empresa, err = resposta.ConsultaUUID(r, "empresa"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.ContaBancariaID, err = resposta.ConsultaUUID(r, "conta"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	competencia, err := caixadto.ParaCompetencia(resposta.ConsultaTexto(r, "competencia"), "parâmetro competencia")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if !competencia.IsZero() {
		input.Competencia = &competencia
	}

	fechamentos, err := h.usecase.Listar(r.Context(), input)
	escreverPagina(w, input.Filtro, fechamentos, err, caixadto.NovosFechamentoResponses)
}

func (h *FechamentoHandler) Obter(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	detalhe, err := h.usecase.Obter(r.Context(), id, usuario.ID)
	escreverItem(w, http.StatusOK, detalhe, err, caixadto.NovoDetalheFechamentoResponse)
}

func (h *FechamentoHandler) Reabrir(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao caixadto.ReabrirRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	fechamento, err := h.usecase.Reabrir(r.Context(), portsin.ReabrirInput{
		UsuarioID:    usuario.ID,
		FechamentoID: id,
		Motivo:       requisicao.Motivo,
	})
	escreverItem(w, http.StatusOK, fechamento, err, caixadto.NovoFechamentoResponse)
}
