package solicitacao

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
		{domain.ErroValidacao("valor deve ser maior que zero"), http.StatusBadRequest, "valor deve ser maior que zero"},
		{domain.ErroPermissao("perfil sem permissão para aprovar"), http.StatusForbidden, "perfil sem permissão para aprovar"},
		{domain.ErrNotFound, http.StatusNotFound, "registro não encontrado"},
		{domain.ErroConflito("solicitação já paga"), http.StatusConflict, "solicitação já paga"},
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

	responderErro(registrador, domain.ErroConflito("solicitação alterada por outra operação"))

	if registrador.Code != http.StatusConflict {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusConflict)
	}
	if conteudo := registrador.Header().Get("Content-Type"); conteudo != "application/json; charset=utf-8" {
		t.Errorf("content-type %q inesperado", conteudo)
	}

	var resposta dto.RespostaErro
	if err := json.Unmarshal(registrador.Body.Bytes(), &resposta); err != nil {
		t.Fatalf("corpo não é um envelope de erro válido: %v", err)
	}
	if resposta.Erro.Codigo != http.StatusConflict || resposta.Erro.Mensagem == "" {
		t.Errorf("envelope %+v, esperado código 409 com mensagem", resposta.Erro)
	}
}
