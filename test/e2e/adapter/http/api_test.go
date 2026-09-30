//go:build e2e

// Package http testa a aplicação montada inteira: requisição HTTP entrando,
// camada de rotas, handlers, usecases e Postgres saindo. É o teste mais
// distante do código e o mais próximo do que o usuário final vê.
//
// Roda atrás da tag `e2e` porque depende do banco e da camada HTTP:
//
//	go test -tags=e2e ./test/...
package http

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	leaddto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/lead"
	"github.com/CunhazadanoDale/paineladministrativo.git/test/helpers"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// statusSaude é o conteúdo devolvido por GET /health.
type statusSaude struct {
	Status string `json:"status"`
}

// servidorDoTeste sobe a aplicação com um banco novo para o teste.
func servidorDoTeste(t *testing.T) (*httptest.Server, *sqlx.DB) {
	t.Helper()

	banco := helpers.BancoDoTeste(t)

	return helpers.NovoServidor(t, banco), banco
}

// envia faz uma requisição ao servidor de teste e devolve a resposta.
func envia(t *testing.T, servidor *httptest.Server, metodo, caminho string, corpo any) *http.Response {
	t.Helper()

	var leitor io.Reader
	if corpo != nil {
		bruto, err := json.Marshal(corpo)
		if err != nil {
			t.Fatalf("não consegui serializar o corpo da requisição: %v", err)
		}
		leitor = bytes.NewReader(bruto)
	}

	requisicao, err := http.NewRequest(metodo, servidor.URL+caminho, leitor)
	if err != nil {
		t.Fatalf("não consegui montar a requisição %s %s: %v", metodo, caminho, err)
	}
	if corpo != nil {
		requisicao.Header.Set("Content-Type", "application/json")
	}

	resposta, err := servidor.Client().Do(requisicao)
	if err != nil {
		t.Fatalf("requisição %s %s falhou: %v", metodo, caminho, err)
	}
	t.Cleanup(func() {
		_ = resposta.Body.Close()
	})

	return resposta
}

// conferirStatus falha o teste se o status HTTP for diferente do esperado.
func conferirStatus(t *testing.T, resposta *http.Response, esperado int) {
	t.Helper()

	if resposta.StatusCode != esperado {
		t.Fatalf("status %d, esperado %d", resposta.StatusCode, esperado)
	}
}

// decodificarEnvelope lê um corpo no formato {"dados": ...}.
func decodificarEnvelope[T any](t *testing.T, resposta *http.Response) dto.Resposta[T] {
	t.Helper()

	var conteudo dto.Resposta[T]
	decodificar(t, resposta, &conteudo)

	return conteudo
}

// decodificarPagina lê um corpo paginado ({"dados": [...], "pagina": ...}).
func decodificarPagina[T any](t *testing.T, resposta *http.Response) dto.Paginado[T] {
	t.Helper()

	var conteudo dto.Paginado[T]
	decodificar(t, resposta, &conteudo)

	return conteudo
}

// decodificarErro lê o envelope de erro {"erro": {"codigo": ..., ...}}.
func decodificarErro(t *testing.T, resposta *http.Response) dto.RespostaErro {
	t.Helper()

	var conteudo dto.RespostaErro
	decodificar(t, resposta, &conteudo)

	return conteudo
}

func decodificar(t *testing.T, resposta *http.Response, destino any) {
	t.Helper()

	if err := json.NewDecoder(resposta.Body).Decode(destino); err != nil {
		t.Fatalf("corpo da resposta não é JSON válido: %v", err)
	}
}

// criarFunil cria um funil pela API e devolve o id.
func criarFunil(t *testing.T, servidor *httptest.Server, nome string) uuid.UUID {
	t.Helper()

	resposta := envia(t, servidor, http.MethodPost, "/api/v1/funils", map[string]string{"nome": nome})
	conferirStatus(t, resposta, http.StatusCreated)

	return decodificarEnvelope[leaddto.FunilResponse](t, resposta).Dados.FunilID
}

