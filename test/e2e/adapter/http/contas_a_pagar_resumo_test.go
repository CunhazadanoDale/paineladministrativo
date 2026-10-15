//go:build e2e

package http

import (
	"net/http"
	"testing"

	solicitacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/solicitacao"
)

func TestPendenciasEResumoDeContasAPagar(t *testing.T) {
	servidor, banco := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	red := criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond").ID
	tokenFinanceiro := tokenFinanceiroDasEmpresas(t, servidor, "Fábio MIA", "fabio@exemplo.com", mia)
	tokenSolicitante, tokenAprovador := usuariosParaSolicitacao(t, servidor, "resumo")

	criarSolicitacaoNaEmpresa(t, servidor, tokenSolicitante, mia)
	criarSolicitacaoNaEmpresa(t, servidor, tokenSolicitante, mia)
	aprovadaEmDia := criarSolicitacaoNaEmpresa(t, servidor, tokenSolicitante, mia)
	aprovadaVencida := criarSolicitacaoNaEmpresa(t, servidor, tokenSolicitante, mia)
	daRed := criarSolicitacaoNaEmpresa(t, servidor, tokenSolicitante, red)
	for _, solicitacao := range []solicitacaodto.SolicitacaoResponse{aprovadaEmDia, aprovadaVencida, daRed} {
		conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/solicitacoes/"+solicitacao.ID.String()+"/aprovar", nil, tokenAprovador), http.StatusOK)
	}
	if _, err := banco.Exec(`UPDATE solicitacao SET prazo_pagamento = current_date - 3 WHERE id = $1`, aprovadaVencida.ID); err != nil {
		t.Fatalf("preparação do vencimento falhou: %v", err)
	}

	pendenciasDe := func(token string) solicitacaodto.PendenciasResponse {
		t.Helper()
		resposta := enviaComToken(t, servidor, http.MethodGet, "/api/v1/solicitacoes/pendencias", nil, token)
		conferirStatus(t, resposta, http.StatusOK)
		return decodificarEnvelope[solicitacaodto.PendenciasResponse](t, resposta).Dados
	}
	valor := func(ponteiro *int) int {
		if ponteiro == nil {
			return -1
		}
		return *ponteiro
	}

	if doSolicitante := pendenciasDe(tokenSolicitante); doSolicitante.Aprovacao != nil || doSolicitante.Financeiro != nil || doSolicitante.Vencidas != nil {
		t.Errorf("pendências do solicitante = %+v, esperado tudo nulo", doSolicitante)
	}
	if doAprovador := pendenciasDe(tokenAprovador); valor(doAprovador.Aprovacao) != 2 || doAprovador.Financeiro != nil {
		t.Errorf("pendências do aprovador = aprovação %d, financeiro %v", valor(doAprovador.Aprovacao), doAprovador.Financeiro)
	}
	if doFinanceiro := pendenciasDe(tokenFinanceiro); doFinanceiro.Aprovacao != nil || valor(doFinanceiro.Financeiro) != 2 || valor(doFinanceiro.Vencidas) != 1 {
		t.Errorf("pendências do financeiro da MIA = %+v (financeiro %d, vencidas %d)", doFinanceiro, valor(doFinanceiro.Financeiro), valor(doFinanceiro.Vencidas))
	}

	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/contas-a-pagar/resumo", nil, tokenSolicitante), http.StatusForbidden)
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/contas-a-pagar/resumo?empresa="+red.String(), nil, tokenFinanceiro),
		http.StatusForbidden, "perfil sem acesso a esta empresa")
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/contas-a-pagar/resumo?de=2026-03-01&ate=2027-03-05", nil, tokenFinanceiro),
		http.StatusBadRequest, "erro de validação: intervalo do resumo deve ter no máximo 1 ano")

	respostaResumo := enviaComToken(t, servidor, http.MethodGet, "/api/v1/contas-a-pagar/resumo", nil, tokenFinanceiro)
	conferirStatus(t, respostaResumo, http.StatusOK)
	resumo := decodificarEnvelope[solicitacaodto.ResumoContasAPagarResponse](t, respostaResumo).Dados

	totais := resumo.Totais
	if totais.Previsto.Quantidade != 2 || totais.Previsto.ValorCentavos != 100000 {
		t.Errorf("previsto = %+v, esperado as duas pendentes da MIA", totais.Previsto)
	}
	if totais.APagar.Quantidade != 1 || totais.Vencido.Quantidade != 1 || totais.Vencido.ValorCentavos != 50000 {
		t.Errorf("a pagar = %+v, vencido = %+v", totais.APagar, totais.Vencido)
	}
	for _, semana := range resumo.Semanas {
		if semana.EmpresaID != mia {
			t.Errorf("semana de outra empresa no resumo do financeiro da MIA: %+v", semana)
		}
	}
	if resumo.De == "" || resumo.Ate <= resumo.De {
		t.Errorf("período = %s a %s", resumo.De, resumo.Ate)
	}

	respostaConsolidada := envia(t, servidor, http.MethodGet, "/api/v1/contas-a-pagar/resumo", nil)
	conferirStatus(t, respostaConsolidada, http.StatusOK)
	if consolidado := decodificarEnvelope[solicitacaodto.ResumoContasAPagarResponse](t, respostaConsolidada).Dados; consolidado.Totais.APagar.Quantidade != 2 {
		t.Errorf("a pagar consolidado = %+v, esperado MIA e Red", consolidado.Totais.APagar)
	}
}
