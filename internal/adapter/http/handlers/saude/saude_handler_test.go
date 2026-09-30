package saude

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
)

type pingerFalso struct {
	erro error
}

func (p pingerFalso) PingContext(ctx context.Context) error {
	return p.erro
}

func TestResponderUsaEnvelope(t *testing.T) {
	requisicao := httptest.NewRequest(http.MethodGet, "/health", nil)
	registrador := httptest.NewRecorder()

	Responder(registrador, requisicao)

	if registrador.Code != http.StatusOK {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusOK)
	}

	var resposta dto.Resposta[Saude]
	if err := json.Unmarshal(registrador.Body.Bytes(), &resposta); err != nil {
		t.Fatalf("corpo não é um envelope válido: %v", err)
	}
	if resposta.Dados.Status != "ok" {
		t.Errorf("status %q, esperado %q", resposta.Dados.Status, "ok")
	}
}

func TestResponderComBancoDisponivel(t *testing.T) {
	requisicao := httptest.NewRequest(http.MethodGet, "/health/db", nil)
	registrador := httptest.NewRecorder()

	ResponderComBanco(pingerFalso{})(registrador, requisicao)

	if registrador.Code != http.StatusOK {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusOK)
	}
}

func TestResponderComBancoIndisponivel(t *testing.T) {
	requisicao := httptest.NewRequest(http.MethodGet, "/health/db", nil)
	registrador := httptest.NewRecorder()

	ResponderComBanco(pingerFalso{erro: errors.New("conexão recusada")})(registrador, requisicao)

	if registrador.Code != http.StatusServiceUnavailable {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusServiceUnavailable)
	}

	verificarEnvelopeDeErro(t, registrador, http.StatusServiceUnavailable)
}

func TestNaoEncontrado(t *testing.T) {
	requisicao := httptest.NewRequest(http.MethodGet, "/nao-existe", nil)
	registrador := httptest.NewRecorder()

	NaoEncontrado(registrador, requisicao)

	if registrador.Code != http.StatusNotFound {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusNotFound)
	}

	verificarEnvelopeDeErro(t, registrador, http.StatusNotFound)
}

func verificarEnvelopeDeErro(t *testing.T, registrador *httptest.ResponseRecorder, codigo int) {
	t.Helper()

	if conteudo := registrador.Header().Get("Content-Type"); conteudo != "application/json; charset=utf-8" {
		t.Errorf("content-type %q inesperado", conteudo)
	}

	var resposta dto.RespostaErro
	if err := json.Unmarshal(registrador.Body.Bytes(), &resposta); err != nil {
		t.Fatalf("corpo não é um envelope de erro válido: %v", err)
	}
	if resposta.Erro.Codigo != codigo {
		t.Errorf("código do envelope %d, esperado %d", resposta.Erro.Codigo, codigo)
	}
	if resposta.Erro.Mensagem == "" {
		t.Error("mensagem do envelope vazia")
	}
}
