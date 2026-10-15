//go:build e2e

package http

import (
	"net/http"
	"testing"

	solicitacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/solicitacao"
	"github.com/google/uuid"
)

func TestFinanceiroCadastraRecorrenciasEGeraACompetencia(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	red := criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond").ID
	tokenFinanceiro := tokenFinanceiroDasEmpresas(t, servidor, "Fábio MIA", "fabio@exemplo.com", mia)
	tokenComum, _ := usuariosParaSolicitacao(t, servidor, "recorrencia")

	raiz := criarCategoriaFinanceiraE2E(t, servidor, map[string]any{
		"nome": "Despesas administrativas", "natureza": "despesa", "linha_dre": "despesa_fixa",
	}, tokenDoTeste)
	aluguel := criarCategoriaFinanceiraE2E(t, servidor, map[string]any{"nome": "Aluguel", "categoria_pai_id": raiz.ID}, tokenDoTeste)
	fornecedor := criarFornecedorE2E(t, servidor, map[string]any{"documento": "11.444.777/0001-61", "nome": "Imobiliária Central"}, tokenDoTeste)

	corpo := func(empresaID uuid.UUID, descricao string) map[string]any {
		return map[string]any{
			"empresa_id":              empresaID,
			"fornecedor_id":           fornecedor.ID,
			"categoria_financeira_id": aluguel.ID,
			"descricao":               descricao,
			"valor_centavos":          420000,
			"estimado":                false,
			"forma_pagamento":         "transferencia",
			"dia_vencimento":          31,
			"inicio":                  "2026-01",
		}
	}

	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/despesas-recorrentes", corpo(mia, "Aluguel"), tokenComum), http.StatusForbidden)
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/despesas-recorrentes", corpo(red, "Aluguel Red"), tokenFinanceiro),
		http.StatusForbidden, "perfil sem acesso a esta empresa")

	respostaCriacao := enviaComToken(t, servidor, http.MethodPost, "/api/v1/despesas-recorrentes", corpo(mia, "Aluguel do galpão"), tokenFinanceiro)
	conferirStatus(t, respostaCriacao, http.StatusCreated)
	criada := decodificarEnvelope[solicitacaodto.DespesaRecorrenteResponse](t, respostaCriacao).Dados
	if criada.Inicio != "2026-01" || criada.Fim != nil || criada.Versao != 1 || !criada.Ativa || criada.DiaVencimento != 31 {
		t.Errorf("criada = %+v", criada)
	}

	conferirStatus(t, envia(t, servidor, http.MethodPost, "/api/v1/despesas-recorrentes", corpo(red, "Aluguel Red")), http.StatusCreated)
	conferirStatus(t, envia(t, servidor, http.MethodPost, "/api/v1/despesas-recorrentes", corpo(mia, "Revista técnica")), http.StatusCreated)

	respostaLista := enviaComToken(t, servidor, http.MethodGet, "/api/v1/despesas-recorrentes", nil, tokenFinanceiro)
	conferirStatus(t, respostaLista, http.StatusOK)
	if lista := decodificarPagina[solicitacaodto.DespesaRecorrenteResponse](t, respostaLista).Dados; len(lista) != 2 {
		t.Errorf("lista do financeiro da MIA = %d, esperado 2", len(lista))
	}

	alteracao := corpo(mia, "Aluguel do galpão reajustado")
	delete(alteracao, "empresa_id")
	alteracao["valor_centavos"] = 450000
	alteracao["versao"] = 1
	respostaAlteracao := enviaComToken(t, servidor, http.MethodPut, "/api/v1/despesas-recorrentes/"+criada.ID.String(), alteracao, tokenFinanceiro)
	conferirStatus(t, respostaAlteracao, http.StatusOK)
	if alterada := decodificarEnvelope[solicitacaodto.DespesaRecorrenteResponse](t, respostaAlteracao).Dados; alterada.Versao != 2 || alterada.ValorCentavos != 450000 {
		t.Errorf("alterada = %+v", alterada)
	}
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPut, "/api/v1/despesas-recorrentes/"+criada.ID.String(), alteracao, tokenFinanceiro),
		http.StatusConflict, "a despesa recorrente foi alterada por outra pessoa; recarregue e tente de novo")

	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/despesas-recorrentes/gerar?competencia=02-2026", nil, tokenFinanceiro),
		http.StatusBadRequest, "erro de validação: competência deve estar no formato AAAA-MM")

	respostaGeracao := enviaComToken(t, servidor, http.MethodPost, "/api/v1/despesas-recorrentes/gerar?competencia=2026-02", nil, tokenFinanceiro)
	conferirStatus(t, respostaGeracao, http.StatusOK)
	geracao := decodificarEnvelope[solicitacaodto.ResultadoGeracaoResponse](t, respostaGeracao).Dados
	if geracao.Competencia != "2026-02" || len(geracao.Geradas) != 2 || geracao.JaExistentes != 0 || len(geracao.Recusadas) != 0 {
		t.Fatalf("geração = %+v; esperado as duas da MIA", geracao)
	}

	repetida := decodificarEnvelope[solicitacaodto.ResultadoGeracaoResponse](t,
		enviaComToken(t, servidor, http.MethodPost, "/api/v1/despesas-recorrentes/gerar?competencia=2026-02", nil, tokenFinanceiro)).Dados
	if len(repetida.Geradas) != 0 || repetida.JaExistentes != 2 {
		t.Errorf("geração repetida = %+v; esperado duas já existentes", repetida)
	}

	respostaGeradas := envia(t, servidor, http.MethodGet, "/api/v1/solicitacoes?escopo=todas&origem=recorrencia&competencia=2026-02", nil)
	conferirStatus(t, respostaGeradas, http.StatusOK)
	geradas := decodificarPagina[solicitacaodto.SolicitacaoResponse](t, respostaGeradas).Dados
	if len(geradas) != 2 {
		t.Fatalf("solicitações geradas = %d, esperado 2", len(geradas))
	}
	for _, gerada := range geradas {
		if gerada.EmpresaID != mia || gerada.Status != "pendente_aprovacao" || gerada.Origem != "recorrencia" || gerada.OrigemID == nil ||
			gerada.PrazoPagamento.Format("2006-01-02") != "2026-02-28" || gerada.CategoriaFinanceiraID == nil || *gerada.CategoriaFinanceiraID != aluguel.ID {
			t.Errorf("gerada = %+v", gerada)
		}
	}

	conferirStatus(t, enviaComToken(t, servidor, http.MethodDelete, "/api/v1/despesas-recorrentes/"+criada.ID.String(), nil, tokenFinanceiro), http.StatusForbidden)
	conferirMensagem(t, envia(t, servidor, http.MethodDelete, "/api/v1/despesas-recorrentes/"+criada.ID.String(), nil),
		http.StatusBadRequest, "erro de validação: despesa recorrente já gerou solicitações e não pode ser excluída; desative-a")

	respostaDesativacao := enviaComToken(t, servidor, http.MethodPatch, "/api/v1/despesas-recorrentes/"+criada.ID.String()+"/desativar",
		map[string]any{"versao": 2}, tokenFinanceiro)
	conferirStatus(t, respostaDesativacao, http.StatusOK)

	marco := decodificarEnvelope[solicitacaodto.ResultadoGeracaoResponse](t,
		enviaComToken(t, servidor, http.MethodPost, "/api/v1/despesas-recorrentes/gerar?competencia=2026-03", nil, tokenFinanceiro)).Dados
	if len(marco.Geradas) != 1 {
		t.Errorf("março = %+v; esperado só a recorrência ativa", marco)
	}

	nova := decodificarEnvelope[solicitacaodto.DespesaRecorrenteResponse](t,
		envia(t, servidor, http.MethodPost, "/api/v1/despesas-recorrentes", corpo(mia, "Assinatura sem uso"))).Dados
	conferirStatus(t, envia(t, servidor, http.MethodDelete, "/api/v1/despesas-recorrentes/"+nova.ID.String(), nil), http.StatusNoContent)
	conferirStatus(t, envia(t, servidor, http.MethodGet, "/api/v1/despesas-recorrentes/"+nova.ID.String(), nil), http.StatusNotFound)
}
