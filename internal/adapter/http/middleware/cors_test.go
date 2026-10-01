package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSPreflightNaoChegaNaRota(t *testing.T) {
	chamou := false
	destino := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chamou = true
	})

	requisicao := httptest.NewRequest(http.MethodOptions, "/api/v1/leads", nil)
	requisicao.Header.Set("Origin", "http://localhost:5173")
	requisicao.Header.Set("Access-Control-Request-Method", "POST")
	requisicao.Header.Set("Access-Control-Request-Headers", "Content-Type")
	registrador := httptest.NewRecorder()

	CORS(destino).ServeHTTP(registrador, requisicao)

	if registrador.Code != http.StatusNoContent {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusNoContent)
	}
	if chamou {
		t.Error("preflight não deve alcançar a rota")
	}
	verificarCabecalhosCORS(t, registrador)
}

func TestCORSRequisicaoComum(t *testing.T) {
	chamou := false
	destino := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chamou = true
		w.WriteHeader(http.StatusCreated)
	})

	requisicao := httptest.NewRequest(http.MethodPost, "/api/v1/leads", nil)
	requisicao.Header.Set("Origin", "http://localhost:5173")
	registrador := httptest.NewRecorder()

	CORS(destino).ServeHTTP(registrador, requisicao)

	if registrador.Code != http.StatusCreated {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusCreated)
	}
	if !chamou {
		t.Error("requisição comum deve alcançar a rota")
	}
	verificarCabecalhosCORS(t, registrador)
}

func verificarCabecalhosCORS(t *testing.T, registrador *httptest.ResponseRecorder) {
	t.Helper()

	casos := []struct {
		cabecalho string
		esperado  string
	}{
		{"Access-Control-Allow-Origin", "*"},
		{"Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS"},
		{"Access-Control-Allow-Headers", "Content-Type, Authorization, Accept"},
		{"Access-Control-Max-Age", "86400"},
	}

	for _, caso := range casos {
		if valor := registrador.Header().Get(caso.cabecalho); valor != caso.esperado {
			t.Errorf("%s = %q, esperado %q", caso.cabecalho, valor, caso.esperado)
		}
	}
}
