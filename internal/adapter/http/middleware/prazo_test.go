package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func servidorComPrazoCurto(t *testing.T, manipulador http.Handler) *httptest.Server {
	t.Helper()

	servidor := httptest.NewUnstartedServer(Registrar(slog.New(slog.DiscardHandler), manipulador))
	servidor.Config.ReadTimeout = 100 * time.Millisecond
	servidor.Config.WriteTimeout = 100 * time.Millisecond
	servidor.Start()
	t.Cleanup(servidor.Close)

	return servidor
}

func respostaLenta(w http.ResponseWriter, r *http.Request) {
	time.Sleep(300 * time.Millisecond)
	_, _ = io.WriteString(w, "concluído")
}

func TestEstenderPrazoPermiteRespostaAlemDoTimeoutDoServidor(t *testing.T) {
	servidor := servidorComPrazoCurto(t, EstenderPrazo(5*time.Second, respostaLenta))

	resposta, err := servidor.Client().Get(servidor.URL)
	if err != nil {
		t.Fatalf("requisição falhou: %v", err)
	}
	defer resposta.Body.Close()

	corpo, err := io.ReadAll(resposta.Body)
	if err != nil {
		t.Fatalf("leitura do corpo falhou: %v", err)
	}
	if resposta.StatusCode != http.StatusOK || string(corpo) != "concluído" {
		t.Errorf("status %d corpo %q, esperado 200 concluído", resposta.StatusCode, corpo)
	}
}

func TestSemPrazoEstendidoORespostaLentaECortada(t *testing.T) {
	servidor := servidorComPrazoCurto(t, http.HandlerFunc(respostaLenta))

	resposta, err := servidor.Client().Get(servidor.URL)
	if err == nil {
		defer resposta.Body.Close()
		if corpo, erroLeitura := io.ReadAll(resposta.Body); erroLeitura == nil && string(corpo) == "concluído" {
			t.Fatal("resposta lenta passou sem prazo estendido; o teste não exercita o timeout")
		}
	}
}

func TestEstenderPrazoSemConexaoRealSegueParaARota(t *testing.T) {
	registro := httptest.NewRecorder()
	EstenderPrazo(time.Second, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}).ServeHTTP(registro, httptest.NewRequest(http.MethodGet, "/", nil))

	if registro.Code != http.StatusNoContent {
		t.Errorf("status %d, esperado %d", registro.Code, http.StatusNoContent)
	}
}
