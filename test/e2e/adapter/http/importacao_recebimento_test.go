//go:build e2e

package http

import (
	"net/http"
	"testing"
	"time"

	estoquedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/estoque"
	importacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/importacao"
	pedidodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/pedido"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
)

func TestAlocacaoERecebimentoDaImportacao(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	red := criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond").ID
	galpao := criarLocal(t, servidor, mia, "Galpão Cabedelo")
	galpaoRed := criarLocal(t, servidor, red, "Galpão Red")
	tokenFinanceiro := tokenFinanceiroDasEmpresas(t, servidor, "Fábio MIA", "fabio@exemplo.com", mia)
	tokenComercial := usuarioComercialDasEmpresas(t, servidor, "Carla MIA", "carla@exemplo.com", mia)
	fornecedor := criarFornecedorE2E(t, servidor, map[string]any{
		"pais": "CN", "identificacao_estrangeira": "91440600MA4UX", "nome": "Foshan Ceramics", "moeda_padrao": "USD",
	}, tokenDoTeste)
	produtoID := produtoDoTeste(t, servidor)
	clienteID := clienteDoPedido(t, servidor)

	pedido := criarPedidoE2E(t, servidor, corpoDePedido(mia, clienteID, produtoID, "importacao", 600), tokenDoTeste)
	caminhoDoPedido := "/api/v1/pedidos/" + pedido.ID.String()
	conferirStatus(t, envia(t, servidor, http.MethodPost, caminhoDoPedido+"/confirmar", map[string]any{"versao": pedido.Versao}), http.StatusOK)
	resposta := envia(t, servidor, http.MethodGet, caminhoDoPedido, nil)
	conferirStatus(t, resposta, http.StatusOK)
	itemDoPedido := decodificarEnvelope[pedidodto.PedidoDetalheResponse](t, resposta).Dados.Itens[0]

	resposta = enviaComToken(t, servidor, http.MethodPost, "/api/v1/importacoes", corpoDeImportacao(mia, fornecedor.ID, produtoID), tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusCreated)
	importacao := decodificarEnvelope[importacaodto.ImportacaoDetalheResponse](t, resposta).Dados
	caminho := "/api/v1/importacoes/" + importacao.ID.String()
	itemDaImportacao := importacao.Itens[0].ID

	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPut, caminho+"/alocacoes", map[string]any{
		"versao": 1, "alocacoes": []map[string]any{{"importacao_item_id": itemDaImportacao, "pedido_item_id": itemDoPedido.ID, "quantidade": 601}},
	}, tokenComercial), http.StatusBadRequest, "erro de validação: as alocações do item do pedido 1 passam do que falta entregar (600, já alocado em outras importações: 0)")
	resposta = enviaComToken(t, servidor, http.MethodPut, caminho+"/alocacoes", map[string]any{
		"versao": 1, "alocacoes": []map[string]any{{"importacao_item_id": itemDaImportacao, "pedido_item_id": itemDoPedido.ID, "quantidade": 600}},
	}, tokenComercial)
	conferirStatus(t, resposta, http.StatusOK)
	alocada := decodificarEnvelope[importacaodto.ImportacaoDetalheResponse](t, resposta).Dados
	if alocada.Versao != 2 || len(alocada.Alocacoes) != 1 || alocada.Alocacoes[0].Quantidade != 600 || alocada.ValorTotalMoedaCentesimos != nil {
		t.Fatalf("importação alocada pelo comercial = %+v", alocada)
	}

	hoje := domainimportacao.Hoje(time.Now())
	data := func(dias int) string { return hoje.AddDate(0, 0, dias).Format("2006-01-02") }
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/embarcar", map[string]any{
		"versao": 2, "embarcada_em": data(-10), "chegada_prevista": data(-2),
	}, tokenFinanceiro), http.StatusOK)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/desembaracar", map[string]any{
		"versao": 3, "chegada_em": data(-2), "desembaracada_em": data(-1), "declaracao": "26/0001234-5",
	}, tokenFinanceiro), http.StatusOK)

	recebimento := map[string]any{"versao": 4, "recebida_em": data(0), "local_id": galpao.ID, "cambio_micro": 5_000_000}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/receber", recebimento, tokenComercial), http.StatusForbidden)
	deOutraEmpresa := map[string]any{"versao": 4, "recebida_em": data(0), "local_id": galpaoRed.ID, "cambio_micro": 5_000_000}
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/receber", deOutraEmpresa, tokenDoTeste),
		http.StatusBadRequest, "erro de validação: o local do recebimento precisa ser da empresa importadora")
	semCambio := map[string]any{"versao": 4, "recebida_em": data(0), "local_id": galpao.ID}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/receber", semCambio, tokenFinanceiro), http.StatusBadRequest)

	resposta = enviaComToken(t, servidor, http.MethodPost, caminho+"/receber", recebimento, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	recebida := decodificarEnvelope[importacaodto.ImportacaoDetalheResponse](t, resposta).Dados
	item := recebida.Itens[0]
	if recebida.Status != "recebida" || recebida.RecebidaEm == nil || *recebida.RecebidaEm != data(0) || recebida.CambioMicro == nil ||
		item.QuantidadeRecebida == nil || *item.QuantidadeRecebida != 1000 || item.CustoUnitarioCentavos == nil ||
		*item.CustoUnitarioCentavos != 2250 || item.MovimentoID == nil {
		t.Fatalf("importação recebida = %+v; item %+v", recebida, item)
	}
	if alocacao := recebida.Alocacoes[0]; alocacao.Reservada == nil || *alocacao.Reservada != 600 || alocacao.ReservaID == nil {
		t.Errorf("alocação depois do recebimento = %+v", alocacao)
	}

	resposta = enviaComToken(t, servidor, http.MethodGet, caminho, nil, tokenComercial)
	conferirStatus(t, resposta, http.StatusOK)
	if vista := decodificarEnvelope[importacaodto.ImportacaoDetalheResponse](t, resposta).Dados; vista.CambioMicro != nil ||
		vista.Itens[0].CustoUnitarioCentavos != nil || vista.Itens[0].QuantidadeRecebida == nil || len(vista.Alocacoes) != 1 {
		t.Errorf("visão do comercial depois do recebimento = %+v", vista)
	}
	resposta = enviaComToken(t, servidor, http.MethodGet, caminho+"/historico", nil, tokenComercial)
	conferirStatus(t, resposta, http.StatusOK)
	for _, linha := range decodificarEnvelope[[]importacaodto.HistoricoResponse](t, resposta).Dados {
		if linha.Acao == "recebida" && linha.Detalhe != "" {
			t.Errorf("comercial viu o detalhe do recebimento: %q", linha.Detalhe)
		}
	}

	resposta = envia(t, servidor, http.MethodGet, "/api/v1/produtos/"+produtoID.String()+"/saldo", nil)
	conferirStatus(t, resposta, http.StatusOK)
	if saldo := decodificarEnvelope[estoquedto.SaldoResponse](t, resposta).Dados; saldo.Saldo != 1000 || saldo.Reservado != 600 {
		t.Errorf("saldo do produto = %+v", saldo)
	}
	resposta = envia(t, servidor, http.MethodGet, "/api/v1/produtos/"+produtoID.String()+"/lotes", nil)
	conferirStatus(t, resposta, http.StatusOK)
	if lotes := decodificarPagina[estoquedto.LoteResponse](t, resposta).Dados; len(lotes) != 1 || lotes[0].CustoUnitarioCentavos == nil ||
		*lotes[0].CustoUnitarioCentavos != 2250 {
		t.Errorf("lotes = %+v", lotes)
	}

	resposta = envia(t, servidor, http.MethodGet, caminhoDoPedido, nil)
	conferirStatus(t, resposta, http.StatusOK)
	if detalhe := decodificarEnvelope[pedidodto.PedidoDetalheResponse](t, resposta).Dados; *detalhe.Itens[0].Reservado != 600 ||
		*detalhe.Itens[0].FaltaCobrir != 0 || len(detalhe.Reservas) != 1 || detalhe.Reservas[0].Origem != "pedido_item" {
		t.Errorf("pedido depois do recebimento = %+v", detalhe)
	}
	resposta = envia(t, servidor, http.MethodGet, caminhoDoPedido+"/historico", nil)
	conferirStatus(t, resposta, http.StatusOK)
	historico := decodificarEnvelope[[]pedidodto.HistoricoResponse](t, resposta).Dados
	if ultimo := historico[len(historico)-1]; ultimo.Acao != "pendencias_reservadas" {
		t.Errorf("último histórico do pedido = %+v", ultimo)
	}

	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/receber", map[string]any{
		"versao": 5, "recebida_em": data(0), "local_id": galpao.ID, "cambio_micro": 5_000_000,
	}, tokenFinanceiro), http.StatusConflict)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPut, caminho+"/alocacoes", map[string]any{"versao": 5}, tokenComercial), http.StatusConflict)
}
