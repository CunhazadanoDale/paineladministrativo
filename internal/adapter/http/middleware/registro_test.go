package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
)

func registradorEmMemoria() (*slog.Logger, *bytes.Buffer) {
	saida := &bytes.Buffer{}

	return slog.New(slog.NewJSONHandler(saida, nil)), saida
}

func lerRegistro(t *testing.T, saida *bytes.Buffer) map[string]any {
	t.Helper()

	linhas := strings.Split(strings.TrimSpace(saida.String()), "\n")
	if len(linhas) != 1 {
		t.Fatalf("%d linhas de registro, esperada 1: %q", len(linhas), saida.String())
	}

	var registro map[string]any
	if err := json.Unmarshal([]byte(linhas[0]), &registro); err != nil {
		t.Fatalf("registro não é JSON: %v", err)
	}

	return registro
}

func atender(registrador *slog.Logger, rota http.HandlerFunc) *httptest.ResponseRecorder {
	registro := httptest.NewRecorder()
	Registrar(registrador, rota).ServeHTTP(registro, httptest.NewRequest(http.MethodGet, "/api/v1/produtos", nil))

	return registro
}

func TestRegistrarAnotaRequisicaoComSucesso(t *testing.T) {
	registrador, saida := registradorEmMemoria()

	resposta := atender(registrador, func(w http.ResponseWriter, r *http.Request) {
		dto.EscreverJSON(w, http.StatusCreated, map[string]string{"ok": "sim"})
	})

	if resposta.Code != http.StatusCreated {
		t.Fatalf("status %d, esperado %d", resposta.Code, http.StatusCreated)
	}

	registro := lerRegistro(t, saida)
	if registro["level"] != "INFO" {
		t.Errorf("nível %v, esperado INFO", registro["level"])
	}
	if registro["status"] != float64(http.StatusCreated) || registro["caminho"] != "/api/v1/produtos" || registro["metodo"] != http.MethodGet {
		t.Errorf("registro %v sem status, caminho ou método esperados", registro)
	}
	if _, temErro := registro["erro"]; temErro {
		t.Errorf("registro de sucesso com erro: %v", registro)
	}

	id := resposta.Header().Get(CabecalhoRequisicao)
	if id == "" || registro["requisicao_id"] != id {
		t.Errorf("id da requisição %q no cabeçalho e %v no registro", id, registro["requisicao_id"])
	}
}

func TestRegistrarIncluiACausaDoErroInterno(t *testing.T) {
	registrador, saida := registradorEmMemoria()

	atender(registrador, func(w http.ResponseWriter, r *http.Request) {
		AnotarErro(w, errors.New("conexão com o banco recusada"))
		dto.EscreverErro(w, http.StatusInternalServerError, "erro interno do servidor")
	})

	registro := lerRegistro(t, saida)
	if registro["level"] != "ERROR" {
		t.Errorf("nível %v, esperado ERROR", registro["level"])
	}
	if registro["erro"] != "conexão com o banco recusada" {
		t.Errorf("erro registrado %v, esperado a causa anotada", registro["erro"])
	}
}

func TestRegistrarRecuperaPanicComErroInterno(t *testing.T) {
	registrador, saida := registradorEmMemoria()

	resposta := atender(registrador, func(w http.ResponseWriter, r *http.Request) {
		panic("mapa nulo")
	})

	if resposta.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, esperado %d", resposta.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(resposta.Body.String(), "erro interno do servidor") {
		t.Errorf("corpo %q sem envelope de erro", resposta.Body.String())
	}

	registro := lerRegistro(t, saida)
	if registro["erro"] != "panic: mapa nulo" {
		t.Errorf("erro registrado %v, esperado o panic", registro["erro"])
	}
}

func TestRegistrarRepassaAbortHandler(t *testing.T) {
	registrador, _ := registradorEmMemoria()

	defer func() {
		if recuperado := recover(); recuperado != http.ErrAbortHandler {
			t.Errorf("recuperado %v, esperado http.ErrAbortHandler", recuperado)
		}
	}()

	atender(registrador, func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	})
}
