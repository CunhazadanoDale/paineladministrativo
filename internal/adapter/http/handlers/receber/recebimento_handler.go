package receber

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	receberdto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/receber"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/receber"
)

type RecebimentoHandler struct {
	usecase portsin.RecebimentoUseCase
}

func NewRecebimentoHandler(usecase portsin.RecebimentoUseCase) *RecebimentoHandler {
	return &RecebimentoHandler{usecase: usecase}
}

func (h *RecebimentoHandler) Receber(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao receberdto.ReceberRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	recebidoEm, err := receberdto.ParaData(requisicao.RecebidoEm, "data do recebimento")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	titulo, recebimento, err := h.usecase.Receber(r.Context(), portsin.ReceberInput{
		UsuarioID:        usuario.ID,
		TituloID:         id,
		Versao:           requisicao.Versao,
		Tipo:             requisicao.Tipo,
		ContaBancariaID:  requisicao.ContaBancariaID,
		RecebidoEm:       recebidoEm,
		ValorCentavos:    requisicao.ValorCentavos,
		JurosCentavos:    requisicao.JurosCentavos,
		DescontoCentavos: requisicao.DescontoCentavos,
	})
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusCreated, dto.Resposta[receberdto.RecebimentoRegistradoResponse]{
		Dados: receberdto.NovoRecebimentoRegistradoResponse(titulo, recebimento),
	})
}

func (h *RecebimentoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	recebimentos, err := h.usecase.ListarRecebimentos(r.Context(), id, usuario.ID)
	escreverLista(w, recebimentos, err, receberdto.NovoRecebimentoResponses)
}

func (h *RecebimentoHandler) Estornar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}
	recebimentoID, ok := resposta.ParametroUUID(w, r, "recebimento_id")
	if !ok {
		return
	}

	var requisicao receberdto.EstornoRecebimentoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	estornadoEm, err := receberdto.ParaData(requisicao.EstornadoEm, "data do estorno")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	titulo, err := h.usecase.EstornarRecebimento(r.Context(), portsin.EstornarRecebimentoInput{
		UsuarioID:     usuario.ID,
		TituloID:      id,
		RecebimentoID: recebimentoID,
		Versao:        requisicao.Versao,
		Motivo:        requisicao.Motivo,
		EstornadoEm:   estornadoEm,
	})
	escreverItem(w, http.StatusOK, titulo, err, receberdto.NovoTituloResponse)
}
