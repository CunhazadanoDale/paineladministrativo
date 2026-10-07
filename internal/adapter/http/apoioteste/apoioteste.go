package apoioteste

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
)

func ExecutarHandler(handler http.HandlerFunc, metodo, caminho, id, corpo string) *httptest.ResponseRecorder {
	requisicao := httptest.NewRequest(metodo, caminho, strings.NewReader(corpo))
	requisicao.SetPathValue("id", id)

	registrador := httptest.NewRecorder()
	handler(registrador, requisicao)

	return registrador
}

func MensagemDoEnvelope(t *testing.T, registrador *httptest.ResponseRecorder) string {
	t.Helper()

	var envelope dto.RespostaErro
	if err := json.Unmarshal(registrador.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("corpo não é um envelope de erro válido: %v", err)
	}

	return envelope.Erro.Mensagem
}

func VerificarEnvelopeDeErro(t *testing.T, registrador *httptest.ResponseRecorder, codigo int) {
	t.Helper()

	if conteudo := registrador.Header().Get("Content-Type"); conteudo != "application/json; charset=utf-8" {
		t.Errorf("content-type %q inesperado", conteudo)
	}

	var envelope dto.RespostaErro
	if err := json.Unmarshal(registrador.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("corpo não é um envelope de erro válido: %v", err)
	}
	if envelope.Erro.Codigo != codigo {
		t.Errorf("código do envelope %d, esperado %d", envelope.Erro.Codigo, codigo)
	}
	if envelope.Erro.Mensagem == "" {
		t.Error("mensagem do envelope vazia")
	}
}
