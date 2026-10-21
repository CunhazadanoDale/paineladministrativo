package dre

import (
	"net/http"

	dredto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/dre"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/dre"
)

func consultaDoPeriodo(w http.ResponseWriter, r *http.Request) (portsin.ConsultarPeriodoInput, bool) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return portsin.ConsultarPeriodoInput{}, false
	}

	input := portsin.ConsultarPeriodoInput{UsuarioID: usuario.ID}
	var err error
	if input.Empresa, err = resposta.ConsultaUUID(r, "empresa"); err != nil {
		resposta.ResponderErro(w, err)
		return portsin.ConsultarPeriodoInput{}, false
	}
	if input.De, err = dredto.MesDeConsulta(resposta.ConsultaTexto(r, "de"), "de"); err != nil {
		resposta.ResponderErro(w, err)
		return portsin.ConsultarPeriodoInput{}, false
	}
	if input.Ate, err = dredto.MesDeConsulta(resposta.ConsultaTexto(r, "ate"), "ate"); err != nil {
		resposta.ResponderErro(w, err)
		return portsin.ConsultarPeriodoInput{}, false
	}

	return input, true
}
