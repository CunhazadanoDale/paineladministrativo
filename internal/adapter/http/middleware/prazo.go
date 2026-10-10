package middleware

import (
	"errors"
	"net/http"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
)

func EstenderPrazo(prazo time.Duration, proximo http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		controle := http.NewResponseController(w)
		limite := time.Now().Add(prazo)

		for _, definir := range []func(time.Time) error{controle.SetReadDeadline, controle.SetWriteDeadline} {
			if err := definir(limite); err != nil && !errors.Is(err, http.ErrNotSupported) {
				AnotarErro(w, err)
				dto.EscreverErro(w, http.StatusInternalServerError, "erro interno do servidor")
				return
			}
		}

		proximo(w, r)
	}
}
