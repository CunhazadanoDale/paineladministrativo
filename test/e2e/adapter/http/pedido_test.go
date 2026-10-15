//go:build e2e

package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	clientedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/cliente"
	estoquedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/estoque"
	pedidodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/pedido"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	"github.com/google/uuid"
)

func clienteDoPedido(t *testing.T, servidor *httptest.Server) uuid.UUID {
	t.Helper()

	resposta := envia(t, servidor, http.MethodPost, "/api/v1/clientes", corpoDeCliente("11444777000161", "Construtora Alfa", "construtora", nil))
	conferirStatus(t, resposta, http.StatusCreated)

	return decodificarEnvelope[clientedto.ClienteResponse](t, resposta).Dados.ID
}

func corpoDePedido(empresaID, clienteID, produtoID uuid.UUID, modelo string, quantidade int) map[string]any {
	return map[string]any{
		"empresa_id": empresaID,
		"cliente_id": clienteID,
		"modelo":     modelo,
		"itens":      []map[string]any{{"produto_id": produtoID, "quantidade": quantidade, "preco_unitario_centavos": 1500}},
	}
}

func criarPedidoE2E(t *testing.T, servidor *httptest.Server, corpo map[string]any, token string) pedidodto.PedidoResponse {
	t.Helper()

	resposta := enviaComToken(t, servidor, http.MethodPost, "/api/v1/pedidos", corpo, token)
	conferirStatus(t, resposta, http.StatusCreated)

	return decodificarEnvelope[pedidodto.PedidoResponse](t, resposta).Dados
}

func usuarioComercialDasEmpresas(t *testing.T, servidor *httptest.Server, nome, email string, empresaIDs ...uuid.UUID) string {
	t.Helper()

	usuarioID, _ := usuarioComercial(t, servidor, nome, email)
	conferirStatus(t, envia(t, servidor, http.MethodPut, "/api/v1/usuarios/"+usuarioID.String()+"/empresas", map[string]any{
		"acesso_todas": false,
		"empresa_ids":  empresaIDs,
	}), http.StatusOK)

	return autenticar(t, servidor, email, "senhaForte123").Token
}

