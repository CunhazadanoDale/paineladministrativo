package usuarios

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

func TestMapearErro(t *testing.T) {
	casos := []struct {
		origem   error
		status   int
		mensagem string
	}{
		{domain.ErroValidacao("nome do usuário é obrigatório"), http.StatusBadRequest, "erro de validação: nome do usuário é obrigatório"},
		{domain.ErroNaoEncontrado("usuário não encontrado"), http.StatusNotFound, "registro não encontrado: usuário não encontrado"},
		{errors.New("banco indisponível"), http.StatusInternalServerError, "erro interno do servidor"},
	}

	for _, caso := range casos {
		status, mensagem := mapearErro(caso.origem)
		if status != caso.status {
			t.Errorf("erro %q mapeado para %d, esperado %d", caso.origem, status, caso.status)
		}
		if mensagem != caso.mensagem {
			t.Errorf("mensagem %q, esperada %q", mensagem, caso.mensagem)
		}
	}
}

func TestResponderErroUsaEnvelope(t *testing.T) {
	registrador := httptest.NewRecorder()

	responderErro(registrador, domain.ErroValidacao("nome do usuário é obrigatório"))

	if registrador.Code != http.StatusBadRequest {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusBadRequest)
	}
	if conteudo := registrador.Header().Get("Content-Type"); conteudo != "application/json; charset=utf-8" {
		t.Errorf("content-type %q inesperado", conteudo)
	}

	verificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)
}

func executarRequisicao(handler http.HandlerFunc, metodo, caminho, id, corpo string) *httptest.ResponseRecorder {
	requisicao := httptest.NewRequest(metodo, caminho, strings.NewReader(corpo))
	requisicao.SetPathValue("id", id)

	registrador := httptest.NewRecorder()
	handler(registrador, requisicao)

	return registrador
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
