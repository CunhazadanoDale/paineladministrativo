package caixa

import (
	"net/http"

	caixadto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/caixa"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
)

type ConciliacaoHandler struct {
	usecase portsin.ConciliacaoUseCase
}

func NewConciliacaoHandler(usecase portsin.ConciliacaoUseCase) *ConciliacaoHandler {
	return &ConciliacaoHandler{usecase: usecase}
}

func (h *ConciliacaoHandler) Sugestoes(w http.ResponseWriter, r *http.Request) {
	usuario, lancamentoID, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	sugestoes, err := h.usecase.Sugestoes(r.Context(), lancamentoID, usuario.ID)
	escreverLista(w, sugestoes, err, caixadto.NovosMovimentoResponses)
}

func (h *ConciliacaoHandler) Conciliar(w http.ResponseWriter, r *http.Request) {
	usuario, lancamentoID, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao caixadto.ConciliarRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	conciliacao, err := h.usecase.Conciliar(r.Context(), requisicao.ParaInput(usuario.ID, lancamentoID))
	escreverItem(w, http.StatusCreated, conciliacao, err, caixadto.NovaConciliacaoResponse)
}

func (h *ConciliacaoHandler) ConciliarAutomatico(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao caixadto.ConciliarAutomaticoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	input, err := requisicao.ParaInput(usuario.ID)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	resultado, err := h.usecase.ConciliarAutomatico(r.Context(), input)
	escreverItem(w, http.StatusOK, resultado, err, caixadto.NovoResultadoAutomaticoResponse)
}

func (h *ConciliacaoHandler) Obter(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	conciliacao, err := h.usecase.Obter(r.Context(), id, usuario.ID)
	escreverItem(w, http.StatusOK, conciliacao, err, caixadto.NovaConciliacaoResponse)
}

func (h *ConciliacaoHandler) Desfazer(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao caixadto.DesfazerConciliacaoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	conciliacao, err := h.usecase.Desfazer(r.Context(), portsin.DesfazerConciliacaoInput{
		UsuarioID:     usuario.ID,
		ConciliacaoID: id,
		Motivo:        requisicao.Motivo,
	})
	escreverItem(w, http.StatusOK, conciliacao, err, caixadto.NovaConciliacaoResponse)
}

func (h *ConciliacaoHandler) Pendencias(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.PendenciasInput{UsuarioID: usuario.ID, Filtro: resposta.ConsultaPaginacao(r)}

	conta, err := resposta.ConsultaUUID(r, "conta")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if conta == nil {
		resposta.ResponderErro(w, domain.ErroValidacao("parâmetro conta é obrigatório"))
		return
	}
	input.ContaBancariaID = *conta

	if input.De, err = consultaData(r, "de"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.Ate, err = consultaData(r, "ate"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	movimentos, err := h.usecase.Pendencias(r.Context(), input)
	escreverPagina(w, input.Filtro, movimentos, err, caixadto.NovosMovimentoResponses)
}

func (h *ConciliacaoHandler) PagamentosSemConta(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.PagamentosSemContaInput{UsuarioID: usuario.ID, Filtro: resposta.ConsultaPaginacao(r)}

	var err error
	if input.Empresa, err = resposta.ConsultaUUID(r, "empresa"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.De, err = consultaData(r, "de"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.Ate, err = consultaData(r, "ate"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	movimentos, err := h.usecase.PagamentosSemConta(r.Context(), input)
	escreverPagina(w, input.Filtro, movimentos, err, caixadto.NovosMovimentoResponses)
}
