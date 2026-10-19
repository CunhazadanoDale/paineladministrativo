package caixa

import (
	"net/http"

	caixadto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/caixa"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
)

type LancamentoCaixaHandler struct {
	usecase portsin.LancamentoCaixaUseCase
}

func NewLancamentoCaixaHandler(usecase portsin.LancamentoCaixaUseCase) *LancamentoCaixaHandler {
	return &LancamentoCaixaHandler{usecase: usecase}
}

func (h *LancamentoCaixaHandler) Classificar(w http.ResponseWriter, r *http.Request) {
	usuario, lancamentoID, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao caixadto.ClassificarRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	classificacao, err := h.usecase.Classificar(r.Context(), requisicao.ParaInput(usuario.ID, lancamentoID))
	escreverItem(w, http.StatusCreated, classificacao, err, caixadto.NovaClassificacaoResponse)
}

func (h *LancamentoCaixaHandler) Transferir(w http.ResponseWriter, r *http.Request) {
	usuario, lancamentoID, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao caixadto.TransferirRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	registrada, err := h.usecase.Transferir(r.Context(), requisicao.ParaInput(usuario.ID, lancamentoID))
	escreverItem(w, http.StatusCreated, registrada, err, caixadto.NovaTransferenciaRegistradaResponse)
}

func (h *LancamentoCaixaHandler) Listar(w http.ResponseWriter, r *http.Request) {
	input, ok := entradaDaListagem(w, r)
	if !ok {
		return
	}

	lancamentos, err := h.usecase.Listar(r.Context(), input)
	escreverPagina(w, input.Filtro, lancamentos, err, caixadto.NovosLancamentoCaixaResponses)
}

func (h *LancamentoCaixaHandler) ListarTransferencias(w http.ResponseWriter, r *http.Request) {
	input, ok := entradaDaListagem(w, r)
	if !ok {
		return
	}

	transferencias, err := h.usecase.ListarTransferencias(r.Context(), input)
	escreverPagina(w, input.Filtro, transferencias, err, caixadto.NovasTransferenciaResponses)
}

func entradaDaListagem(w http.ResponseWriter, r *http.Request) (portsin.ListarCaixaInput, bool) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return portsin.ListarCaixaInput{}, false
	}

	input := portsin.ListarCaixaInput{UsuarioID: usuario.ID, Filtro: resposta.ConsultaPaginacao(r)}

	var err error
	if input.Empresa, err = resposta.ConsultaUUID(r, "empresa"); err == nil {
		if input.ContaBancariaID, err = resposta.ConsultaUUID(r, "conta"); err == nil {
			if input.De, err = consultaData(r, "de"); err == nil {
				input.Ate, err = consultaData(r, "ate")
			}
		}
	}
	if err != nil {
		resposta.ResponderErro(w, err)
		return portsin.ListarCaixaInput{}, false
	}

	return input, true
}
