package cliente

import (
	"errors"
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	clientedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/cliente"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/cliente"
)

const tamanhoMaximoImportacao = 5 << 20

type ImportacaoHandler struct {
	usecase portsin.ImportacaoUseCase
}

func NewImportacaoHandler(usecase portsin.ImportacaoUseCase) *ImportacaoHandler {
	return &ImportacaoHandler{usecase: usecase}
}

func (h *ImportacaoHandler) Importar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	simular, err := resposta.ConsultaBooleanaOpcional(r, "simular")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	linhas, err := clientedto.LerLinhasCSV(http.MaxBytesReader(w, r.Body, tamanhoMaximoImportacao))
	if err != nil {
		var erroTamanho *http.MaxBytesError
		if errors.As(err, &erroTamanho) {
			dto.EscreverErro(w, http.StatusRequestEntityTooLarge, "arquivo excede o limite de 5MB")
			return
		}
		resposta.ResponderErro(w, err)
		return
	}

	relatorio, err := h.usecase.Importar(r.Context(), portsin.ImportarClientesInput{
		UsuarioID: usuario.ID,
		Linhas:    linhas,
		Simular:   simular != nil && *simular,
	})
	escreverItem(w, http.StatusOK, relatorio, err, clientedto.NovoRelatorioImportacaoResponse)
}

func (h *ClienteHandler) ConverterLead(w http.ResponseWriter, r *http.Request) {
	usuario, leadID, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao clientedto.ConverterLeadRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	cliente, err := h.usecase.ConverterLead(r.Context(), portsin.ConverterLeadInput{
		UsuarioID: usuario.ID,
		LeadID:    leadID,
		Documento: requisicao.Documento,
		Dados:     requisicao.DadosClienteRequest.ParaInput(),
	})
	escreverItem(w, http.StatusCreated, cliente, err, clientedto.NovoClienteResponse)
}
