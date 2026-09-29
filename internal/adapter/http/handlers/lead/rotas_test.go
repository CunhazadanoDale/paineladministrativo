package lead

import (
	"net/http"
	"strings"
	"testing"
)

func TestRotasRegistradasSemConflito(t *testing.T) {
	mux := http.NewServeMux()
	NovosHandlers(nil, nil, nil, nil).RegistrarRotas(mux)

	casos := []struct {
		requisicao string
		padrao     string
	}{
		{"POST /leads", "POST /leads"},
		{"GET /leads", "GET /leads"},
		{"GET /leads/11111111-1111-1111-1111-111111111111", "GET /leads/{id}"},
		{"PUT /leads/11111111-1111-1111-1111-111111111111", "PUT /leads/{id}"},
		{"DELETE /leads/11111111-1111-1111-1111-111111111111", "DELETE /leads/{id}"},
		{"PATCH /leads/11111111-1111-1111-1111-111111111111/etapa", "PATCH /leads/{id}/etapa"},
		{"GET /funils/22222222-2222-2222-2222-222222222222/leads", "GET /funils/{funil_id}/leads"},
		{"GET /etapas/33333333-3333-3333-3333-333333333333/leads", "GET /etapas/{etapa_id}/leads"},
		{"GET /funils/22222222-2222-2222-2222-222222222222/leads/contagem", "GET /funils/{funil_id}/leads/contagem"},
		{"GET /etapas/33333333-3333-3333-3333-333333333333/leads/contagem", "GET /etapas/{etapa_id}/leads/contagem"},
		{"GET /leads/11111111-1111-1111-1111-111111111111/historico", "GET /leads/{lead_id}/historico"},
		{"POST /leads/11111111-1111-1111-1111-111111111111/historico", "POST /leads/{lead_id}/historico"},
		{"POST /funils", "POST /funils"},
		{"GET /funils", "GET /funils"},
		{"GET /funils/22222222-2222-2222-2222-222222222222", "GET /funils/{funil_id}"},
		{"PUT /funils/22222222-2222-2222-2222-222222222222", "PUT /funils/{funil_id}"},
		{"DELETE /funils/22222222-2222-2222-2222-222222222222", "DELETE /funils/{funil_id}"},
		{"GET /funils/22222222-2222-2222-2222-222222222222/etapas", "GET /funils/{funil_id}/etapas"},
		{"PUT /funils/22222222-2222-2222-2222-222222222222/etapas/ordem", "PUT /funils/{funil_id}/etapas/ordem"},
		{"POST /etapas", "POST /etapas"},
		{"GET /etapas/33333333-3333-3333-3333-333333333333", "GET /etapas/{etapa_id}"},
		{"PUT /etapas/33333333-3333-3333-3333-333333333333", "PUT /etapas/{etapa_id}"},
		{"DELETE /etapas/33333333-3333-3333-3333-333333333333", "DELETE /etapas/{etapa_id}"},
		{"GET /etapas/33333333-3333-3333-3333-333333333333/proxima", "GET /etapas/{etapa_id}/proxima"},
		{"GET /etapas/33333333-3333-3333-3333-333333333333/anterior", "GET /etapas/{etapa_id}/anterior"},
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

func TestRotaNaoRegistrada(t *testing.T) {
	mux := http.NewServeMux()
	NovosHandlers(nil, nil, nil, nil).RegistrarRotas(mux)

	requisicao, err := http.NewRequest(http.MethodGet, "/nao-existe", nil)
	if err != nil {
		t.Fatalf("não foi possível montar a requisição: %v", err)
	}

	_, padrao := mux.Handler(requisicao)
	if padrao != "" {
		t.Errorf("esperava nenhuma rota para /nao-existe, veio %q", padrao)
	}
}
