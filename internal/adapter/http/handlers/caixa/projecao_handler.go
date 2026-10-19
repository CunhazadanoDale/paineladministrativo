package caixa

import (
	"net/http"

	caixadto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/caixa"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
)

type ProjecaoHandler struct {
	usecase portsin.ProjecaoUseCase
}

func NewProjecaoHandler(usecase portsin.ProjecaoUseCase) *ProjecaoHandler {
	return &ProjecaoHandler{usecase: usecase}
}

func (h *ProjecaoHandler) Projetar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.ProjetarInput{UsuarioID: usuario.ID}

	var err error
	if input.Empresa, err = resposta.ConsultaUUID(r, "empresa"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if input.Ate, err = consultaData(r, "ate"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	projecao, err := h.usecase.Projetar(r.Context(), input)
	escreverItem(w, http.StatusOK, projecao, err, caixadto.NovaProjecaoResponse)
}

func (h *ProjecaoHandler) MovimentoDiario(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.MovimentoDiarioInput{UsuarioID: usuario.ID}

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
		return
	}

	movimentos, err := h.usecase.MovimentoDiario(r.Context(), input)
	escreverLista(w, movimentos, err, caixadto.NovosMovimentoDiarioResponses)
}
