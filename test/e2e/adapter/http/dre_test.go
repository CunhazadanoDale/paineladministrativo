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

func TestRelatorioSemanalPorEmpresa(t *testing.T) {
	servidor, banco := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond")
	tokenFinanceiro := tokenFinanceiroDasEmpresas(t, servidor, "Fabio Fin", "fabio@mia.exemplo.com", mia)
	tokenSolicitante, _ := usuariosParaSolicitacao(t, servidor, "semanal")

	importarNota(t, servidor, map[string]any{"arquivo_id": enviarXML(t, servidor, nfe.Nova(cnpjDaMia, cnpjDoCliente, 401).XML(), tokenDoTeste)}, tokenDoTeste)
	importarNota(t, servidor, map[string]any{"arquivo_id": enviarXML(t, servidor, nfe.Nova(cnpjDaRed, cnpjDoCliente, 402).XML(), tokenDoTeste)}, tokenDoTeste)
	vencida := criarSolicitacaoNaEmpresa(t, servidor, tokenSolicitante, mia)
	criarContaE2E(t, servidor, map[string]any{
		"empresa_id": mia, "banco": "748", "tipo": "corrente", "agencia": "0812", "conta": "54321-0", "apelido": "Sicredi",
	}, tokenFinanceiro)
	categoria := criarCategoria(t, servidor, "Cimentos", nil)
	produto := criarProduto(t, servidor, criarCategoria(t, servidor, "CP II", &categoria), "Cimento CP II")
	for _, preparo := range []struct {
		query      string
		argumentos []any
	}{
		{`UPDATE titulo_receber SET vencimento = '2026-10-01'`, nil},
		{`UPDATE solicitacao SET prazo_pagamento = '2026-10-01' WHERE id = $1`, []any{vencida.ID}},
		{`UPDATE produto SET estoque_minimo = 5 WHERE id = $1`, []any{produto}},
	} {
		if _, err := banco.Exec(preparo.query, preparo.argumentos...); err != nil {
			t.Fatalf("preparação falhou: %v", err)
		}
	}

	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/dre/relatorio-semanal", nil, tokenSolicitante), http.StatusForbidden)
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/dre/relatorio-semanal?semana=07/10/2026", nil, tokenFinanceiro),
		http.StatusBadRequest, "erro de validação: semana deve estar no formato AAAA-MM-DD")
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/dre/relatorio-semanal?semana=2099-01-05", nil, tokenFinanceiro),
		http.StatusBadRequest, "erro de validação: semana não pode estar no futuro")

	resposta := enviaComToken(t, servidor, http.MethodGet, "/api/v1/dre/relatorio-semanal?semana=2026-10-07", nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	relatorio := decodificarEnvelope[dredto.RelatorioSemanalResponse](t, resposta).Dados
	if relatorio.Inicio != "2026-10-05" || relatorio.Fim != "2026-10-11" {
		t.Errorf("semana = %s a %s", relatorio.Inicio, relatorio.Fim)
	}
	if relatorio.NotasEmitidas.TotalVenda.Quantidade != 1 || relatorio.NotasEmitidas.TotalVenda.ValorCentavos != 109000 {
		t.Errorf("notas da MIA = %+v", relatorio.NotasEmitidas)
	}
	if relatorio.Pagamentos.Total.Quantidade != 0 || len(relatorio.Pagamentos.PorEmpresa) != 0 || len(relatorio.Recebimentos.PorEmpresa) != 0 {
		t.Errorf("pagamentos = %+v, recebimentos = %+v", relatorio.Pagamentos, relatorio.Recebimentos)
	}
	if relatorio.TitulosVencidos.Total.Quantidade != 2 || relatorio.TitulosVencidos.Total.ValorCentavos != 109000 || len(relatorio.TitulosVencidos.Itens) != 2 {
		t.Errorf("títulos vencidos da MIA = %+v", relatorio.TitulosVencidos)
	}
	for _, titulo := range relatorio.TitulosVencidos.Itens {
		if titulo.EmpresaID != mia || titulo.Vencimento != "2026-10-01" || titulo.ClienteNome == "" {
			t.Errorf("título vencido = %+v", titulo)
		}
	}
	if relatorio.ContasVencidas.Total.Quantidade != 1 || len(relatorio.ContasVencidas.Itens) != 1 || relatorio.ContasVencidas.Itens[0].ID != vencida.ID {
		t.Errorf("contas vencidas = %+v", relatorio.ContasVencidas)
	}
	if relatorio.EstoqueBaixo.Total != 1 || len(relatorio.EstoqueBaixo.Itens) != 1 || relatorio.EstoqueBaixo.Itens[0].ProdutoID != produto {
		t.Errorf("estoque baixo = %+v", relatorio.EstoqueBaixo)
	}
	if saldos := relatorio.SaldosBancarios; len(saldos.Contas) != 1 || saldos.ContasSemAbertura != 1 || saldos.Contas[0].BancoNome != "Sicredi" ||
		saldos.Contas[0].SaldoAtualCentavos != nil {
		t.Errorf("saldos = %+v", saldos)
	}

	resposta = envia(t, servidor, http.MethodGet, "/api/v1/dre/relatorio-semanal?semana=2026-10-07", nil)
	conferirStatus(t, resposta, http.StatusOK)
	if consolidado := decodificarEnvelope[dredto.RelatorioSemanalResponse](t, resposta).Dados; consolidado.NotasEmitidas.TotalVenda.Quantidade != 2 ||
		consolidado.TitulosVencidos.Total.Quantidade != 4 {
		t.Errorf("relatório consolidado = notas %+v, vencidos %+v", consolidado.NotasEmitidas.TotalVenda, consolidado.TitulosVencidos.Total)
	}

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/dre/relatorio-semanal", nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if padrao := decodificarEnvelope[dredto.RelatorioSemanalResponse](t, resposta).Dados; padrao.Fim >= padrao.Hoje || padrao.Inicio >= padrao.Fim {
		t.Errorf("semana padrão = %s a %s com hoje %s, esperado a última encerrada", padrao.Inicio, padrao.Fim, padrao.Hoje)
	}
}