func TestFluxoDePedidoComReservaEEntregaParcial(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	galpao := criarLocal(t, servidor, mia, "Galpão")
	produtoID := produtoDoTeste(t, servidor)
	caminhoDoProduto := "/api/v1/produtos/" + produtoID.String()
	conferirStatus(t, envia(t, servidor, http.MethodPost, caminhoDoProduto+"/movimentos", map[string]any{
		"local_id": galpao.ID, "tipo": "entrada", "quantidade": 8, "custo_unitario_centavos": 1990,
	}), http.StatusCreated)
	clienteID := clienteDoPedido(t, servidor)

	pedido := criarPedidoE2E(t, servidor, corpoDePedido(mia, clienteID, produtoID, "pronta_entrega", 10), tokenDoTeste)
	if pedido.Numero != 1 || pedido.Status != "proposta" || pedido.Versao != 1 || pedido.ValorTotalCentavos != 15000 {
		t.Fatalf("pedido criado = %+v", pedido)
	}
	caminho := "/api/v1/pedidos/" + pedido.ID.String()

	alteracao := corpoDePedido(mia, clienteID, produtoID, "pronta_entrega", 12)
	delete(alteracao, "empresa_id")
	alteracao["versao"] = 1
	alteracao["prazo_limite"] = "2026-12-20"
	resposta := envia(t, servidor, http.MethodPut, caminho, alteracao)
	conferirStatus(t, resposta, http.StatusOK)
	if alterado := decodificarEnvelope[pedidodto.PedidoResponse](t, resposta).Dados; alterado.Versao != 2 || alterado.ValorTotalCentavos != 18000 ||
		alterado.PrazoLimite == nil || alterado.PrazoLimite.Format("2006-01-02") != "2026-12-20" {
		t.Errorf("pedido alterado = %+v", alterado)
	}

	conferirStatus(t, envia(t, servidor, http.MethodPost, caminho+"/confirmar", map[string]any{"versao": 1}), http.StatusConflict)
	resposta = envia(t, servidor, http.MethodPost, caminho+"/confirmar", map[string]any{"versao": 2})
	conferirStatus(t, resposta, http.StatusOK)
	if confirmado := decodificarEnvelope[pedidodto.PedidoResponse](t, resposta).Dados; confirmado.Status != "confirmado" || confirmado.Versao != 3 {
		t.Errorf("pedido confirmado = %+v", confirmado)
	}
	conferirStatus(t, envia(t, servidor, http.MethodPut, caminho, alteracao), http.StatusConflict)

	resposta = envia(t, servidor, http.MethodGet, caminho, nil)
	conferirStatus(t, resposta, http.StatusOK)
	detalhe := decodificarEnvelope[pedidodto.PedidoDetalheResponse](t, resposta).Dados
	item := detalhe.Itens[0]
	if item.Reservado == nil || *item.Reservado != 8 || item.FaltaCobrir == nil || *item.FaltaCobrir != 4 || len(detalhe.Reservas) != 1 ||
		detalhe.Reservas[0].Origem != "pedido_item" {
		t.Errorf("detalhe depois de confirmar = %+v", detalhe)
	}

	conferirStatus(t, envia(t, servidor, http.MethodPost, caminho+"/entrada", map[string]any{
		"versao": 3, "pago_em": "2026-10-10", "valor_centavos": 100,
	}), http.StatusBadRequest)

	resposta = envia(t, servidor, http.MethodPost, caminho+"/entregas", map[string]any{
		"frete_tipo": "cif", "frete_centavos": 18000, "transportadora": "Rodonaves",
		"itens": []map[string]any{{"pedido_item_id": item.ID, "local_id": galpao.ID, "quantidade": 5}},
	})
	conferirStatus(t, resposta, http.StatusCreated)
	entrega := decodificarEnvelope[pedidodto.EntregaResponse](t, resposta).Dados
	if entrega.Status != "pendente" || entrega.FreteCentavos == nil || *entrega.FreteCentavos != 18000 {
		t.Errorf("entrega criada = %+v", entrega)
	}
	conferirStatus(t, envia(t, servidor, http.MethodPost, caminho+"/entregas", map[string]any{
		"frete_tipo": "fob",
		"itens":      []map[string]any{{"pedido_item_id": item.ID, "local_id": galpao.ID, "quantidade": 8}},
	}), http.StatusBadRequest)

	caminhoDaEntrega := "/api/v1/entregas/" + entrega.ID.String()
	resposta = envia(t, servidor, http.MethodPost, caminhoDaEntrega+"/saiu", map[string]any{"versao": 1})
	conferirStatus(t, resposta, http.StatusOK)
	if saiu := decodificarEnvelope[pedidodto.EntregaResponse](t, resposta).Dados; saiu.Status != "saiu" || saiu.SaiuEm == nil || saiu.Itens[0].MovimentoID == nil {
		t.Errorf("entrega depois da saída = %+v", saiu)
	}

	resposta = envia(t, servidor, http.MethodGet, caminhoDoProduto+"/saldo", nil)
	conferirStatus(t, resposta, http.StatusOK)
	if saldo := decodificarEnvelope[estoquedto.SaldoResponse](t, resposta).Dados; saldo.Saldo != 3 || saldo.Reservado != 3 {
		t.Errorf("saldo depois da saída = %+v, esperado 3 com 3 reservados", saldo)
	}

	resposta = envia(t, servidor, http.MethodGet, caminho, nil)
	conferirStatus(t, resposta, http.StatusOK)
	depois := decodificarEnvelope[pedidodto.PedidoDetalheResponse](t, resposta).Dados
	if depois.Status != "parcialmente_entregue" || depois.Itens[0].QuantidadeEntregue != 5 || *depois.Itens[0].Reservado != 3 {
		t.Errorf("pedido depois da saída = %+v", depois)
	}

	hoje := domainpedido.Hoje(time.Now()).Format("2006-01-02")
	resposta = envia(t, servidor, http.MethodPost, caminhoDaEntrega+"/entregue", map[string]any{"versao": 2, "entregue_em": hoje})
	conferirStatus(t, resposta, http.StatusOK)
	if concluida := decodificarEnvelope[pedidodto.EntregaResponse](t, resposta).Dados; concluida.Status != "entregue" {
		t.Errorf("entrega concluída = %+v", concluida)
	}

	resposta = envia(t, servidor, http.MethodGet, caminho+"/entregas", nil)
	conferirStatus(t, resposta, http.StatusOK)
	if entregas := decodificarEnvelope[[]pedidodto.EntregaResponse](t, resposta).Dados; len(entregas) != 1 {
		t.Errorf("entregas do pedido = %d, esperado 1", len(entregas))
	}

	conferirStatus(t, envia(t, servidor, http.MethodPost, caminho+"/cancelar", map[string]any{"versao": depois.Versao, "motivo": "obra parada"}), http.StatusConflict)
	resposta = envia(t, servidor, http.MethodGet, caminho, nil)
	conferirStatus(t, resposta, http.StatusOK)
	atual := decodificarEnvelope[pedidodto.PedidoDetalheResponse](t, resposta).Dados

	resposta = envia(t, servidor, http.MethodPost, caminho+"/cancelar", map[string]any{"versao": atual.Versao, "motivo": "obra parada"})
	conferirStatus(t, resposta, http.StatusOK)
	if cancelado := decodificarEnvelope[pedidodto.PedidoResponse](t, resposta).Dados; cancelado.Status != "cancelado" || cancelado.CanceladoMotivo == nil {
		t.Errorf("pedido cancelado = %+v", cancelado)
	}
	resposta = envia(t, servidor, http.MethodGet, caminhoDoProduto+"/saldo", nil)
	conferirStatus(t, resposta, http.StatusOK)
	if saldo := decodificarEnvelope[estoquedto.SaldoResponse](t, resposta).Dados; saldo.Saldo != 3 || saldo.Reservado != 0 {
		t.Errorf("saldo depois do cancelamento = %+v, esperado 3 livres", saldo)
	}

	resposta = envia(t, servidor, http.MethodGet, caminho+"/historico", nil)
	conferirStatus(t, resposta, http.StatusOK)
	historico := decodificarEnvelope[[]pedidodto.HistoricoResponse](t, resposta).Dados
	acoes := make([]string, 0, len(historico))
	for _, registro := range historico {
		acoes = append(acoes, registro.Acao)
	}
	esperadas := []string{"criado", "alterado", "confirmado", "entrega_criada", "entrega_saiu", "entrega_concluida", "cancelado"}
	if len(acoes) != len(esperadas) {
		t.Fatalf("ações do histórico = %v, esperado %v", acoes, esperadas)
	}
	for indice := range esperadas {
		if acoes[indice] != esperadas[indice] {
			t.Errorf("ações do histórico = %v, esperado %v", acoes, esperadas)
			break
		}
	}
}

