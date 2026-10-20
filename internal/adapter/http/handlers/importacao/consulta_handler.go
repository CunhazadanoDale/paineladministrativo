package importacao

import (
	"net/http"
	"strconv"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	importacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/importacao"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/importacao"
)

type ConsultaHandler struct {
	usecase portsin.ConsultaUseCase
}

func NewConsultaHandler(usecase portsin.ConsultaUseCase) *ConsultaHandler {
	return &ConsultaHandler{usecase: usecase}
}

func (h *ConsultaHandler) Alertas(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.AlertasInput{UsuarioID: usuario.ID}
	var err error
	if input.Empresa, err = resposta.ConsultaUUID(r, "empresa"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if texto := resposta.ConsultaTexto(r, "folga"); texto != "" {
		folga, err := strconv.Atoi(texto)
		if err != nil {
			resposta.ResponderErro(w, domain.ErroValidacao("parâmetro folga deve ser um número inteiro"))
			return
		}
		input.FolgaDias = &folga
	}

	alertas, err := h.usecase.Alertas(r.Context(), input)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[importacaodto.AlertasResponse]{Dados: importacaodto.NovosAlertasResponse(alertas)})
}

func (h *ConsultaHandler) ImportacoesDoPedido(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	atendimentos, err := h.usecase.ImportacoesDoPedido(r.Context(), id, usuario.ID)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[[]importacaodto.AtendimentoResponse]{Dados: importacaodto.NovosAtendimentosResponse(atendimentos)})
}
