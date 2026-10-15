//go:build e2e

package http

import (
	"net/http"
	"testing"
	"time"

	pedidodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/pedido"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
)

func TestConsultasDePendenciasPrazosECancelamentos(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	red := criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond").ID
	galpao := criarLocal(t, servidor, mia, "Galpão")
	produtoID := produtoDoTeste(t, servidor)
	conferirStatus(t, envia(t, servidor, http.MethodPost, "/api/v1/produtos/"+produtoID.String()+"/movimentos", map[string]any{
		"local_id": galpao.ID, "tipo": "entrada", "quantidade": 4, "custo_unitario_centavos": 1990,
	}), http.StatusCreated)
	clienteID := clienteDoPedido(t, servidor)
	tokenRed := usuarioComercialDasEmpresas(t, servidor, "Rui Red", "rui@red.exemplo.com", red)

	ontem := domainpedido.Hoje(time.Now()).AddDate(0, 0, -1).Format("2006-01-02")
	corpo := corpoDePedido(mia, clienteID, produtoID, "pronta_entrega", 10)
	corpo["prazo_limite"] = ontem
	pedido := criarPedidoE2E(t, servidor, corpo, tokenDoTeste)
	caminho := "/api/v1/pedidos/" + pedido.ID.String()
	conferirStatus(t, envia(t, servidor, http.MethodPost, caminho+"/confirmar", map[string]any{"versao": 1}), http.StatusOK)

	resposta := envia(t, servidor, http.MethodGet, "/api/v1/pedidos/pendencias?produto="+produtoID.String(), nil)
	conferirStatus(t, resposta, http.StatusOK)
	pendencias := decodificarPagina[pedidodto.PendenciaResponse](t, resposta).Dados
	if len(pendencias) != 1 || pendencias[0].PedidoID != pedido.ID || pendencias[0].Reservado != 4 || pendencias[0].FaltaCobrir != 6 ||
		pendencias[0].DisponivelNaEmpresa != 0 || pendencias[0].ProdutoNome != "Cimento CP II" {
		t.Errorf("pendências = %+v", pendencias)
	}

	resposta = envia(t, servidor, http.MethodGet, "/api/v1/pedidos/prazos?situacao=vencido", nil)
	conferirStatus(t, resposta, http.StatusOK)
	vencidos := decodificarPagina[pedidodto.PedidoNoPrazoResponse](t, resposta).Dados
	if len(vencidos) != 1 || vencidos[0].ID != pedido.ID || vencidos[0].DiasParaOPrazo == nil || *vencidos[0].DiasParaOPrazo != -1 {
		t.Errorf("vencidos = %+v", vencidos)
	}
	resposta = envia(t, servidor, http.MethodGet, "/api/v1/pedidos/prazos?situacao=a_vencer", nil)
	conferirStatus(t, resposta, http.StatusOK)
	if aVencer := decodificarPagina[pedidodto.PedidoNoPrazoResponse](t, resposta).Dados; len(aVencer) != 0 {
		t.Errorf("a vencer = %d, esperado 0", len(aVencer))
	}
	conferirStatus(t, envia(t, servidor, http.MethodGet, "/api/v1/pedidos/prazos?situacao=atrasado", nil), http.StatusBadRequest)

	conferirStatus(t, envia(t, servidor, http.MethodPost, caminho+"/cancelar", map[string]any{"versao": 2, "motivo": "obra parada"}), http.StatusOK)

	hoje := domainpedido.Hoje(time.Now()).Format("2006-01-02")
	resposta = envia(t, servidor, http.MethodGet, "/api/v1/pedidos/cancelamentos?de="+hoje+"&ate="+hoje, nil)
	conferirStatus(t, resposta, http.StatusOK)
	if cancelados := decodificarPagina[pedidodto.PedidoResponse](t, resposta).Dados; len(cancelados) != 1 || cancelados[0].ID != pedido.ID {
		t.Errorf("cancelados hoje = %+v", cancelados)
	}
	resposta = envia(t, servidor, http.MethodGet, "/api/v1/pedidos/cancelamentos?com_devolucao=true", nil)
	conferirStatus(t, resposta, http.StatusOK)
	if comDevolucao := decodificarPagina[pedidodto.PedidoResponse](t, resposta).Dados; len(comDevolucao) != 0 {
		t.Errorf("cancelados com devolução = %d, esperado 0", len(comDevolucao))
	}

	resposta = envia(t, servidor, http.MethodGet, "/api/v1/pedidos/pendencias", nil)
	conferirStatus(t, resposta, http.StatusOK)
	if depois := decodificarPagina[pedidodto.PendenciaResponse](t, resposta).Dados; len(depois) != 0 {
		t.Errorf("pendências depois do cancelamento = %d, esperado 0", len(depois))
	}

	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/pedidos/pendencias?empresa="+mia.String(), nil, tokenRed), http.StatusForbidden)
	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/pedidos/cancelamentos", nil, tokenRed)
	conferirStatus(t, resposta, http.StatusOK)
	if daRed := decodificarPagina[pedidodto.PedidoResponse](t, resposta).Dados; len(daRed) != 0 {
		t.Errorf("Red viu %d cancelamentos da MIA", len(daRed))
	}
}
