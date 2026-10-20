//go:build e2e

package http

import (
	"net/http"
	"testing"
	"time"

	importacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/importacao"
	solicitacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/solicitacao"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
)

func TestParcelasCambioECustosDaImportacao(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	tokenFinanceiro := tokenFinanceiroDasEmpresas(t, servidor, "Fábio MIA", "fabio@exemplo.com", mia)
	tokenComercial := usuarioComercialDasEmpresas(t, servidor, "Carla MIA", "carla@exemplo.com", mia)
	fornecedor := criarFornecedorE2E(t, servidor, map[string]any{
		"pais": "CN", "identificacao_estrangeira": "91440600MA4UX", "nome": "Foshan Ceramics", "moeda_padrao": "USD",
	}, tokenDoTeste)
	despachante := criarFornecedorE2E(t, servidor, map[string]any{"documento": "45.723.174/0001-10", "nome": "Despachos Cabedelo Ltda"}, tokenDoTeste)
	raiz := criarCategoriaFinanceiraE2E(t, servidor, map[string]any{
		"nome": "Compras", "natureza": "despesa", "linha_dre": "compra_mercadoria",
	}, tokenDoTeste)
	compras := criarCategoriaFinanceiraE2E(t, servidor, map[string]any{"nome": "Importação", "categoria_pai_id": raiz.ID}, tokenDoTeste)
	produtoID := produtoDoTeste(t, servidor)

	resposta := enviaComToken(t, servidor, http.MethodPost, "/api/v1/importacoes", corpoDeImportacao(mia, fornecedor.ID, produtoID), tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusCreated)
	importacao := decodificarEnvelope[importacaodto.ImportacaoDetalheResponse](t, resposta).Dados
	caminho := "/api/v1/importacoes/" + importacao.ID.String()
	hoje := domainimportacao.Hoje(time.Now()).Format("2006-01-02")

	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/parcelas", map[string]any{
		"versao": 1, "descricao": "Sinal", "valor_moeda_centesimos": 450001, "vencimento_previsto": hoje,
	}, tokenFinanceiro), http.StatusBadRequest, "erro de validação: a soma das parcelas não pode passar do valor dos itens da importação")
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/parcelas", map[string]any{
		"versao": 1, "descricao": "Sinal", "valor_moeda_centesimos": 135000, "vencimento_previsto": hoje,
	}, tokenComercial), http.StatusForbidden)
	resposta = enviaComToken(t, servidor, http.MethodPost, caminho+"/parcelas", map[string]any{
		"versao": 1, "descricao": "Sinal 30%", "valor_moeda_centesimos": 135000, "vencimento_previsto": hoje,
	}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusCreated)
	comParcela := decodificarEnvelope[importacaodto.ImportacaoDetalheResponse](t, resposta).Dados
	if comParcela.Versao != 2 || comParcela.Parcelas == nil || len(*comParcela.Parcelas) != 1 || (*comParcela.Parcelas)[0].Status != "aberta" {
		t.Fatalf("importação com parcela = %+v", comParcela)
	}
	parcela := (*comParcela.Parcelas)[0]
	caminhoDaParcela := caminho + "/parcelas/" + parcela.ID.String()

	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, caminhoDaParcela+"/cambio", map[string]any{
		"versao": 2, "cambio_micro": 5432100, "vencimento": hoje, "categoria_financeira_id": raiz.ID,
	}, tokenFinanceiro), http.StatusBadRequest)
	resposta = enviaComToken(t, servidor, http.MethodPost, caminhoDaParcela+"/cambio", map[string]any{
		"versao": 2, "cambio_micro": 5432100, "vencimento": hoje, "categoria_financeira_id": compras.ID,
	}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	comCambio := decodificarEnvelope[importacaodto.ImportacaoDetalheResponse](t, resposta).Dados
	fechada := (*comCambio.Parcelas)[0]
	if fechada.Status != "cambio_fechado" || fechada.ValorCentavos == nil || *fechada.ValorCentavos != 733334 || fechada.SolicitacaoID == nil ||
		fechada.StatusContaAPagar == nil || *fechada.StatusContaAPagar != "pendente_aprovacao" {
		t.Fatalf("parcela com câmbio = %+v", fechada)
	}

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/solicitacoes/"+fechada.SolicitacaoID.String(), nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if conta := decodificarEnvelope[solicitacaodto.SolicitacaoResponse](t, resposta).Dados; conta.Origem != "importacao_parcela" ||
		conta.OrigemID == nil || *conta.OrigemID != parcela.ID || conta.ValorCentavos != 733334 || *conta.FornecedorID != fornecedor.ID {
		t.Errorf("conta a pagar da parcela = %+v", conta)
	}

	resposta = enviaComToken(t, servidor, http.MethodPost, caminho+"/custos", map[string]any{
		"versao": 3, "tipo": "despachante", "fornecedor_id": despachante.ID, "valor_centavos": 250000,
		"vencimento": hoje, "categoria_financeira_id": compras.ID,
	}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusCreated)
	comCusto := decodificarEnvelope[importacaodto.ImportacaoDetalheResponse](t, resposta).Dados
	custo := (*comCusto.Custos)[0]
	if custo.Descricao != "Despachante aduaneiro" || !custo.CompoeCusto || custo.StatusContaAPagar == nil {
		t.Fatalf("custo lançado = %+v", custo)
	}

	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, caminho+"/custo-previsto", nil, tokenFinanceiro), http.StatusBadRequest)
	resposta = enviaComToken(t, servidor, http.MethodGet, caminho+"/custo-previsto?cambio_micro=5500000", nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	previsto := decodificarEnvelope[importacaodto.CustoPrevistoResponse](t, resposta).Dados
	if previsto.ParcelasFechadasCentavos != 733334 || previsto.SemCambioFechadoMoedaCentesimos != 315000 ||
		previsto.ValorMercadoriaCentavos != 733334+1732500 || previsto.CustosCentavos != 250000 ||
		previsto.Itens[0].CustoUnitarioCentavos == nil || *previsto.Itens[0].CustoUnitarioCentavos != 2716 {
		t.Errorf("custo previsto = %+v", previsto)
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, caminho+"/custo-previsto?cambio_micro=5500000", nil, tokenComercial), http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodGet, caminho, nil, tokenComercial)
	conferirStatus(t, resposta, http.StatusOK)
	if vista := decodificarEnvelope[importacaodto.ImportacaoDetalheResponse](t, resposta).Dados; vista.Parcelas != nil || vista.Custos != nil {
		t.Errorf("comercial vê parcelas ou custos: %+v", vista)
	}

	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/cancelar", map[string]any{"versao": 4, "motivo": "desistência"}, tokenFinanceiro),
		http.StatusConflict, "cancele ou rejeite antes as contas a pagar geradas pela importação")
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, caminhoDaParcela+"/reabrir", map[string]any{"versao": 4}, tokenFinanceiro), http.StatusConflict)

	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/solicitacoes/"+fechada.SolicitacaoID.String()+"/cancelar", nil, tokenFinanceiro), http.StatusOK)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/solicitacoes/"+custo.SolicitacaoID.String()+"/cancelar", nil, tokenFinanceiro), http.StatusOK)

	resposta = enviaComToken(t, servidor, http.MethodPost, caminhoDaParcela+"/reabrir", map[string]any{"versao": 4}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	reaberta := decodificarEnvelope[importacaodto.ImportacaoDetalheResponse](t, resposta).Dados
	if parcelas := *reaberta.Parcelas; len(parcelas) != 2 || parcelas[0].Status != "cancelada" || parcelas[1].Status != "aberta" ||
		parcelas[1].ValorMoedaCentesimos != 135000 {
		t.Errorf("parcelas depois de reabrir = %+v", parcelas)
	}
	nova := (*reaberta.Parcelas)[1]
	resposta = enviaComToken(t, servidor, http.MethodDelete, caminho+"/parcelas/"+nova.ID.String()+"?versao=5", nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)

	resposta = enviaComToken(t, servidor, http.MethodPost, caminho+"/custos/"+custo.ID.String()+"/cancelar", map[string]any{"versao": 6}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if semCusto := decodificarEnvelope[importacaodto.ImportacaoDetalheResponse](t, resposta).Dados; !(*semCusto.Custos)[0].Cancelado {
		t.Errorf("custo depois de cancelar = %+v", (*semCusto.Custos)[0])
	}

	resposta = enviaComToken(t, servidor, http.MethodPost, caminho+"/cancelar", map[string]any{"versao": 7, "motivo": "desistência"}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/custos", map[string]any{
		"versao": 8, "tipo": "seguro", "fornecedor_id": despachante.ID, "valor_centavos": 100, "vencimento": hoje,
		"categoria_financeira_id": compras.ID,
	}, tokenFinanceiro), http.StatusConflict, "importação cancelada não recebe parcela nem custo")

	resposta = enviaComToken(t, servidor, http.MethodGet, caminho+"/historico", nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	acoes := make([]string, 0)
	for _, linha := range decodificarEnvelope[[]importacaodto.HistoricoResponse](t, resposta).Dados {
		acoes = append(acoes, linha.Acao)
	}
	esperadas := []string{"criada", "parcela_incluida", "parcela_cambio_fechado", "custo_lancado", "parcela_reaberta", "parcela_removida", "custo_cancelado", "cancelada"}
	if len(acoes) != len(esperadas) {
		t.Fatalf("histórico = %v", acoes)
	}
	for i := range esperadas {
		if acoes[i] != esperadas[i] {
			t.Errorf("histórico = %v, esperado %v", acoes, esperadas)
			break
		}
	}

	resposta = enviaComToken(t, servidor, http.MethodGet, caminho+"/historico", nil, tokenComercial)
	conferirStatus(t, resposta, http.StatusOK)
	for _, linha := range decodificarEnvelope[[]importacaodto.HistoricoResponse](t, resposta).Dados {
		comValores := linha.Acao != "criada" && linha.Acao != "cancelada"
		if comValores && linha.Detalhe != "" {
			t.Errorf("comercial viu o detalhe de %q: %q", linha.Acao, linha.Detalhe)
		}
		if linha.Acao == "cancelada" && linha.Detalhe != "desistência" {
			t.Errorf("motivo do cancelamento escondido: %q", linha.Detalhe)
		}
	}
}
