package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func muxDoTeste(t *testing.T) *http.ServeMux {
	t.Helper()

	return novasRotas(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
}

func TestNewRouterAplicaCORS(t *testing.T) {
	requisicao := httptest.NewRequest(http.MethodOptions, "/api/v1/leads", nil)
	requisicao.Header.Set("Origin", "http://localhost:5173")
	requisicao.Header.Set("Access-Control-Request-Method", "POST")
	registrador := httptest.NewRecorder()

	NewRouter(nil, []string{"http://localhost:5173"}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil).ServeHTTP(registrador, requisicao)

	if registrador.Code != http.StatusNoContent {
		t.Errorf("status %d, esperado %d", registrador.Code, http.StatusNoContent)
	}
	if origem := registrador.Header().Get("Access-Control-Allow-Origin"); origem != "http://localhost:5173" {
		t.Errorf("Access-Control-Allow-Origin %q, esperado %q", origem, "http://localhost:5173")
	}
}

func TestRotasRegistradasSemConflito(t *testing.T) {
	mux := muxDoTeste(t)

	casos := []struct {
		requisicao string
		padrao     string
	}{
		{"GET /health", "GET /health"},
		{"GET /health/db", "GET /health/db"},
		{"POST /api/v1/leads", "POST /api/v1/leads"},
		{"GET /api/v1/leads", "GET /api/v1/leads"},
		{"GET /api/v1/leads/11111111-1111-1111-1111-111111111111", "GET /api/v1/leads/{id}"},
		{"PUT /api/v1/leads/11111111-1111-1111-1111-111111111111", "PUT /api/v1/leads/{id}"},
		{"DELETE /api/v1/leads/11111111-1111-1111-1111-111111111111", "DELETE /api/v1/leads/{id}"},
		{"PATCH /api/v1/leads/11111111-1111-1111-1111-111111111111/etapa", "PATCH /api/v1/leads/{id}/etapa"},
		{"GET /api/v1/funils/22222222-2222-2222-2222-222222222222/leads", "GET /api/v1/funils/{funil_id}/leads"},
		{"GET /api/v1/etapas/33333333-3333-3333-3333-333333333333/leads", "GET /api/v1/etapas/{etapa_id}/leads"},
		{"GET /api/v1/funils/22222222-2222-2222-2222-222222222222/leads/contagem", "GET /api/v1/funils/{funil_id}/leads/contagem"},
		{"GET /api/v1/etapas/33333333-3333-3333-3333-333333333333/leads/contagem", "GET /api/v1/etapas/{etapa_id}/leads/contagem"},
		{"GET /api/v1/leads/11111111-1111-1111-1111-111111111111/historico", "GET /api/v1/leads/{lead_id}/historico"},
		{"POST /api/v1/leads/11111111-1111-1111-1111-111111111111/historico", "POST /api/v1/leads/{lead_id}/historico"},
		{"POST /api/v1/funils", "POST /api/v1/funils"},
		{"GET /api/v1/funils", "GET /api/v1/funils"},
		{"GET /api/v1/funils/22222222-2222-2222-2222-222222222222", "GET /api/v1/funils/{funil_id}"},
		{"PUT /api/v1/funils/22222222-2222-2222-2222-222222222222", "PUT /api/v1/funils/{funil_id}"},
		{"DELETE /api/v1/funils/22222222-2222-2222-2222-222222222222", "DELETE /api/v1/funils/{funil_id}"},
		{"GET /api/v1/funils/22222222-2222-2222-2222-222222222222/etapas", "GET /api/v1/funils/{funil_id}/etapas"},
		{"PUT /api/v1/funils/22222222-2222-2222-2222-222222222222/etapas/ordem", "PUT /api/v1/funils/{funil_id}/etapas/ordem"},
		{"POST /api/v1/etapas", "POST /api/v1/etapas"},
		{"GET /api/v1/etapas/33333333-3333-3333-3333-333333333333", "GET /api/v1/etapas/{etapa_id}"},
		{"PUT /api/v1/etapas/33333333-3333-3333-3333-333333333333", "PUT /api/v1/etapas/{etapa_id}"},
		{"DELETE /api/v1/etapas/33333333-3333-3333-3333-333333333333", "DELETE /api/v1/etapas/{etapa_id}"},
		{"GET /api/v1/etapas/33333333-3333-3333-3333-333333333333/proxima", "GET /api/v1/etapas/{etapa_id}/proxima"},
		{"GET /api/v1/etapas/33333333-3333-3333-3333-333333333333/anterior", "GET /api/v1/etapas/{etapa_id}/anterior"},
		{"POST /api/v1/cargos", "POST /api/v1/cargos"},
		{"GET /api/v1/cargos", "GET /api/v1/cargos"},
		{"GET /api/v1/cargos/busca?nome=Gerente", "GET /api/v1/cargos/busca"},
		{"GET /api/v1/cargos/44444444-4444-4444-4444-444444444444", "GET /api/v1/cargos/{id}"},
		{"PUT /api/v1/cargos/44444444-4444-4444-4444-444444444444", "PUT /api/v1/cargos/{id}"},
		{"DELETE /api/v1/cargos/44444444-4444-4444-4444-444444444444", "DELETE /api/v1/cargos/{id}"},
		{"POST /api/v1/usuarios", "POST /api/v1/usuarios"},
		{"GET /api/v1/usuarios", "GET /api/v1/usuarios"},
		{"POST /api/v1/usuarios/autenticar", "POST /api/v1/usuarios/autenticar"},
		{"GET /api/v1/usuarios/55555555-5555-5555-5555-555555555555", "GET /api/v1/usuarios/{id}"},
		{"PUT /api/v1/usuarios/55555555-5555-5555-5555-555555555555", "PUT /api/v1/usuarios/{id}"},
		{"DELETE /api/v1/usuarios/55555555-5555-5555-5555-555555555555", "DELETE /api/v1/usuarios/{id}"},
		{"POST /api/v1/usuarios/55555555-5555-5555-5555-555555555555/senha", "POST /api/v1/usuarios/{id}/senha"},
		{"PATCH /api/v1/usuarios/55555555-5555-5555-5555-555555555555/ativar", "PATCH /api/v1/usuarios/{id}/ativar"},
		{"PATCH /api/v1/usuarios/55555555-5555-5555-5555-555555555555/desativar", "PATCH /api/v1/usuarios/{id}/desativar"},
		{"POST /api/v1/solicitacoes", "POST /api/v1/solicitacoes"},
		{"GET /api/v1/solicitacoes", "GET /api/v1/solicitacoes"},
		{"GET /api/v1/solicitacoes/66666666-6666-6666-6666-666666666666", "GET /api/v1/solicitacoes/{id}"},
		{"POST /api/v1/solicitacoes/66666666-6666-6666-6666-666666666666/aprovar", "POST /api/v1/solicitacoes/{id}/aprovar"},
		{"POST /api/v1/solicitacoes/66666666-6666-6666-6666-666666666666/rejeitar", "POST /api/v1/solicitacoes/{id}/rejeitar"},
		{"POST /api/v1/solicitacoes/66666666-6666-6666-6666-666666666666/cancelar", "POST /api/v1/solicitacoes/{id}/cancelar"},
		{"POST /api/v1/solicitacoes/66666666-6666-6666-6666-666666666666/pagamento", "POST /api/v1/solicitacoes/{id}/pagamento"},
		{"GET /api/v1/solicitacoes/66666666-6666-6666-6666-666666666666/pagamento", "GET /api/v1/solicitacoes/{id}/pagamento"},
		{"GET /api/v1/solicitacoes/66666666-6666-6666-6666-666666666666/historico", "GET /api/v1/solicitacoes/{id}/historico"},
		{"GET /api/v1/solicitacoes/66666666-6666-6666-6666-666666666666/arquivos", "GET /api/v1/solicitacoes/{id}/arquivos"},
		{"POST /api/v1/arquivos", "POST /api/v1/arquivos"},
		{"GET /api/v1/arquivos", "GET /api/v1/arquivos"},
		{"GET /api/v1/arquivos/77777777-7777-7777-7777-777777777777", "GET /api/v1/arquivos/{id}"},
		{"DELETE /api/v1/arquivos/77777777-7777-7777-7777-777777777777", "DELETE /api/v1/arquivos/{id}"},
		{"POST /api/v1/aprovadores", "POST /api/v1/aprovadores"},
		{"GET /api/v1/aprovadores", "GET /api/v1/aprovadores"},
		{"DELETE /api/v1/aprovadores/88888888-8888-8888-8888-888888888888", "DELETE /api/v1/aprovadores/{id}"},
		{"POST /api/v1/categorias", "POST /api/v1/categorias"},
		{"GET /api/v1/categorias", "GET /api/v1/categorias"},
		{"GET /api/v1/categorias/99999999-9999-9999-9999-999999999999", "GET /api/v1/categorias/{id}"},
		{"PUT /api/v1/categorias/99999999-9999-9999-9999-999999999999", "PUT /api/v1/categorias/{id}"},
		{"PATCH /api/v1/categorias/99999999-9999-9999-9999-999999999999/ativar", "PATCH /api/v1/categorias/{id}/ativar"},
		{"PATCH /api/v1/categorias/99999999-9999-9999-9999-999999999999/desativar", "PATCH /api/v1/categorias/{id}/desativar"},
		{"POST /api/v1/produtos", "POST /api/v1/produtos"},
		{"GET /api/v1/produtos", "GET /api/v1/produtos"},
		{"GET /api/v1/produtos/99999999-9999-9999-9999-999999999999", "GET /api/v1/produtos/{id}"},
		{"GET /api/v1/produtos/99999999-9999-9999-9999-999999999999/saldo", "GET /api/v1/produtos/{id}/saldo"},
		{"PUT /api/v1/produtos/99999999-9999-9999-9999-999999999999", "PUT /api/v1/produtos/{id}"},
		{"PATCH /api/v1/produtos/99999999-9999-9999-9999-999999999999/ativar", "PATCH /api/v1/produtos/{id}/ativar"},
		{"PATCH /api/v1/produtos/99999999-9999-9999-9999-999999999999/desativar", "PATCH /api/v1/produtos/{id}/desativar"},
		{"PATCH /api/v1/produtos/99999999-9999-9999-9999-999999999999/destaque", "PATCH /api/v1/produtos/{id}/destaque"},
		{"POST /api/v1/produtos/99999999-9999-9999-9999-999999999999/movimentos", "POST /api/v1/produtos/{id}/movimentos"},
		{"GET /api/v1/produtos/99999999-9999-9999-9999-999999999999/movimentos", "GET /api/v1/produtos/{id}/movimentos"},
	}

	for _, caso := range casos {
		metodo, caminho, ok := strings.Cut(caso.requisicao, " ")
		if !ok {
			t.Fatalf("requisição inválida no teste: %q", caso.requisicao)
		}

		requisicao, err := http.NewRequest(metodo, caminho, nil)
		if err != nil {
			t.Fatalf("não foi possível montar a requisição %q: %v", caso.requisicao, err)
		}

		_, padrao := mux.Handler(requisicao)
		if padrao != caso.padrao {
			t.Errorf("requisição %q resolvida como %q, esperado %q", caso.requisicao, padrao, caso.padrao)
		}
	}
}

func TestRotaNaoRegistradaCaiNoCoringa(t *testing.T) {
	mux := muxDoTeste(t)

	requisicao, err := http.NewRequest(http.MethodGet, "/nao-existe", nil)
	if err != nil {
		t.Fatalf("não foi possível montar a requisição: %v", err)
	}

	_, padrao := mux.Handler(requisicao)
	if padrao != "GET /" {
		t.Errorf("esperava o coringa GET / para /nao-existe, veio %q", padrao)
	}
}
