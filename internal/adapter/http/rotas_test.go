package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func muxDoTeste(t *testing.T) *http.ServeMux {
	t.Helper()

	return novasRotas(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
}

func TestNewRouterAplicaCORS(t *testing.T) {
	requisicao := httptest.NewRequest(http.MethodOptions, "/api/v1/leads", nil)
	requisicao.Header.Set("Origin", "http://localhost:5173")
	requisicao.Header.Set("Access-Control-Request-Method", "POST")
	registrador := httptest.NewRecorder()

	NewRouter(nil, []string{"http://localhost:5173"}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil).ServeHTTP(registrador, requisicao)

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
		{"PATCH /api/v1/solicitacoes/66666666-6666-6666-6666-666666666666/classificacao", "PATCH /api/v1/solicitacoes/{id}/classificacao"},
		{"PUT /api/v1/solicitacoes/66666666-6666-6666-6666-666666666666", "PUT /api/v1/solicitacoes/{id}"},
		{"POST /api/v1/solicitacoes/66666666-6666-6666-6666-666666666666/adiar", "POST /api/v1/solicitacoes/{id}/adiar"},
		{"POST /api/v1/solicitacoes/66666666-6666-6666-6666-666666666666/pagamento/estorno", "POST /api/v1/solicitacoes/{id}/pagamento/estorno"},
		{"GET /api/v1/solicitacoes/66666666-6666-6666-6666-666666666666/pagamentos", "GET /api/v1/solicitacoes/{id}/pagamentos"},
		{"POST /api/v1/despesas-recorrentes", "POST /api/v1/despesas-recorrentes"},
		{"GET /api/v1/despesas-recorrentes", "GET /api/v1/despesas-recorrentes"},
		{"POST /api/v1/despesas-recorrentes/gerar?competencia=2026-11", "POST /api/v1/despesas-recorrentes/gerar"},
		{"GET /api/v1/despesas-recorrentes/66666666-6666-6666-6666-666666666666", "GET /api/v1/despesas-recorrentes/{id}"},
		{"PUT /api/v1/despesas-recorrentes/66666666-6666-6666-6666-666666666666", "PUT /api/v1/despesas-recorrentes/{id}"},
		{"PATCH /api/v1/despesas-recorrentes/66666666-6666-6666-6666-666666666666/ativar", "PATCH /api/v1/despesas-recorrentes/{id}/ativar"},
		{"PATCH /api/v1/despesas-recorrentes/66666666-6666-6666-6666-666666666666/desativar", "PATCH /api/v1/despesas-recorrentes/{id}/desativar"},
		{"DELETE /api/v1/despesas-recorrentes/66666666-6666-6666-6666-666666666666", "DELETE /api/v1/despesas-recorrentes/{id}"},
		{"GET /api/v1/solicitacoes/pendencias", "GET /api/v1/solicitacoes/pendencias"},
		{"GET /api/v1/contas-a-pagar/resumo", "GET /api/v1/contas-a-pagar/resumo"},
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
		{"GET /api/v1/estoque/resumo", "GET /api/v1/estoque/resumo"},
		{"POST /api/v1/produtos/99999999-9999-9999-9999-999999999999/imagens", "POST /api/v1/produtos/{id}/imagens"},
		{"DELETE /api/v1/produtos/99999999-9999-9999-9999-999999999999/imagens/88888888-8888-8888-8888-888888888888", "DELETE /api/v1/produtos/{id}/imagens/{imagemId}"},
		{"GET /api/v1/publico/categorias", "GET /api/v1/publico/categorias"},
		{"GET /api/v1/publico/produtos", "GET /api/v1/publico/produtos"},
		{"GET /api/v1/publico/produtos/cimento-cp-ii-50", "GET /api/v1/publico/produtos/{slug}"},
		{"GET /api/v1/publico/destaques", "GET /api/v1/publico/destaques"},
		{"GET /api/v1/publico/imagens/88888888-8888-8888-8888-888888888888", "GET /api/v1/publico/imagens/{id}"},
		{"POST /api/v1/empresas", "POST /api/v1/empresas"},
		{"GET /api/v1/empresas", "GET /api/v1/empresas"},
		{"GET /api/v1/empresas/99999999-9999-9999-9999-999999999999", "GET /api/v1/empresas/{id}"},
		{"PUT /api/v1/empresas/99999999-9999-9999-9999-999999999999", "PUT /api/v1/empresas/{id}"},
		{"PATCH /api/v1/empresas/99999999-9999-9999-9999-999999999999/ativar", "PATCH /api/v1/empresas/{id}/ativar"},
		{"PATCH /api/v1/empresas/99999999-9999-9999-9999-999999999999/desativar", "PATCH /api/v1/empresas/{id}/desativar"},
		{"DELETE /api/v1/empresas/99999999-9999-9999-9999-999999999999", "DELETE /api/v1/empresas/{id}"},
		{"GET /api/v1/usuarios/44444444-4444-4444-4444-444444444444/empresas", "GET /api/v1/usuarios/{id}/empresas"},
		{"PUT /api/v1/usuarios/44444444-4444-4444-4444-444444444444/empresas", "PUT /api/v1/usuarios/{id}/empresas"},
		{"POST /api/v1/estoque/locais", "POST /api/v1/estoque/locais"},
		{"GET /api/v1/estoque/locais", "GET /api/v1/estoque/locais"},
		{"GET /api/v1/estoque/locais/88888888-8888-8888-8888-888888888888", "GET /api/v1/estoque/locais/{id}"},
		{"PUT /api/v1/estoque/locais/88888888-8888-8888-8888-888888888888", "PUT /api/v1/estoque/locais/{id}"},
		{"PATCH /api/v1/estoque/locais/88888888-8888-8888-8888-888888888888/ativar", "PATCH /api/v1/estoque/locais/{id}/ativar"},
		{"PATCH /api/v1/estoque/locais/88888888-8888-8888-8888-888888888888/desativar", "PATCH /api/v1/estoque/locais/{id}/desativar"},
		{"DELETE /api/v1/estoque/locais/88888888-8888-8888-8888-888888888888", "DELETE /api/v1/estoque/locais/{id}"},
		{"GET /api/v1/estoque/locais/88888888-8888-8888-8888-888888888888/saldos", "GET /api/v1/estoque/locais/{id}/saldos"},
		{"GET /api/v1/produtos/77777777-7777-7777-7777-777777777777/lotes", "GET /api/v1/produtos/{id}/lotes"},
		{"POST /api/v1/produtos/77777777-7777-7777-7777-777777777777/transferencias", "POST /api/v1/produtos/{id}/transferencias"},
		{"POST /api/v1/produtos/77777777-7777-7777-7777-777777777777/reservas", "POST /api/v1/produtos/{id}/reservas"},
		{"GET /api/v1/produtos/77777777-7777-7777-7777-777777777777/reservas", "GET /api/v1/produtos/{id}/reservas"},
		{"POST /api/v1/reservas/66666666-6666-6666-6666-666666666666/liberar", "POST /api/v1/reservas/{id}/liberar"},
		{"POST /api/v1/clientes", "POST /api/v1/clientes"},
		{"GET /api/v1/clientes", "GET /api/v1/clientes"},
		{"GET /api/v1/clientes/55555555-5555-5555-5555-555555555555", "GET /api/v1/clientes/{id}"},
		{"PUT /api/v1/clientes/55555555-5555-5555-5555-555555555555", "PUT /api/v1/clientes/{id}"},
		{"PATCH /api/v1/clientes/55555555-5555-5555-5555-555555555555/ativar", "PATCH /api/v1/clientes/{id}/ativar"},
		{"PATCH /api/v1/clientes/55555555-5555-5555-5555-555555555555/desativar", "PATCH /api/v1/clientes/{id}/desativar"},
		{"DELETE /api/v1/clientes/55555555-5555-5555-5555-555555555555", "DELETE /api/v1/clientes/{id}"},
		{"POST /api/v1/clientes/55555555-5555-5555-5555-555555555555/obras", "POST /api/v1/clientes/{id}/obras"},
		{"GET /api/v1/clientes/55555555-5555-5555-5555-555555555555/obras", "GET /api/v1/clientes/{id}/obras"},
		{"GET /api/v1/obras/55555555-5555-5555-5555-555555555555", "GET /api/v1/obras/{id}"},
		{"PUT /api/v1/obras/55555555-5555-5555-5555-555555555555", "PUT /api/v1/obras/{id}"},
		{"PATCH /api/v1/obras/55555555-5555-5555-5555-555555555555/ativar", "PATCH /api/v1/obras/{id}/ativar"},
		{"PATCH /api/v1/obras/55555555-5555-5555-5555-555555555555/desativar", "PATCH /api/v1/obras/{id}/desativar"},
		{"POST /api/v1/cliente-grupos", "POST /api/v1/cliente-grupos"},
		{"GET /api/v1/cliente-grupos", "GET /api/v1/cliente-grupos"},
		{"GET /api/v1/cliente-grupos/55555555-5555-5555-5555-555555555555", "GET /api/v1/cliente-grupos/{id}"},
		{"PUT /api/v1/cliente-grupos/55555555-5555-5555-5555-555555555555", "PUT /api/v1/cliente-grupos/{id}"},
		{"PATCH /api/v1/cliente-grupos/55555555-5555-5555-5555-555555555555/ativar", "PATCH /api/v1/cliente-grupos/{id}/ativar"},
		{"PATCH /api/v1/cliente-grupos/55555555-5555-5555-5555-555555555555/desativar", "PATCH /api/v1/cliente-grupos/{id}/desativar"},
		{"POST /api/v1/representantes", "POST /api/v1/representantes"},
		{"GET /api/v1/representantes", "GET /api/v1/representantes"},
		{"GET /api/v1/representantes/55555555-5555-5555-5555-555555555555", "GET /api/v1/representantes/{id}"},
		{"PUT /api/v1/representantes/55555555-5555-5555-5555-555555555555", "PUT /api/v1/representantes/{id}"},
		{"PATCH /api/v1/representantes/55555555-5555-5555-5555-555555555555/ativar", "PATCH /api/v1/representantes/{id}/ativar"},
		{"PATCH /api/v1/representantes/55555555-5555-5555-5555-555555555555/desativar", "PATCH /api/v1/representantes/{id}/desativar"},
		{"POST /api/v1/leads/11111111-1111-1111-1111-111111111111/converter", "POST /api/v1/leads/{id}/converter"},
		{"POST /api/v1/clientes/importacao", "POST /api/v1/clientes/importacao"},
		{"POST /api/v1/fornecedores", "POST /api/v1/fornecedores"},
		{"GET /api/v1/fornecedores", "GET /api/v1/fornecedores"},
		{"GET /api/v1/fornecedores/66666666-6666-6666-6666-666666666666", "GET /api/v1/fornecedores/{id}"},
		{"PUT /api/v1/fornecedores/66666666-6666-6666-6666-666666666666", "PUT /api/v1/fornecedores/{id}"},
		{"PATCH /api/v1/fornecedores/66666666-6666-6666-6666-666666666666/ativar", "PATCH /api/v1/fornecedores/{id}/ativar"},
		{"PATCH /api/v1/fornecedores/66666666-6666-6666-6666-666666666666/desativar", "PATCH /api/v1/fornecedores/{id}/desativar"},
		{"DELETE /api/v1/fornecedores/66666666-6666-6666-6666-666666666666", "DELETE /api/v1/fornecedores/{id}"},
		{"POST /api/v1/contas-bancarias", "POST /api/v1/contas-bancarias"},
		{"GET /api/v1/contas-bancarias", "GET /api/v1/contas-bancarias"},
		{"GET /api/v1/contas-bancarias/77777777-7777-7777-7777-777777777777", "GET /api/v1/contas-bancarias/{id}"},
		{"PUT /api/v1/contas-bancarias/77777777-7777-7777-7777-777777777777", "PUT /api/v1/contas-bancarias/{id}"},
		{"PATCH /api/v1/contas-bancarias/77777777-7777-7777-7777-777777777777/ativar", "PATCH /api/v1/contas-bancarias/{id}/ativar"},
		{"PATCH /api/v1/contas-bancarias/77777777-7777-7777-7777-777777777777/desativar", "PATCH /api/v1/contas-bancarias/{id}/desativar"},
		{"DELETE /api/v1/contas-bancarias/77777777-7777-7777-7777-777777777777", "DELETE /api/v1/contas-bancarias/{id}"},
		{"GET /api/v1/bancos", "GET /api/v1/bancos"},
		{"POST /api/v1/categorias-financeiras", "POST /api/v1/categorias-financeiras"},
		{"GET /api/v1/categorias-financeiras", "GET /api/v1/categorias-financeiras"},
		{"GET /api/v1/categorias-financeiras/arvore", "GET /api/v1/categorias-financeiras/arvore"},
		{"GET /api/v1/categorias-financeiras/88888888-8888-8888-8888-888888888888", "GET /api/v1/categorias-financeiras/{id}"},
		{"PUT /api/v1/categorias-financeiras/88888888-8888-8888-8888-888888888888", "PUT /api/v1/categorias-financeiras/{id}"},
		{"PATCH /api/v1/categorias-financeiras/88888888-8888-8888-8888-888888888888/ativar", "PATCH /api/v1/categorias-financeiras/{id}/ativar"},
		{"PATCH /api/v1/categorias-financeiras/88888888-8888-8888-8888-888888888888/desativar", "PATCH /api/v1/categorias-financeiras/{id}/desativar"},
		{"DELETE /api/v1/categorias-financeiras/88888888-8888-8888-8888-888888888888", "DELETE /api/v1/categorias-financeiras/{id}"},
		{"POST /api/v1/pedidos", "POST /api/v1/pedidos"},
		{"GET /api/v1/pedidos", "GET /api/v1/pedidos"},
		{"GET /api/v1/pedidos/99999999-9999-9999-9999-999999999999", "GET /api/v1/pedidos/{id}"},
		{"PUT /api/v1/pedidos/99999999-9999-9999-9999-999999999999", "PUT /api/v1/pedidos/{id}"},
		{"POST /api/v1/pedidos/99999999-9999-9999-9999-999999999999/confirmar", "POST /api/v1/pedidos/{id}/confirmar"},
		{"POST /api/v1/pedidos/99999999-9999-9999-9999-999999999999/entrada", "POST /api/v1/pedidos/{id}/entrada"},
		{"POST /api/v1/pedidos/99999999-9999-9999-9999-999999999999/reservar-pendencias", "POST /api/v1/pedidos/{id}/reservar-pendencias"},
		{"POST /api/v1/pedidos/99999999-9999-9999-9999-999999999999/cancelar", "POST /api/v1/pedidos/{id}/cancelar"},
		{"GET /api/v1/pedidos/99999999-9999-9999-9999-999999999999/historico", "GET /api/v1/pedidos/{id}/historico"},
		{"POST /api/v1/pedidos/99999999-9999-9999-9999-999999999999/entregas", "POST /api/v1/pedidos/{id}/entregas"},
		{"GET /api/v1/pedidos/99999999-9999-9999-9999-999999999999/entregas", "GET /api/v1/pedidos/{id}/entregas"},
		{"GET /api/v1/entregas", "GET /api/v1/entregas"},
		{"GET /api/v1/entregas/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "GET /api/v1/entregas/{id}"},
		{"POST /api/v1/entregas/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/saiu", "POST /api/v1/entregas/{id}/saiu"},
		{"POST /api/v1/entregas/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/entregue", "POST /api/v1/entregas/{id}/entregue"},
		{"POST /api/v1/entregas/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/cancelar", "POST /api/v1/entregas/{id}/cancelar"},
		{"GET /api/v1/pedidos/pendencias", "GET /api/v1/pedidos/pendencias"},
		{"GET /api/v1/pedidos/prazos", "GET /api/v1/pedidos/prazos"},
		{"GET /api/v1/pedidos/cancelamentos", "GET /api/v1/pedidos/cancelamentos"},
		{"POST /api/v1/titulos-receber", "POST /api/v1/titulos-receber"},
		{"GET /api/v1/titulos-receber", "GET /api/v1/titulos-receber"},
		{"GET /api/v1/titulos-receber/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "GET /api/v1/titulos-receber/{id}"},
		{"GET /api/v1/titulos-receber/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/historico", "GET /api/v1/titulos-receber/{id}/historico"},
		{"PUT /api/v1/titulos-receber/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "PUT /api/v1/titulos-receber/{id}"},
		{"POST /api/v1/titulos-receber/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/prorrogar", "POST /api/v1/titulos-receber/{id}/prorrogar"},
		{"POST /api/v1/titulos-receber/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/enviar-banco", "POST /api/v1/titulos-receber/{id}/enviar-banco"},
		{"POST /api/v1/titulos-receber/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/retirar-banco", "POST /api/v1/titulos-receber/{id}/retirar-banco"},
		{"POST /api/v1/titulos-receber/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/boleto-enviado", "POST /api/v1/titulos-receber/{id}/boleto-enviado"},
		{"POST /api/v1/titulos-receber/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/cancelar", "POST /api/v1/titulos-receber/{id}/cancelar"},
		{"POST /api/v1/pedidos/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/titulo-entrada", "POST /api/v1/pedidos/{id}/titulo-entrada"},
		{"POST /api/v1/titulos-receber/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/recebimentos", "POST /api/v1/titulos-receber/{id}/recebimentos"},
		{"GET /api/v1/titulos-receber/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/recebimentos", "GET /api/v1/titulos-receber/{id}/recebimentos"},
		{"POST /api/v1/titulos-receber/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/recebimentos/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/estorno", "POST /api/v1/titulos-receber/{id}/recebimentos/{recebimento_id}/estorno"},
		{"POST /api/v1/antecipacoes", "POST /api/v1/antecipacoes"},
		{"GET /api/v1/antecipacoes", "GET /api/v1/antecipacoes"},
		{"GET /api/v1/antecipacoes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "GET /api/v1/antecipacoes/{id}"},
		{"POST /api/v1/antecipacoes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/estorno", "POST /api/v1/antecipacoes/{id}/estorno"},
		{"POST /api/v1/titulos-receber/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/recompra", "POST /api/v1/titulos-receber/{id}/recompra"},
		{"GET /api/v1/titulos-receber/boletos-pendentes", "GET /api/v1/titulos-receber/boletos-pendentes"},
		{"GET /api/v1/contas-a-receber/resumo", "GET /api/v1/contas-a-receber/resumo"},
		{"GET /api/v1/pedidos/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/comissao", "GET /api/v1/pedidos/{id}/comissao"},
		{"PUT /api/v1/pedidos/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/comissao", "PUT /api/v1/pedidos/{id}/comissao"},
		{"POST /api/v1/comissoes/apurar", "POST /api/v1/comissoes/apurar"},
		{"GET /api/v1/comissoes/pendencias", "GET /api/v1/comissoes/pendencias"},
		{"GET /api/v1/comissoes", "GET /api/v1/comissoes"},
		{"POST /api/v1/comissoes/ajustes", "POST /api/v1/comissoes/ajustes"},
		{"GET /api/v1/comissoes/ajustes", "GET /api/v1/comissoes/ajustes"},
		{"PUT /api/v1/representantes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/fornecedor", "PUT /api/v1/representantes/{id}/fornecedor"},
		{"POST /api/v1/comissoes/fechamentos", "POST /api/v1/comissoes/fechamentos"},
		{"GET /api/v1/comissoes/fechamentos", "GET /api/v1/comissoes/fechamentos"},
		{"GET /api/v1/comissoes/fechamentos/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "GET /api/v1/comissoes/fechamentos/{id}"},
		{"GET /api/v1/comissoes/resumo", "GET /api/v1/comissoes/resumo"},
		{"POST /api/v1/tabela-precos", "POST /api/v1/tabela-precos"},
		{"GET /api/v1/tabela-precos", "GET /api/v1/tabela-precos"},
		{"GET /api/v1/tabela-precos/vigentes", "GET /api/v1/tabela-precos/vigentes"},
		{"DELETE /api/v1/tabela-precos/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "DELETE /api/v1/tabela-precos/{id}"},
		{"GET /api/v1/pedidos/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/proposta", "GET /api/v1/pedidos/{id}/proposta"},
		{"GET /api/v1/modelos-venda", "GET /api/v1/modelos-venda"},
		{"PUT /api/v1/caixa/contas/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/abertura", "PUT /api/v1/caixa/contas/{id}/abertura"},
		{"GET /api/v1/caixa/contas/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/abertura", "GET /api/v1/caixa/contas/{id}/abertura"},
		{"GET /api/v1/caixa/saldos", "GET /api/v1/caixa/saldos"},
		{"POST /api/v1/extratos/importacao", "POST /api/v1/extratos/importacao"},
		{"GET /api/v1/extratos/importacoes", "GET /api/v1/extratos/importacoes"},
		{"DELETE /api/v1/extratos/importacoes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "DELETE /api/v1/extratos/importacoes/{id}"},
		{"GET /api/v1/extratos/lancamentos", "GET /api/v1/extratos/lancamentos"},
		{"GET /api/v1/extratos/lancamentos/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/sugestoes", "GET /api/v1/extratos/lancamentos/{id}/sugestoes"},
		{"POST /api/v1/extratos/lancamentos/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/conciliacao", "POST /api/v1/extratos/lancamentos/{id}/conciliacao"},
		{"POST /api/v1/extratos/conciliar-automatico", "POST /api/v1/extratos/conciliar-automatico"},
		{"GET /api/v1/conciliacoes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "GET /api/v1/conciliacoes/{id}"},
		{"POST /api/v1/conciliacoes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/desfazer", "POST /api/v1/conciliacoes/{id}/desfazer"},
		{"GET /api/v1/caixa/pendencias-sistema", "GET /api/v1/caixa/pendencias-sistema"},
		{"GET /api/v1/caixa/pagamentos-sem-conta", "GET /api/v1/caixa/pagamentos-sem-conta"},
		{"POST /api/v1/extratos/lancamentos/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/classificacao", "POST /api/v1/extratos/lancamentos/{id}/classificacao"},
		{"POST /api/v1/extratos/lancamentos/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/transferencia", "POST /api/v1/extratos/lancamentos/{id}/transferencia"},
		{"GET /api/v1/caixa/lancamentos", "GET /api/v1/caixa/lancamentos"},
		{"GET /api/v1/caixa/transferencias", "GET /api/v1/caixa/transferencias"},
		{"GET /api/v1/caixa/fechamentos/previa", "GET /api/v1/caixa/fechamentos/previa"},
		{"POST /api/v1/caixa/fechamentos", "POST /api/v1/caixa/fechamentos"},
		{"GET /api/v1/caixa/fechamentos", "GET /api/v1/caixa/fechamentos"},
		{"GET /api/v1/caixa/fechamentos/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "GET /api/v1/caixa/fechamentos/{id}"},
		{"POST /api/v1/caixa/fechamentos/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/reabrir", "POST /api/v1/caixa/fechamentos/{id}/reabrir"},
		{"GET /api/v1/caixa/projecao", "GET /api/v1/caixa/projecao"},
		{"GET /api/v1/caixa/movimento-diario", "GET /api/v1/caixa/movimento-diario"},
		{"POST /api/v1/importacoes", "POST /api/v1/importacoes"},
		{"GET /api/v1/importacoes", "GET /api/v1/importacoes"},
		{"GET /api/v1/importacoes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "GET /api/v1/importacoes/{id}"},
		{"PUT /api/v1/importacoes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "PUT /api/v1/importacoes/{id}"},
		{"POST /api/v1/importacoes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/embarcar", "POST /api/v1/importacoes/{id}/embarcar"},
		{"POST /api/v1/importacoes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/desembaracar", "POST /api/v1/importacoes/{id}/desembaracar"},
		{"POST /api/v1/importacoes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/cancelar", "POST /api/v1/importacoes/{id}/cancelar"},
		{"GET /api/v1/importacoes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/historico", "GET /api/v1/importacoes/{id}/historico"},
		{"POST /api/v1/importacoes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/parcelas", "POST /api/v1/importacoes/{id}/parcelas"},
		{"PUT /api/v1/importacoes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/parcelas/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "PUT /api/v1/importacoes/{id}/parcelas/{parcela_id}"},
		{"DELETE /api/v1/importacoes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/parcelas/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "DELETE /api/v1/importacoes/{id}/parcelas/{parcela_id}"},
		{"POST /api/v1/importacoes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/parcelas/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb/cambio", "POST /api/v1/importacoes/{id}/parcelas/{parcela_id}/cambio"},
		{"POST /api/v1/importacoes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/parcelas/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb/reabrir", "POST /api/v1/importacoes/{id}/parcelas/{parcela_id}/reabrir"},
		{"POST /api/v1/importacoes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/custos", "POST /api/v1/importacoes/{id}/custos"},
		{"POST /api/v1/importacoes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/custos/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb/cancelar", "POST /api/v1/importacoes/{id}/custos/{custo_id}/cancelar"},
		{"GET /api/v1/importacoes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/custo-previsto", "GET /api/v1/importacoes/{id}/custo-previsto"},
		{"PUT /api/v1/importacoes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/alocacoes", "PUT /api/v1/importacoes/{id}/alocacoes"},
		{"POST /api/v1/importacoes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/receber", "POST /api/v1/importacoes/{id}/receber"},
		{"GET /api/v1/importacoes/alertas", "GET /api/v1/importacoes/alertas"},
		{"GET /api/v1/pedidos/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/importacoes", "GET /api/v1/pedidos/{id}/importacoes"},
		{"GET /api/v1/dre/resumo", "GET /api/v1/dre/resumo"},
		{"GET /api/v1/dre/faturamento", "GET /api/v1/dre/faturamento"},
		{"GET /api/v1/dre/relatorio-semanal", "GET /api/v1/dre/relatorio-semanal"},
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

func TestRotaNaoAtendidaRespondeNoEnvelopeJSON(t *testing.T) {
	roteador := NewRouter(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	casos := []struct {
		metodo string
		rota   string
		status int
	}{
		{http.MethodGet, "/nao-existe", http.StatusNotFound},
		{http.MethodPost, "/nao-existe", http.StatusNotFound},
		{http.MethodDelete, "/api/v1/sessao", http.StatusMethodNotAllowed},
	}

	for _, caso := range casos {
		registrador := httptest.NewRecorder()
		roteador.ServeHTTP(registrador, httptest.NewRequest(caso.metodo, caso.rota, nil))

		if registrador.Code != caso.status {
			t.Errorf("%s %s: status %d, esperado %d", caso.metodo, caso.rota, registrador.Code, caso.status)
		}
		if !strings.Contains(registrador.Body.String(), `"erro"`) {
			t.Errorf("%s %s: corpo %q fora do envelope de erro", caso.metodo, caso.rota, registrador.Body.String())
		}
	}
}
