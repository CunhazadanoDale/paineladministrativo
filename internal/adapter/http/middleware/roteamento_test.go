package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
)

func roteadorDeTeste() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/leads", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("POST /api/v1/leads", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	return ErrosDeRoteamento(mux)
}

func TestErrosDeRoteamentoResponde405EmJSON(t *testing.T) {
	registro := httptest.NewRecorder()
	roteadorDeTeste().ServeHTTP(registro, httptest.NewRequest(http.MethodDelete, "/api/v1/leads", nil))

	if registro.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status %d, esperado %d", registro.Code, http.StatusMethodNotAllowed)
	}
	if permitido := registro.Header().Get("Allow"); permitido == "" {
		t.Error("405 sem cabeçalho Allow")
	}

	var corpo dto.RespostaErro
	if err := json.Unmarshal(registro.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("corpo %q não é JSON: %v", registro.Body.String(), err)
	}
	if corpo.Erro.Codigo != http.StatusMethodNotAllowed || corpo.Erro.Mensagem != "método não permitido nesta rota" {
		t.Errorf("erro %+v inesperado", corpo.Erro)
	}
}

func TestErrosDeRoteamentoResponde404EmJSON(t *testing.T) {
	registro := httptest.NewRecorder()
	roteadorDeTeste().ServeHTTP(registro, httptest.NewRequest(http.MethodGet, "/api/v1/nada", nil))

	var corpo dto.RespostaErro
	if err := json.Unmarshal(registro.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("corpo %q não é JSON: %v", registro.Body.String(), err)
	}
	if registro.Code != http.StatusNotFound || corpo.Erro.Mensagem != "rota não encontrada" {
		t.Errorf("status %d erro %+v, esperado 404 rota não encontrada", registro.Code, corpo.Erro)
	}
}

func TestErrosDeRoteamentoDeixaRotaConhecidaPassar(t *testing.T) {
	registro := httptest.NewRecorder()
	roteadorDeTeste().ServeHTTP(registro, httptest.NewRequest(http.MethodPost, "/api/v1/leads", nil))

	if registro.Code != http.StatusCreated {
		t.Errorf("status %d, esperado %d", registro.Code, http.StatusCreated)
	}
}