// criarEtapa cria uma etapa pela API e devolve o id.
func criarEtapa(t *testing.T, servidor *httptest.Server, funilID uuid.UUID, nome string) uuid.UUID {
	t.Helper()

	resposta := envia(t, servidor, http.MethodPost, "/api/v1/etapas", map[string]any{
		"nome":     nome,
		"funil_id": funilID,
	})
	conferirStatus(t, resposta, http.StatusCreated)

	return decodificarEnvelope[leaddto.EtapaResponse](t, resposta).Dados.EtapaID
}

// criarLead cria um lead pela API e devolve o id.
func criarLead(t *testing.T, servidor *httptest.Server, etapaID uuid.UUID, nome string) uuid.UUID {
	t.Helper()

	resposta := envia(t, servidor, http.MethodPost, "/api/v1/leads", map[string]any{
		"nome":     nome,
		"email":    "ana@exemplo.com",
		"etapa_id": etapaID,
	})
	conferirStatus(t, resposta, http.StatusCreated)

	return decodificarEnvelope[leaddto.LeadResponse](t, resposta).Dados.ID
}

func TestHealth(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	resposta := envia(t, servidor, http.MethodGet, "/health", nil)
	conferirStatus(t, resposta, http.StatusOK)

	if status := decodificarEnvelope[statusSaude](t, resposta).Dados.Status; status != "ok" {
		t.Errorf("status %q, esperado %q", status, "ok")
	}
}

func TestHealthDoBanco(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	resposta := envia(t, servidor, http.MethodGet, "/health/db", nil)
	conferirStatus(t, resposta, http.StatusOK)
}

func TestCriarLeadValidoRespondeCreated(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	funil := criarFunil(t, servidor, "Funil E2E")
	etapa := criarEtapa(t, servidor, funil, "Novo")

	resposta := envia(t, servidor, http.MethodPost, "/api/v1/leads", map[string]any{
		"nome":     "Ana Souza",
		"email":    "ana@exemplo.com",
		"telefone": "11999999999",
		"origem":   "site",
		"etapa_id": etapa,
	})
	conferirStatus(t, resposta, http.StatusCreated)

	criado := decodificarEnvelope[leaddto.LeadResponse](t, resposta).Dados
	if criado.ID == uuid.Nil {
		t.Error("resposta veio sem id")
	}
	if criado.Nome != "Ana Souza" {
		t.Errorf("nome %q, esperado %q", criado.Nome, "Ana Souza")
	}
	if criado.EtapaID != etapa {
		t.Errorf("etapa %s, esperada %s", criado.EtapaID, etapa)
	}
	if !criado.Ativo {
		t.Error("lead deveria ter sido criado ativo")
	}
}

func TestCriarLeadSemNomeRespondeErroDeValidacao(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	funil := criarFunil(t, servidor, "Funil E2E")
	etapa := criarEtapa(t, servidor, funil, "Novo")

	resposta := envia(t, servidor, http.MethodPost, "/api/v1/leads", map[string]any{
		"etapa_id": etapa,
	})
	conferirStatus(t, resposta, http.StatusBadRequest)

	erro := decodificarErro(t, resposta)
	if erro.Erro.Codigo != http.StatusBadRequest {
		t.Errorf("código do envelope %d, esperado %d", erro.Erro.Codigo, http.StatusBadRequest)
	}
	if !strings.Contains(erro.Erro.Mensagem, "nome") {
		t.Errorf("mensagem %q deveria citar o campo nome", erro.Erro.Mensagem)
	}
}

func TestCriarLeadComCampoDesconhecidoRespondeErro(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	funil := criarFunil(t, servidor, "Funil E2E")
	etapa := criarEtapa(t, servidor, funil, "Novo")

	resposta := envia(t, servidor, http.MethodPost, "/api/v1/leads", map[string]any{
		"nome":              "Ana Souza",
		"etapa_id":          etapa,
		"campo_inexistente": "valor",
	})
	conferirStatus(t, resposta, http.StatusBadRequest)

	erro := decodificarErro(t, resposta)
	if !strings.Contains(erro.Erro.Mensagem, "corpo da requisição inválido") {
		t.Errorf("mensagem %q inesperada", erro.Erro.Mensagem)
	}
}

