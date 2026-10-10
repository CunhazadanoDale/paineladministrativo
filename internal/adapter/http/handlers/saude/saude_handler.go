package saude

import (
	"context"
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
)

type Saude struct {
	Status string `json:"status"`
}

type Pinger interface {
	PingContext(ctx context.Context) error
}

func Responder(w http.ResponseWriter, r *http.Request) {
	responderOK(w)
}

func ResponderComBanco(banco Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := banco.PingContext(r.Context()); err != nil {
			dto.EscreverErro(w, http.StatusServiceUnavailable, "banco de dados indisponível")
			return
		}

		responderOK(w)
	}
}

func responderOK(w http.ResponseWriter) {
	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[Saude]{
		Dados: Saude{Status: "ok"},
	})
}
