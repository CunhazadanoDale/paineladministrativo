package importacao

import (
	"net/http"

	importacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/importacao"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/importacao"
)

type RecebimentoHandler struct {
	usecase portsin.RecebimentoUseCase
}

func NewRecebimentoHandler(usecase portsin.RecebimentoUseCase) *RecebimentoHandler {
	return &RecebimentoHandler{usecase: usecase}
}

func (h *RecebimentoHandler) DefinirAlocacoes(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao importacaodto.AlocacoesRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	visao, err := h.usecase.DefinirAlocacoes(r.Context(), portsin.DefinirAlocacoesInput{
		UsuarioID: usuario.ID, ImportacaoID: id, Versao: requisicao.Versao, Alocacoes: requisicao.ParaAlocacoes(),
	})
	escreverDetalhe(w, http.StatusOK, visao, err)
}

func (h *RecebimentoHandler) Receber(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao importacaodto.ReceberRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	recebidaEm, err := importacaodto.ParaData(requisicao.RecebidaEm, "data do recebimento")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	recebidas, err := requisicao.ParaRecebidas()
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	visao, err := h.usecase.Receber(r.Context(), portsin.ReceberInput{
		UsuarioID: usuario.ID, ImportacaoID: id, Versao: requisicao.Versao, RecebidaEm: recebidaEm,
		LocalID: requisicao.LocalID, CambioMicro: requisicao.CambioMicro, Recebidas: recebidas,
	})
	escreverDetalhe(w, http.StatusOK, visao, err)
}
