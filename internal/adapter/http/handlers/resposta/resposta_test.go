package resposta

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

func TestMapearErro(t *testing.T) {
	casos := []struct {
		origem   error
		status   int
		mensagem string
	}{
		{domain.ErroValidacao("nome do lead é obrigatório"), http.StatusBadRequest, "erro de validação: nome do lead é obrigatório"},
		{domain.ErroValidacao("valor deve ser maior que zero"), http.StatusBadRequest, "erro de validação: valor deve ser maior que zero"},
		{domain.ErroPermissao("perfil sem permissão para registrar pagamento"), http.StatusForbidden, "perfil sem permissão para registrar pagamento"},
		{domain.ErrPermissao, http.StatusForbidden, "perfil sem permissão para esta operação"},
		{domain.ErroNaoEncontrado("lead não encontrado"), http.StatusNotFound, "registro não encontrado: lead não encontrado"},
		{domain.ErroNaoEncontrado("pagamento não encontrado"), http.StatusNotFound, "registro não encontrado: pagamento não encontrado"},
		{domain.ErrNotFound, http.StatusNotFound, "registro não encontrado"},
		{domain.ErroConflito("solicitação já paga"), http.StatusConflict, "solicitação já paga"},
		{domain.ErrConflito, http.StatusConflict, "operação em conflito com o estado atual"},
		{errors.New("banco indisponível"), http.StatusInternalServerError, "erro interno do servidor"},
	}

	for _, caso := range casos {
		status, mensagem := MapearErro(caso.origem)
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

	ResponderErro(registrador, domain.ErroValidacao("nome do lead é obrigatório"))

	if registrador.Code != http.StatusBadRequest {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusBadRequest)
	}
	if conteudo := registrador.Header().Get("Content-Type"); conteudo != "application/json; charset=utf-8" {
		t.Errorf("content-type %q inesperado", conteudo)
	}

	var envelope dto.RespostaErro
	if err := json.Unmarshal(registrador.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("corpo não é um envelope de erro válido: %v", err)
	}
	if envelope.Erro.Codigo != http.StatusBadRequest {
		t.Errorf("código do envelope %d, esperado %d", envelope.Erro.Codigo, http.StatusBadRequest)
	}
	if envelope.Erro.Mensagem == "" {
		t.Error("mensagem do envelope vazia")
	}
}

func TestCorpoJSONRejeitaCampoDesconhecido(t *testing.T) {
	requisicao := httptest.NewRequest(http.MethodPost, "/leads", strings.NewReader(`{"nome":"x","desconhecido":1}`))
	registrador := httptest.NewRecorder()

	var destino struct {
		Nome string `json:"nome"`
	}

	if CorpoJSON(registrador, requisicao, &destino) {
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

	if CorpoJSON(registrador, requisicao, &destino) {
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

	if _, ok := ParametroUUID(registrador, requisicao, "id"); ok {
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

	id, ok := ParametroUUID(registrador, requisicao, "id")
	if !ok {
		t.Fatal("esperava aceitação do parâmetro UUID")
	}
	if id.String() != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("id %q inesperado", id)
	}
}

func TestUsuarioDoContextoSemUsuarioResponde401(t *testing.T) {
	requisicao := httptest.NewRequest(http.MethodGet, "/api/v1/arquivos", nil)
	registrador := httptest.NewRecorder()

	if _, ok := UsuarioDoContexto(registrador, requisicao); ok {
		t.Fatal("esperava rejeição sem usuário no contexto")
	}
	if registrador.Code != http.StatusUnauthorized {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusUnauthorized)
	}
}

func TestConsultaPaginacaoNormaliza(t *testing.T) {
	requisicao := httptest.NewRequest(http.MethodGet, "/api/v1/leads?pagina=0&tamanho=9999", nil)

	filtro := ConsultaPaginacao(requisicao)
	if filtro.Page != 1 || filtro.Size != 100 {
		t.Errorf("filtro {pagina: %d, tamanho: %d}, esperado {pagina: 1, tamanho: 100}", filtro.Page, filtro.Size)
	}
}

func TestConsultaBooleana(t *testing.T) {
	casos := map[string]bool{
		"":      false,
		"true":  true,
		"1":     true,
		"false": false,
		"0":     false,
		"sim":   false,
	}

	for valor, esperado := range casos {
		requisicao := httptest.NewRequest(http.MethodGet, "/api/v1/usuarios?ativos="+valor, nil)
		if obtido := ConsultaBooleana(requisicao, "ativos"); obtido != esperado {
			t.Errorf("ConsultaBooleana(%q) = %v, esperado %v", valor, obtido, esperado)
		}
	}
}

func TestConsultaBooleanaOpcionalDistingueAusenteDeFalso(t *testing.T) {
	verdadeiro := true
	falso := false

	casos := []struct {
		valor    string
		esperado *bool
		erro     bool
	}{
		{valor: "", esperado: nil},
		{valor: "true", esperado: &verdadeiro},
		{valor: "false", esperado: &falso},
		{valor: "0", esperado: &falso},
		{valor: "sim", erro: true},
	}

	for _, caso := range casos {
		requisicao := httptest.NewRequest(http.MethodGet, "/api/v1/produtos?ativo="+caso.valor, nil)

		obtido, err := ConsultaBooleanaOpcional(requisicao, "ativo")
		if caso.erro {
			if err == nil {
				t.Errorf("valor %q deveria ser rejeitado", caso.valor)
			}
			continue
		}
		if err != nil {
			t.Errorf("valor %q devolveu erro: %v", caso.valor, err)
			continue
		}
		if (obtido == nil) != (caso.esperado == nil) {
			t.Errorf("valor %q = %v, esperado %v", caso.valor, obtido, caso.esperado)
			continue
		}
		if obtido != nil && *obtido != *caso.esperado {
			t.Errorf("valor %q = %v, esperado %v", caso.valor, *obtido, *caso.esperado)
		}
	}
}

func TestConsultaUUID(t *testing.T) {
	id := uuid.New()

	casos := map[string]*uuid.UUID{
		"":          nil,
		id.String(): &id,
	}

	for valor, esperado := range casos {
		requisicao := httptest.NewRequest(http.MethodGet, "/api/v1/produtos?categoria_id="+valor, nil)

		obtido, err := ConsultaUUID(requisicao, "categoria_id")
		if err != nil {
			t.Errorf("valor %q devolveu erro: %v", valor, err)
			continue
		}
		if (obtido == nil) != (esperado == nil) {
			t.Errorf("valor %q = %v, esperado %v", valor, obtido, esperado)
			continue
		}
		if obtido != nil && *obtido != *esperado {
			t.Errorf("valor %q = %s, esperado %s", valor, *obtido, *esperado)
		}
	}
}

func TestConsultaUUIDInvalidaRespondeErroDeValidacao(t *testing.T) {
	requisicao := httptest.NewRequest(http.MethodGet, "/api/v1/produtos?categoria_id=nao-e-uuid", nil)

	if _, err := ConsultaUUID(requisicao, "categoria_id"); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %v, esperado domain.ErrValidacao", err)
	}
}
