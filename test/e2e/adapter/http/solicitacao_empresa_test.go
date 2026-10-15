//go:build e2e

package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	solicitacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/solicitacao"
	"github.com/google/uuid"
)

func criarSolicitacaoNaEmpresa(t *testing.T, servidor *httptest.Server, token string, empresaID uuid.UUID) solicitacaodto.SolicitacaoResponse {
	t.Helper()

	resposta := enviaComToken(t, servidor, http.MethodPost, "/api/v1/solicitacoes", map[string]any{
		"empresa_id":      empresaID,
		"valor_centavos":  50000,
		"prazo_pagamento": time.Now().UTC().AddDate(0, 0, 5).Format("2006-01-02"),
		"observacao":      "Compra de cimento",
		"forma_pagamento": "pix",
	}, token)
	conferirStatus(t, resposta, http.StatusCreated)

	return decodificarEnvelope[solicitacaodto.SolicitacaoResponse](t, resposta).Dados
}

func TestSolicitacoesFicamIsoladasPorEmpresa(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	red := criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond").ID

	cargo := criarCargo(t, servidor, "Operador", false)
	criarUsuarioComAcesso(t, servidor, "Ana Souza", "ana@exemplo.com", "senhaForte123", cargo)
	idAprovador := criarUsuario(t, servidor, "Bruno Costa", "bruno@exemplo.com", "senhaForte123", cargo)
	conferirStatus(t, envia(t, servidor, http.MethodPost, "/api/v1/aprovadores", map[string]any{"usuario_id": idAprovador}), http.StatusCreated)
	conferirStatus(t, envia(t, servidor, http.MethodPut, "/api/v1/usuarios/"+idAprovador.String()+"/empresas", map[string]any{
		"empresa_ids": []uuid.UUID{mia},
	}), http.StatusOK)

	tokenAna := autenticar(t, servidor, "ana@exemplo.com", "senhaForte123").Token
	tokenAprovador := autenticar(t, servidor, "bruno@exemplo.com", "senhaForte123").Token

	daMia := criarSolicitacaoNaEmpresa(t, servidor, tokenAna, mia)
	daRed := criarSolicitacaoNaEmpresa(t, servidor, tokenAna, red)
	if daRed.EmpresaID != red {
		t.Errorf("solicitação devolvida com empresa %s, esperado %s", daRed.EmpresaID, red)
	}

	resposta := enviaComToken(t, servidor, http.MethodGet, "/api/v1/solicitacoes?escopo=aprovacao", nil, tokenAprovador)
	conferirStatus(t, resposta, http.StatusOK)
	if fila := decodificarPagina[solicitacaodto.SolicitacaoResponse](t, resposta).Dados; len(fila) != 1 || fila[0].ID != daMia.ID {
		t.Errorf("fila do aprovador %+v, esperado só a solicitação da MIA", fila)
	}

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/solicitacoes/"+daRed.ID.String(), nil, tokenAprovador)
	conferirStatus(t, resposta, http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodPost, "/api/v1/solicitacoes/"+daRed.ID.String()+"/aprovar", nil, tokenAprovador)
	conferirStatus(t, resposta, http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/solicitacoes?escopo=aprovacao&empresa="+red.String(), nil, tokenAprovador)
	conferirStatus(t, resposta, http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodPost, "/api/v1/solicitacoes/"+daMia.ID.String()+"/aprovar", nil, tokenAprovador)
	conferirStatus(t, resposta, http.StatusOK)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/solicitacoes?empresa="+red.String(), nil, tokenAna)
	conferirStatus(t, resposta, http.StatusOK)
	if minhas := decodificarPagina[solicitacaodto.SolicitacaoResponse](t, resposta).Dados; len(minhas) != 1 || minhas[0].ID != daRed.ID {
		t.Errorf("filtro por empresa devolveu %+v, esperado só a da Red Diamond", minhas)
	}
}

func TestCriarSolicitacaoExigeEmpresaDoAcesso(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	cargo := criarCargo(t, servidor, "Operador", false)
	criarUsuario(t, servidor, "Lia Prado", "lia@exemplo.com", "senhaForte123", cargo)
	token := autenticar(t, servidor, "lia@exemplo.com", "senhaForte123").Token

	corpo := func(empresaID any) map[string]any {
		return map[string]any{
			"empresa_id":      empresaID,
			"valor_centavos":  50000,
			"prazo_pagamento": time.Now().UTC().AddDate(0, 0, 5).Format("2006-01-02"),
			"observacao":      "Compra de cimento",
			"forma_pagamento": "pix",
		}
	}

	resposta := enviaComToken(t, servidor, http.MethodPost, "/api/v1/solicitacoes", corpo(uuid.Nil), token)
	conferirStatus(t, resposta, http.StatusBadRequest)

	resposta = enviaComToken(t, servidor, http.MethodPost, "/api/v1/solicitacoes", corpo(mia), token)
	conferirStatus(t, resposta, http.StatusForbidden)

	resposta = envia(t, servidor, http.MethodPatch, "/api/v1/empresas/"+mia.String()+"/desativar", nil)
	conferirStatus(t, resposta, http.StatusOK)

	resposta = envia(t, servidor, http.MethodPost, "/api/v1/solicitacoes", corpo(mia))
	conferirStatus(t, resposta, http.StatusBadRequest)
}

func TestFinanceiroERejeicaoFicamIsoladosPorEmpresa(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	red := criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond").ID

	cargo := criarCargo(t, servidor, "Operador", false)
	cargoFinanceiro := criarCargoComPerfil(t, servidor, "Setor financeiro", false, true)
	criarUsuarioComAcesso(t, servidor, "Ana Souza", "ana@exemplo.com", "senhaForte123", cargo)
	idAprovador := criarUsuario(t, servidor, "Bruno Costa", "bruno@exemplo.com", "senhaForte123", cargo)
	idFinanceiro := criarUsuario(t, servidor, "Caio Lima", "caio@exemplo.com", "senhaForte123", cargoFinanceiro)
	conferirStatus(t, envia(t, servidor, http.MethodPost, "/api/v1/aprovadores", map[string]any{"usuario_id": idAprovador}), http.StatusCreated)
	for _, id := range []uuid.UUID{idAprovador, idFinanceiro} {
		conferirStatus(t, envia(t, servidor, http.MethodPut, "/api/v1/usuarios/"+id.String()+"/empresas", map[string]any{
			"empresa_ids": []uuid.UUID{mia},
		}), http.StatusOK)
	}

	tokenAna := autenticar(t, servidor, "ana@exemplo.com", "senhaForte123").Token
	tokenAprovador := autenticar(t, servidor, "bruno@exemplo.com", "senhaForte123").Token
	tokenFinanceiro := autenticar(t, servidor, "caio@exemplo.com", "senhaForte123").Token

	pendenteDaRed := criarSolicitacaoNaEmpresa(t, servidor, tokenAna, red)
	resposta := enviaComToken(t, servidor, http.MethodPost, "/api/v1/solicitacoes/"+pendenteDaRed.ID.String()+"/rejeitar",
		map[string]any{"motivo": "fora do orçamento"}, tokenAprovador)
	conferirStatus(t, resposta, http.StatusForbidden)

	daMia := criarSolicitacaoNaEmpresa(t, servidor, tokenAna, mia)
	daRed := criarSolicitacaoNaEmpresa(t, servidor, tokenAna, red)
	for _, id := range []uuid.UUID{daMia.ID, daRed.ID} {
		conferirStatus(t, envia(t, servidor, http.MethodPost, "/api/v1/solicitacoes/"+id.String()+"/aprovar", nil), http.StatusOK)
	}

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/solicitacoes?escopo=financeiro", nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if fila := decodificarPagina[solicitacaodto.SolicitacaoResponse](t, resposta).Dados; len(fila) != 1 || fila[0].ID != daMia.ID {
		t.Errorf("fila do financeiro %+v, esperado só a solicitação da MIA", fila)
	}

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/solicitacoes/"+daRed.ID.String()+"/historico", nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodPost, "/api/v1/solicitacoes/"+daRed.ID.String()+"/pagamento",
		map[string]any{"valor_centavos": 50000}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusForbidden)

	conta := prepararPagamentoE2E(t, servidor, mia, daMia.ID, 2)
	resposta = enviaComToken(t, servidor, http.MethodPost, "/api/v1/solicitacoes/"+daMia.ID.String()+"/pagamento",
		map[string]any{"valor_centavos": 50000, "conta_bancaria_id": conta}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusCreated)
}
