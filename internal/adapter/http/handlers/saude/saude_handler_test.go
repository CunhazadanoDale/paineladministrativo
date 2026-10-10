package saude

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
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

	apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusServiceUnavailable)
}
