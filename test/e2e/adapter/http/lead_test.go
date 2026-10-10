//go:build e2e

package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	leaddto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/lead"
	usuariosdto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/usuarios"
	"github.com/google/uuid"
)

func tokenComercial(t *testing.T, servidor *httptest.Server) string {
	t.Helper()

	resposta := envia(t, servidor, http.MethodPost, "/api/v1/cargos", map[string]any{
		"nome":      "Vendedor",
		"descricao": "Cargo comercial criado no teste",
		"comercial": true,
	})
	conferirStatus(t, resposta, http.StatusCreated)
	cargo := decodificarEnvelope[usuariosdto.CargoResponse](t, resposta).Dados
	if !cargo.Comercial {
		t.Fatal("cargo criado sem o perfil comercial")
	}

	criarUsuario(t, servidor, "Vera Nunes", "vera@exemplo.com", "senhaForte123", cargo.ID)

	sessao := autenticar(t, servidor, "vera@exemplo.com", "senhaForte123")
	if !sessao.Comercial || sessao.Administrador {
		t.Fatalf("sessão comercial=%v administrador=%v, esperado só comercial", sessao.Comercial, sessao.Administrador)
	}

	return sessao.Token
}

func TestPerfilComercialOperaLeadsMasNaoAEstruturaDoFunil(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	funilID := criarFunil(t, servidor, "Vendas")
	origem := criarEtapa(t, servidor, funilID, "Contato")
	destino := criarEtapa(t, servidor, funilID, "Proposta")
	token := tokenComercial(t, servidor)

	resposta := enviaComToken(t, servidor, http.MethodPost, "/api/v1/leads", map[string]any{
		"nome":     "Construtora Alfa",
		"etapa_id": origem,
	}, token)
	conferirStatus(t, resposta, http.StatusCreated)
	leadID := decodificarEnvelope[leaddto.LeadResponse](t, resposta).Dados.ID

	resposta = enviaComToken(t, servidor, http.MethodPatch, "/api/v1/leads/"+leadID.String()+"/etapa", map[string]any{
		"etapa_id": destino,
	}, token)
	conferirStatus(t, resposta, http.StatusOK)

	for _, rota := range []string{"/api/v1/leads", "/api/v1/funils", "/api/v1/funils/" + funilID.String() + "/etapas", "/api/v1/leads/" + leadID.String() + "/historico"} {
		resposta = enviaComToken(t, servidor, http.MethodGet, rota, nil, token)
		conferirStatus(t, resposta, http.StatusOK)
	}

	resposta = enviaComToken(t, servidor, http.MethodPost, "/api/v1/funils", map[string]string{"nome": "Paralelo"}, token)
	conferirStatus(t, resposta, http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodPost, "/api/v1/etapas", map[string]any{"nome": "Extra", "funil_id": funilID}, token)
	conferirStatus(t, resposta, http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodDelete, "/api/v1/etapas/"+destino.String(), nil, token)
	conferirStatus(t, resposta, http.StatusForbidden)
}

func TestUsuarioSemPerfilComercialNaoAcessaLeads(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	cargoID := criarCargo(t, servidor, "Almoxarife", false)
	criarUsuario(t, servidor, "Davi Rocha", "davi@exemplo.com", "senhaForte123", cargoID)
	token := autenticar(t, servidor, "davi@exemplo.com", "senhaForte123").Token

	for _, rota := range []string{"/api/v1/leads", "/api/v1/funils"} {
		resposta := enviaComToken(t, servidor, http.MethodGet, rota, nil, token)
		conferirStatus(t, resposta, http.StatusForbidden)
	}
}

func TestAtualizarLeadExigeAtivoENaoTrocaAEtapa(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	funilID := criarFunil(t, servidor, "Vendas")
	origem := criarEtapa(t, servidor, funilID, "Contato")
	outra := criarEtapa(t, servidor, funilID, "Proposta")
	leadID := criarLead(t, servidor, origem, "Ana Souza")
	caminho := "/api/v1/leads/" + leadID.String()

	resposta := envia(t, servidor, http.MethodPut, caminho, map[string]any{"nome": "Ana Lima"})
	conferirStatus(t, resposta, http.StatusBadRequest)

	resposta = envia(t, servidor, http.MethodPut, caminho, map[string]any{"nome": "Ana Lima", "ativo": true, "etapa_id": outra})
	conferirStatus(t, resposta, http.StatusBadRequest)

	resposta = envia(t, servidor, http.MethodPut, caminho, map[string]any{"nome": "Ana Lima", "ativo": true, "etapa_id": origem})
	conferirStatus(t, resposta, http.StatusOK)

	salvo := decodificarEnvelope[leaddto.LeadResponse](t, resposta).Dados
	if salvo.Nome != "Ana Lima" || salvo.EtapaID != origem || !salvo.Ativo {
		t.Errorf("lead salvo %+v, esperado nome novo na etapa original e ativo", salvo)
	}
}

func TestMoverLeadValidaEtapaDeDestino(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	funilID := criarFunil(t, servidor, "Vendas")
	origem := criarEtapa(t, servidor, funilID, "Contato")
	outroFunil := criarFunil(t, servidor, "Pós-venda")
	etapaDeOutroFunil := criarEtapa(t, servidor, outroFunil, "Suporte")
	leadID := criarLead(t, servidor, origem, "Ana Souza")
	caminho := "/api/v1/leads/" + leadID.String() + "/etapa"

	for _, destino := range []uuid.UUID{etapaDeOutroFunil, uuid.New()} {
		resposta := envia(t, servidor, http.MethodPatch, caminho, map[string]any{"etapa_id": destino})
		conferirStatus(t, resposta, http.StatusBadRequest)
	}
}

func TestHistoricoDoLeadNaoAceitaGravacaoManual(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	funilID := criarFunil(t, servidor, "Vendas")
	origem := criarEtapa(t, servidor, funilID, "Contato")
	destino := criarEtapa(t, servidor, funilID, "Proposta")
	leadID := criarLead(t, servidor, origem, "Ana Souza")

	resposta := envia(t, servidor, http.MethodPost, "/api/v1/leads/"+leadID.String()+"/historico", map[string]any{
		"etapa_anterior_id": origem,
		"etapa_atual_id":    destino,
	})
	conferirStatus(t, resposta, http.StatusMethodNotAllowed)
}

func TestCriarLeadComDadosForaDoLimiteRespondeErroDeValidacao(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	funilID := criarFunil(t, servidor, "Vendas")
	etapaID := criarEtapa(t, servidor, funilID, "Contato")

	casos := []map[string]any{
		{"nome": "Ana", "email": "sem-arroba", "etapa_id": etapaID},
		{"nome": "Ana", "telefone": "1234567890123456789012345678901234567890123", "etapa_id": etapaID},
		{"nome": "Ana", "etapa_id": uuid.New()},
	}

	for _, corpo := range casos {
		resposta := envia(t, servidor, http.MethodPost, "/api/v1/leads", corpo)
		conferirStatus(t, resposta, http.StatusBadRequest)
	}
}
