//go:build e2e

package http

import (
	"net/http"
	"testing"

	dredto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/dre"
	"github.com/CunhazadanoDale/paineladministrativo.git/test/fixtures/nfe"
	"github.com/google/uuid"
)

func linhaDaDRE(t *testing.T, demonstrativo dredto.DemonstrativoResponse, linha string) dredto.LinhaResponse {
	t.Helper()

	for _, atual := range demonstrativo.Linhas {
		if atual.Linha == linha {
			return atual
		}
	}
	t.Fatalf("linha %s ausente do demonstrativo", linha)

	return dredto.LinhaResponse{}
}

func TestDREEFaturamentoPorEmpresa(t *testing.T) {
	servidor, banco := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	red := criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond").ID
	tokenFinanceiro := tokenFinanceiroDasEmpresas(t, servidor, "Fabia Fin", "fabia@mia.exemplo.com", mia)
	tokenSolicitante, _ := usuariosParaSolicitacao(t, servidor, "dre")

	for _, documento := range []nfe.NFe{
		nfe.Nova(cnpjDaMia, cnpjDoCliente, 301),
		nfe.Nova(cnpjDaRed, cnpjDoCliente, 302),
		nfe.Nova(cnpjDaMia, cnpjDaRed, 303),
	} {
		importarNota(t, servidor, map[string]any{"arquivo_id": enviarXML(t, servidor, documento.XML(), tokenDoTeste)}, tokenDoTeste)
	}

	semCategoria := criarSolicitacaoNaEmpresa(t, servidor, tokenSolicitante, mia)
	if _, err := banco.Exec(`UPDATE solicitacao SET competencia = '2026-10-01' WHERE id = $1`, semCategoria.ID); err != nil {
		t.Fatalf("preparação da competência falhou: %v", err)
	}

	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/dre/resumo", nil, tokenSolicitante), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/dre/faturamento", nil, tokenSolicitante), http.StatusForbidden)
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/dre/resumo?empresa="+red.String(), nil, tokenFinanceiro),
		http.StatusForbidden, "perfil sem acesso a esta empresa")
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/dre/resumo?de=10-2026", nil, tokenFinanceiro),
		http.StatusBadRequest, "erro de validação: de deve estar no formato AAAA-MM")
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/dre/resumo?de=2026-10&ate=2026-09", nil, tokenFinanceiro),
		http.StatusBadRequest, "erro de validação: mês final não pode ser anterior ao mês inicial")
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/dre/faturamento?de=2020-01&ate=2026-10", nil, tokenFinanceiro),
		http.StatusBadRequest, "erro de validação: período deve ter no máximo 60 meses")

	resposta := envia(t, servidor, http.MethodGet, "/api/v1/dre/resumo?de=2026-10&ate=2026-10", nil)
	conferirStatus(t, resposta, http.StatusOK)
	consolidado := decodificarEnvelope[dredto.ResumoDREResponse](t, resposta).Dados
	if consolidado.De != "2026-10" || consolidado.Ate != "2026-10" || len(consolidado.Meses) != 1 || len(consolidado.Total.Empresas) != 2 {
		t.Fatalf("resumo consolidado = %+v", consolidado)
	}
	total := consolidado.Total.Consolidado
	if receita := linhaDaDRE(t, total, "receita_bruta"); receita.ValorCentavos != 218000 || receita.Quantidade != 2 {
		t.Errorf("receita bruta = %+v, esperado as duas vendas sem a transferência", receita)
	}
	if naoClassificado := linhaDaDRE(t, total, "nao_classificado"); naoClassificado.ValorCentavos != -50000 || naoClassificado.Quantidade != 1 {
		t.Errorf("não classificado = %+v", naoClassificado)
	}
	if total.Subtotais.ResultadoCentavos != 168000 || total.Indicadores.MargemContribuicaoBP == nil || *total.Indicadores.MargemContribuicaoBP != 10000 {
		t.Errorf("subtotais = %+v, indicadores = %+v", total.Subtotais, total.Indicadores)
	}
	if itens := linhaDaDRE(t, total, "receita_bruta").Itens; len(itens) != 1 || itens[0].Chave != "nf_venda" || itens[0].CategoriaID != nil {
		t.Errorf("itens da receita = %+v", itens)
	}

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/dre/resumo?de=2026-10&ate=2026-10", nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	daMia := decodificarEnvelope[dredto.ResumoDREResponse](t, resposta).Dados
	if len(daMia.Total.Empresas) != 1 || daMia.Total.Empresas[0].EmpresaID != mia {
		t.Fatalf("empresas do financeiro da MIA = %+v", daMia.Total.Empresas)
	}
	if receita := linhaDaDRE(t, daMia.Total.Consolidado, "receita_bruta"); receita.ValorCentavos != 109000 {
		t.Errorf("receita da MIA = %+v", receita)
	}

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/dre/faturamento?de=2026-09&ate=2026-10&empresa="+mia.String(), nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	faturamento := decodificarEnvelope[dredto.ResumoFaturamentoResponse](t, resposta).Dados
	if len(faturamento.Meses) != 2 || faturamento.Meses[0].Consolidado.Quantidade != 0 {
		t.Fatalf("faturamento = %+v", faturamento)
	}
	if faturamento.Consolidado.Quantidade != 1 || faturamento.Consolidado.ValorCentavos != 109000 || faturamento.Consolidado.TicketMedioCentavos != 109000 {
		t.Errorf("faturamento consolidado = %+v", faturamento.Consolidado)
	}
	if len(faturamento.Empresas) != 1 || faturamento.Empresas[0].EmpresaID != mia {
		t.Errorf("empresas do faturamento = %+v", faturamento.Empresas)
	}

	resposta = envia(t, servidor, http.MethodGet, "/api/v1/dre/faturamento?empresa="+uuid.NewString(), nil)
	conferirStatus(t, resposta, http.StatusOK)
	if vazio := decodificarEnvelope[dredto.ResumoFaturamentoResponse](t, resposta).Dados; len(vazio.Meses) != 12 || vazio.Consolidado.Quantidade != 0 {
		t.Errorf("faturamento padrão de empresa sem notas = %d meses, %+v", len(vazio.Meses), vazio.Consolidado)
	}
}
