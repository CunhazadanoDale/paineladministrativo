package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSPreflightDeOrigemPermitida(t *testing.T) {
	chamou := false
	destino := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chamou = true
	})

	requisicao := preflight("http://localhost:5173")
	registrador := httptest.NewRecorder()

	CORS([]string{"http://localhost:5173", "http://localhost:3000"}, destino).ServeHTTP(registrador, requisicao)

	if registrador.Code != http.StatusNoContent {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusNoContent)
	}
	if chamou {
		t.Error("preflight não deve alcançar a rota")
	}
	if origem := registrador.Header().Get("Access-Control-Allow-Origin"); origem != "http://localhost:5173" {
		t.Errorf("Access-Control-Allow-Origin %q, esperado %q", origem, "http://localhost:5173")
	}
	if variacao := registrador.Header().Get("Vary"); variacao != "Origin" {
		t.Errorf("Vary %q, esperado %q", variacao, "Origin")
	}
	verificarCabecalhosDeMetodo(t, registrador)
}

func TestCORSRequisicaoComumDeOrigemPermitida(t *testing.T) {
	chamou := false
	destino := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chamou = true
		w.WriteHeader(http.StatusCreated)
	})

	requisicao := httptest.NewRequest(http.MethodPost, "/api/v1/leads", nil)
	requisicao.Header.Set("Origin", "http://localhost:3000")
	registrador := httptest.NewRecorder()

	CORS([]string{"http://localhost:5173", "http://localhost:3000"}, destino).ServeHTTP(registrador, requisicao)

	if registrador.Code != http.StatusCreated {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusCreated)
	}
	if !chamou {
		t.Error("requisição comum deve alcançar a rota")
	}
	if origem := registrador.Header().Get("Access-Control-Allow-Origin"); origem != "http://localhost:3000" {
		t.Errorf("Access-Control-Allow-Origin %q, esperado %q", origem, "http://localhost:3000")
	}
}

func TestCORSOrigemNaoPermitida(t *testing.T) {
	destino := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	requisicao := preflight("http://site-malicioso.com")
	registrador := httptest.NewRecorder()

	CORS([]string{"http://localhost:5173"}, destino).ServeHTTP(registrador, requisicao)

	if registrador.Code != http.StatusNoContent {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusNoContent)
	}
	if origem := registrador.Header().Get("Access-Control-Allow-Origin"); origem != "" {
		t.Errorf("origem bloqueada recebeu Access-Control-Allow-Origin %q", origem)
	}
}

func TestCORSListaVaziaNaoLiberaOrigemNenhuma(t *testing.T) {
	destino := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	requisicao := httptest.NewRequest(http.MethodGet, "/api/v1/leads", nil)
	requisicao.Header.Set("Origin", "http://qualquer-uma.com")
	registrador := httptest.NewRecorder()

	CORS([]string{}, destino).ServeHTTP(registrador, requisicao)

	if origem := registrador.Header().Get("Access-Control-Allow-Origin"); origem != "" {
		t.Errorf("Access-Control-Allow-Origin %q, esperado vazio", origem)
	}
}

func TestCORSComAsteriscoNaLista(t *testing.T) {
	destino := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	requisicao := httptest.NewRequest(http.MethodGet, "/api/v1/leads", nil)
	requisicao.Header.Set("Origin", "http://qualquer-uma.com")
	registrador := httptest.NewRecorder()

	CORS([]string{"*"}, destino).ServeHTTP(registrador, requisicao)

	if origem := registrador.Header().Get("Access-Control-Allow-Origin"); origem != "*" {
		t.Errorf("Access-Control-Allow-Origin %q, esperado %q", origem, "*")
	}
}

func TestCORSRequisicaoSemOrigin(t *testing.T) {
	chamou := false
	destino := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chamou = true
	})

	requisicao := httptest.NewRequest(http.MethodGet, "/health", nil)
	registrador := httptest.NewRecorder()

	CORS([]string{"http://localhost:5173"}, destino).ServeHTTP(registrador, requisicao)

	if !chamou {
		t.Error("requisição sem Origin deve alcançar a rota")
	}
	if origem := registrador.Header().Get("Access-Control-Allow-Origin"); origem != "" {
		t.Errorf("requisição sem Origin não deveria receber Access-Control-Allow-Origin, veio %q", origem)
	}
}

func preflight(origem string) *http.Request {
	requisicao := httptest.NewRequest(http.MethodOptions, "/api/v1/leads", nil)
	requisicao.Header.Set("Origin", origem)
	requisicao.Header.Set("Access-Control-Request-Method", "POST")
	requisicao.Header.Set("Access-Control-Request-Headers", "Content-Type")

	return requisicao
}

func verificarCabecalhosDeMetodo(t *testing.T, registrador *httptest.ResponseRecorder) {
	t.Helper()
	tabela := []struct {
		cabecalho string
		esperado  string
	}{
		{"Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS"},
		{"Access-Control-Allow-Headers", "Content-Type, Authorization, Accept"},
		{"Access-Control-Max-Age", "86400"},
	}

	for _, linha := range tabela {
		if valor := registrador.Header().Get(linha.cabecalho); valor != linha.esperado {
			t.Errorf("%s = %q, esperado %q", linha.cabecalho, valor, linha.esperado)
		}
	}
}

func TestCORSExpoeCabecalhosDeRastreamento(t *testing.T) {
	destino := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	requisicao := httptest.NewRequest(http.MethodGet, "/api/v1/leads", nil)
	requisicao.Header.Set("Origin", "http://localhost:5173")
	registrador := httptest.NewRecorder()

	CORS([]string{"http://localhost:5173"}, destino).ServeHTTP(registrador, requisicao)

	if expostos := registrador.Header().Get("Access-Control-Expose-Headers"); expostos != "X-Request-Id, Retry-After" {
		t.Errorf("Access-Control-Expose-Headers %q, esperado %q", expostos, "X-Request-Id, Retry-After")
	}
}
