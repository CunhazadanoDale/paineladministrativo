package middleware

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
)

type respostaDoRoteador struct {
	cabecalhos http.Header
	status     int
}

func (r *respostaDoRoteador) Header() http.Header {
	return r.cabecalhos
}

func (r *respostaDoRoteador) Write(conteudo []byte) (int, error) {
	return len(conteudo), nil
}

func (r *respostaDoRoteador) WriteHeader(status int) {
	r.status = status
}

func ErrosDeRoteamento(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		manipulador, padrao := mux.Handler(r)
		if padrao != "" {
			mux.ServeHTTP(w, r)
			return
		}

		resposta := &respostaDoRoteador{cabecalhos: http.Header{}}
		manipulador.ServeHTTP(resposta, r)

		if resposta.status == http.StatusMethodNotAllowed {
			w.Header().Set("Allow", resposta.cabecalhos.Get("Allow"))
			dto.EscreverErro(w, http.StatusMethodNotAllowed, "método não permitido nesta rota")
			return
		}

		dto.EscreverErro(w, http.StatusNotFound, "rota não encontrada")
	})
}