func TestPedidosIsoladosPorEmpresaEPerfil(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	red := criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond").ID
	produtoID := produtoDoTeste(t, servidor)
	clienteID := clienteDoPedido(t, servidor)

	tokenMia := usuarioComercialDasEmpresas(t, servidor, "Ana Vendas", "ana@vendas.exemplo.com", mia)
	tokenFinanceiro := tokenFinanceiroDasEmpresas(t, servidor, "Fábio Financeiro", "fabio@financeiro.exemplo.com", mia)

	daMia := criarPedidoE2E(t, servidor, corpoDePedido(mia, clienteID, produtoID, "importacao", 10), tokenMia)
	daRed := criarPedidoE2E(t, servidor, corpoDePedido(red, clienteID, produtoID, "pronta_entrega", 2), tokenDoTeste)

	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/pedidos", corpoDePedido(red, clienteID, produtoID, "pronta_entrega", 1), tokenMia), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/pedidos", corpoDePedido(mia, clienteID, produtoID, "pronta_entrega", 1), tokenFinanceiro), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/pedidos/"+daRed.ID.String(), nil, tokenMia), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/pedidos?empresa="+red.String(), nil, tokenMia), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/pedidos/"+daRed.ID.String()+"/confirmar", map[string]any{"versao": 1}, tokenMia), http.StatusForbidden)

	resposta := enviaComToken(t, servidor, http.MethodGet, "/api/v1/pedidos", nil, tokenMia)
	conferirStatus(t, resposta, http.StatusOK)
	if pedidos := decodificarPagina[pedidodto.PedidoResponse](t, resposta).Dados; len(pedidos) != 1 || pedidos[0].ID != daMia.ID {
		t.Errorf("pedidos vistos pela MIA = %+v", pedidos)
	}
	resposta = envia(t, servidor, http.MethodGet, "/api/v1/pedidos?numero=1", nil)
	conferirStatus(t, resposta, http.StatusOK)
	if pedidos := decodificarPagina[pedidodto.PedidoResponse](t, resposta).Dados; len(pedidos) != 2 {
		t.Errorf("pedidos número 1 vistos pelo administrador = %d, esperado 2", len(pedidos))
	}

	caminho := "/api/v1/pedidos/" + daMia.ID.String()
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/confirmar", map[string]any{"versao": 1}, tokenMia), http.StatusOK)
	entrada := map[string]any{"versao": 2, "pago_em": domainpedido.Hoje(time.Now()).Format("2006-01-02"), "valor_centavos": 4500}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/entrada", entrada, tokenMia), http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodPost, caminho+"/entrada", entrada, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	comEntrada := decodificarEnvelope[pedidodto.PedidoResponse](t, resposta).Dados
	if comEntrada.EntradaPagaCentavos == nil || *comEntrada.EntradaPagaCentavos != 4500 || comEntrada.PrazoLimite == nil {
		t.Errorf("pedido com entrada = %+v", comEntrada)
	}

	resposta = enviaComToken(t, servidor, http.MethodPost, caminho+"/cancelar", map[string]any{"versao": 3, "motivo": "atraso"}, tokenMia)
	conferirStatus(t, resposta, http.StatusOK)
	if cancelado := decodificarEnvelope[pedidodto.PedidoResponse](t, resposta).Dados; cancelado.ValorADevolverCentavos == nil || *cancelado.ValorADevolverCentavos != 4500 {
		t.Errorf("valor a devolver = %v, esperado 4500", cancelado.ValorADevolverCentavos)
	}

	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/entregas", nil, tokenMia), http.StatusOK)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/entregas?empresa="+red.String(), nil, tokenMia), http.StatusForbidden)
	conferirStatus(t, enviaSemToken(t, servidor, http.MethodGet, "/api/v1/pedidos", nil), http.StatusUnauthorized)
}
