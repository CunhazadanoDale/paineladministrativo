package middleware

import "net/http"

func CORS(origens []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		liberarOrigem(w, r.Header.Get("Origin"), origens)

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept")
		w.Header().Set("Access-Control-Expose-Headers", CabecalhoRequisicao+", Retry-After")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func liberarOrigem(w http.ResponseWriter, origem string, origens []string) {
	if contem(origens, "*") {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		return
	}

	if origem != "" && contem(origens, origem) {
		w.Header().Set("Access-Control-Allow-Origin", origem)
	}
}

func contem(origens []string, procurado string) bool {
	for _, origem := range origens {
		if origem == procurado {
			return true
		}
	}

	return false
}
