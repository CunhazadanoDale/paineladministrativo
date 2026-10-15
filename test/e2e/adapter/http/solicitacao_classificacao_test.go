//go:build e2e

package http

import (
	"net/http"
	"testing"
	"time"

	solicitacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/solicitacao"
)

func TestFinanceiroClassificaContaAPagarAprovada(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	red := criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond").ID
	tokenFinanceiro := tokenFinanceiroDasEmpresas(t, servidor, "Fábio MIA", "fabio@exemplo.com", mia)
	tokenFinanceiroRed := tokenFinanceiroDasEmpresas(t, servidor, "Rita Red", "rita@exemplo.com", red)
	tokenSolicitante, tokenAprovador := usuariosParaSolicitacao(t, servidor, "contas")

	fornecedor := criarFornecedorE2E(t, servidor, map[string]any{
		"documento": "11.444.777/0001-61", "nome": "Imobiliária Central",
	}, tokenFinanceiro)
	raiz := criarCategoriaFinanceiraE2E(t, servidor, map[string]any{
		"nome": "Despesas administrativas", "natureza": "despesa", "linha_dre": "despesa_fixa",
	}, tokenFinanceiro)
	aluguel := criarCategoriaFinanceiraE2E(t, servidor, map[string]any{
		"nome": "Aluguel", "categoria_pai_id": raiz.ID,
	}, tokenFinanceiro)
	raizReceita := criarCategoriaFinanceiraE2E(t, servidor, map[string]any{
		"nome": "Outras receitas", "natureza": "receita", "linha_dre": "receita_operacional",
	}, tokenFinanceiro)
	receita := criarCategoriaFinanceiraE2E(t, servidor, map[string]any{
		"nome": "Aluguel recebido", "categoria_pai_id": raizReceita.ID,
	}, tokenFinanceiro)

	vencimento := time.Now().UTC().AddDate(0, 0, 5)
	corpo := map[string]any{
		"empresa_id":      mia,
		"valor_centavos":  420000,
		"prazo_pagamento": vencimento.Format("2006-01-02"),
		"observacao":      "Aluguel do galpão",
		"forma_pagamento": "debito_automatico",
		"fornecedor_id":   fornecedor.ID,
		"estimado":        true,
	}

	corpoReceita := map[string]any{"categoria_financeira_id": receita.ID}
	for chave, valor := range corpo {
		corpoReceita[chave] = valor
	}
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/solicitacoes", corpoReceita, tokenSolicitante),
		http.StatusBadRequest, "erro de validação: categoria financeira deve ser de despesa")

	respostaCriacao := enviaComToken(t, servidor, http.MethodPost, "/api/v1/solicitacoes", corpo, tokenSolicitante)
	conferirStatus(t, respostaCriacao, http.StatusCreated)
	criada := decodificarEnvelope[solicitacaodto.SolicitacaoResponse](t, respostaCriacao).Dados
	if criada.Competencia != vencimento.Format("2006-01") || criada.Versao != 1 || criada.Origem != "manual" {
		t.Errorf("criada = competência %q, versão %d, origem %q", criada.Competencia, criada.Versao, criada.Origem)
	}
	if !criada.Estimado || criada.Vencida || criada.CategoriaFinanceiraID != nil || criada.FornecedorID == nil || *criada.FornecedorID != fornecedor.ID {
		t.Errorf("criada = %+v", criada)
	}
	if criada.FormaPagamento != "debito_automatico" {
		t.Errorf("forma = %q, esperado debito_automatico", criada.FormaPagamento)
	}

	caminho := "/api/v1/solicitacoes/" + criada.ID.String()
	classificacao := map[string]any{
		"fornecedor_id":           fornecedor.ID,
		"categoria_financeira_id": aluguel.ID,
		"competencia":             "2026-09",
		"versao":                  1,
	}
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPatch, caminho+"/classificacao", classificacao, tokenFinanceiro),
		http.StatusConflict, "só é possível classificar solicitação aprovada")

	respostaAprovacao := enviaComToken(t, servidor, http.MethodPost, caminho+"/aprovar", nil, tokenAprovador)
	conferirStatus(t, respostaAprovacao, http.StatusOK)
	if aprovada := decodificarEnvelope[solicitacaodto.SolicitacaoResponse](t, respostaAprovacao).Dados; aprovada.Versao != 2 {
		t.Fatalf("versão depois da aprovação = %d, esperado 2", aprovada.Versao)
	}

	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPatch, caminho+"/classificacao", classificacao, tokenSolicitante),
		http.StatusForbidden, "perfil sem permissão para classificar solicitação")
	classificacao["versao"] = 2
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPatch, caminho+"/classificacao", classificacao, tokenFinanceiroRed),
		http.StatusForbidden, "perfil sem acesso a esta empresa")

	classificacao["versao"] = 1
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPatch, caminho+"/classificacao", classificacao, tokenFinanceiro),
		http.StatusConflict, "solicitação alterada por outra operação, recarregue e tente novamente")

	classificacao["versao"] = 2
	respostaClassificacao := enviaComToken(t, servidor, http.MethodPatch, caminho+"/classificacao", classificacao, tokenFinanceiro)
	conferirStatus(t, respostaClassificacao, http.StatusOK)
	classificada := decodificarEnvelope[solicitacaodto.SolicitacaoResponse](t, respostaClassificacao).Dados
	if classificada.Versao != 3 || classificada.Competencia != "2026-09" || classificada.Status != "aprovado" {
		t.Errorf("classificada = versão %d, competência %q, status %q", classificada.Versao, classificada.Competencia, classificada.Status)
	}
	if classificada.CategoriaFinanceiraID == nil || *classificada.CategoriaFinanceiraID != aluguel.ID {
		t.Errorf("categoria = %v, esperado %s", classificada.CategoriaFinanceiraID, aluguel.ID)
	}

	respostaHistorico := enviaComToken(t, servidor, http.MethodGet, caminho+"/historico", nil, tokenFinanceiro)
	conferirStatus(t, respostaHistorico, http.StatusOK)
	historico := decodificarEnvelope[[]solicitacaodto.HistoricoResponse](t, respostaHistorico).Dados
	if len(historico) != 3 || historico[0].Evento != "classificada" || historico[1].Evento != "aprovada" || historico[2].Evento != "criada" {
		t.Errorf("histórico = %+v", historico)
	}

	respostaFila := enviaComToken(t, servidor, http.MethodGet,
		"/api/v1/solicitacoes?escopo=financeiro&categoria="+aluguel.ID.String()+"&competencia=2026-09", nil, tokenFinanceiro)
	conferirStatus(t, respostaFila, http.StatusOK)
	if fila := decodificarPagina[solicitacaodto.SolicitacaoResponse](t, respostaFila).Dados; len(fila) != 1 || fila[0].ID != criada.ID {
		t.Errorf("fila filtrada = %+v", fila)
	}

	respostaVencidas := enviaComToken(t, servidor, http.MethodGet, "/api/v1/solicitacoes?escopo=financeiro&vencidas=true", nil, tokenFinanceiro)
	conferirStatus(t, respostaVencidas, http.StatusOK)
	if vencidas := decodificarPagina[solicitacaodto.SolicitacaoResponse](t, respostaVencidas).Dados; len(vencidas) != 0 {
		t.Errorf("vencidas = %d, esperado 0", len(vencidas))
	}

	conferirMensagem(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/solicitacoes?vencimento_de=01/10/2026", nil, tokenFinanceiro),
		http.StatusBadRequest, "erro de validação: consulta vencimento_de deve estar no formato AAAA-MM-DD")
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/solicitacoes?competencia=09-2026", nil, tokenFinanceiro),
		http.StatusBadRequest, "erro de validação: competência deve estar no formato AAAA-MM")
}
