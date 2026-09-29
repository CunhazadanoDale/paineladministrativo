package lead

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
		{domain.ErroValidacao("nome do lead é obrigatório"), http.StatusBadRequest, "erro de validação: nome do lead é obrigatório"},
		{domain.ErroNaoEncontrado("lead não encontrado"), http.StatusNotFound, "registro não encontrado: lead não encontrado"},
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

	responderErro(registrador, domain.ErroValidacao("nome do lead é obrigatório"))

	if registrador.Code != http.StatusBadRequest {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusBadRequest)
	}
	if conteudo := registrador.Header().Get("Content-Type"); conteudo != "application/json; charset=utf-8" {
		t.Errorf("content-type %q inesperado", conteudo)
	}

	var resposta dto.RespostaErro
	if err := json.Unmarshal(registrador.Body.Bytes(), &resposta); err != nil {
		t.Fatalf("corpo não é um envelope de erro válido: %v", err)
	}
	if resposta.Erro.Codigo != http.StatusBadRequest {
		t.Errorf("código do envelope %d, esperado %d", resposta.Erro.Codigo, http.StatusBadRequest)
	}
	if resposta.Erro.Mensagem == "" {
		t.Error("mensagem do envelope vazia")
	}
}

func TestCorpoJSONRejeitaCampoDesconhecido(t *testing.T) {
	requisicao := httptest.NewRequest(http.MethodPost, "/leads", strings.NewReader(`{"nome":"x","desconhecido":1}`))
	registrador := httptest.NewRecorder()

	var destino struct {
		Nome string `json:"nome"`
	}

	if corpoJSON(registrador, requisicao, &destino) {
		t.Fatal("esperava rejeição de campo desconhecido")
	}
	if registrador.Code != http.StatusBadRequest {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusBadRequest)
	}
}

func TestCorpoJSONRejeitaCorpoInvalido(t *testing.T) {
	requisicao := httptest.NewRequest(http.MethodPost, "/leads", strings.NewReader(`{`))
	registrador := httptest.NewRecorder()

	var destino struct {
		Nome string `json:"nome"`
	}

	if corpoJSON(registrador, requisicao, &destino) {
		t.Fatal("esperava rejeição de corpo inválido")
	}
	if registrador.Code != http.StatusBadRequest {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusBadRequest)
	}
}

func TestParametroUUIDInvalido(t *testing.T) {
	requisicao := httptest.NewRequest(http.MethodGet, "/leads/abc", nil)
	requisicao.SetPathValue("id", "abc")
	registrador := httptest.NewRecorder()

	if _, ok := parametroUUID(registrador, requisicao, "id"); ok {
		t.Fatal("esperava rejeição de parâmetro não UUID")
	}
	if registrador.Code != http.StatusBadRequest {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusBadRequest)
	}
}

func TestParametroUUIDValido(t *testing.T) {
	requisicao := httptest.NewRequest(http.MethodGet, "/leads/11111111-1111-1111-1111-111111111111", nil)
	requisicao.SetPathValue("id", "11111111-1111-1111-1111-111111111111")
	registrador := httptest.NewRecorder()

	id, ok := parametroUUID(registrador, requisicao, "id")
	if !ok {
		t.Fatal("esperava aceitação do parâmetro UUID")
	}
	if id.String() != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("id %q inesperado", id)
	}
}