func TestObterLeadInexistenteRespondeNotFound(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	resposta := envia(t, servidor, http.MethodGet, "/api/v1/leads/"+uuid.NewString(), nil)
	conferirStatus(t, resposta, http.StatusNotFound)

	erro := decodificarErro(t, resposta)
	if erro.Erro.Codigo != http.StatusNotFound {
		t.Errorf("código do envelope %d, esperado %d", erro.Erro.Codigo, http.StatusNotFound)
	}
}

func TestRotaDesconhecidaRespondeNotFoundComEnvelope(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	resposta := envia(t, servidor, http.MethodGet, "/rota-que-nao-existe", nil)
	conferirStatus(t, resposta, http.StatusNotFound)

	erro := decodificarErro(t, resposta)
	if erro.Erro.Mensagem == "" {
		t.Error("resposta 404 veio sem mensagem no envelope")
	}
}

func TestListarLeadsDevolvePaginacao(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	funil := criarFunil(t, servidor, "Funil E2E")
	etapa := criarEtapa(t, servidor, funil, "Novo")
	criarLead(t, servidor, etapa, "Ana Souza")
	criarLead(t, servidor, etapa, "Bruno Lima")

	resposta := envia(t, servidor, http.MethodGet, "/api/v1/leads", nil)
	conferirStatus(t, resposta, http.StatusOK)

	listagem := decodificarPagina[leaddto.LeadResponse](t, resposta)
	if len(listagem.Dados) != 2 {
		t.Errorf("%d leads listados, esperado 2", len(listagem.Dados))
	}
	if listagem.Pagina != 1 {
		t.Errorf("página %d, esperada 1", listagem.Pagina)
	}
	if listagem.Tamanho != 20 {
		t.Errorf("tamanho %d, esperado 20 (padrão)", listagem.Tamanho)
	}
}

func TestMoverLeadEntreEtapasRegistraHistorico(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	funil := criarFunil(t, servidor, "Funil E2E")
	etapaNovo := criarEtapa(t, servidor, funil, "Novo")
	etapaProposta := criarEtapa(t, servidor, funil, "Proposta")
	leadID := criarLead(t, servidor, etapaNovo, "Ana Souza")

	resposta := envia(t, servidor, http.MethodPatch, "/api/v1/leads/"+leadID.String()+"/etapa", map[string]any{
		"etapa_id": etapaProposta,
	})
	conferirStatus(t, resposta, http.StatusOK)

	movido := decodificarEnvelope[leaddto.LeadResponse](t, resposta).Dados
	if movido.EtapaID != etapaProposta {
		t.Errorf("etapa do lead %s, esperada %s", movido.EtapaID, etapaProposta)
	}

	resposta = envia(t, servidor, http.MethodGet, "/api/v1/leads/"+leadID.String()+"/historico", nil)
	conferirStatus(t, resposta, http.StatusOK)

	historico := decodificarPagina[leaddto.LeadHistoricoResponse](t, resposta)
	if len(historico.Dados) != 1 {
		t.Fatalf("%d registros de histórico, esperado 1", len(historico.Dados))
	}

	registro := historico.Dados[0]
	if registro.LeadID != leadID {
		t.Errorf("histórico do lead %s, esperado %s", registro.LeadID, leadID)
	}
	if registro.EtapaAnteriorID != etapaNovo || registro.EtapaAtualID != etapaProposta {
		t.Errorf(
			"movimentação %s -> %s, esperada %s -> %s",
			registro.EtapaAnteriorID,
			registro.EtapaAtualID,
			etapaNovo,
			etapaProposta,
		)
	}
}