func TestDREEEndividamentoComAsContasDasDividas(t *testing.T) {
	c := novoCenarioDividasE2E(t)
	c.criar(t)
	mes := c.hoje.Format("2006-01")

	resposta := enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/dre/resumo?de="+mes+"&ate="+mes, nil, c.tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	total := decodificarEnvelope[dredto.ResumoDREResponse](t, resposta).Dados.Total.Consolidado
	if financeiro := linhaDaDRE(t, total, "resultado_financeiro"); financeiro.ValorCentavos != -5550 || financeiro.Quantidade != 1 {
		t.Errorf("juros da parcela do mês = %+v", financeiro)
	}
	if fora := linhaDaDRE(t, total, "fora_dre"); fora.ValorCentavos != -100000 || fora.Quantidade != 1 {
		t.Errorf("amortização do mês = %+v", fora)
	}
	if total.Subtotais.ResultadoCentavos != -5550 {
		t.Errorf("resultado = %d, esperado só os juros (amortização fora da DRE)", total.Subtotais.ResultadoCentavos)
	}

	resposta = enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/dre/relatorio-semanal", nil, c.tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	endividamento := decodificarEnvelope[dredto.RelatorioSemanalResponse](t, resposta).Dados.Endividamento
	if endividamento.SaldoDevedorCentavos != 300000 || endividamento.Dividas != 1 || len(endividamento.PorCredor) != 4 ||
		endividamento.PorCredor[0].CredorTipo != "banco" || endividamento.PorCredor[0].SaldoDevedorCentavos != 300000 {
		t.Errorf("endividamento da MIA = %+v", endividamento)
	}
	if len(endividamento.PorEmpresa) != 1 || endividamento.PorEmpresa[0].EmpresaID != c.mia || endividamento.AVencer.Parcelas != 3 {
		t.Errorf("endividamento por empresa = %+v, a vencer = %+v", endividamento.PorEmpresa, endividamento.AVencer)
	}

	resposta = enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/dre/relatorio-semanal", nil, c.tokenRed)
	conferirStatus(t, resposta, http.StatusOK)
	if daRed := decodificarEnvelope[dredto.RelatorioSemanalResponse](t, resposta).Dados.Endividamento; daRed.SaldoDevedorCentavos != 0 || len(daRed.PorEmpresa) != 0 {
		t.Errorf("endividamento visto pela Red = %+v", daRed)
	}
}
