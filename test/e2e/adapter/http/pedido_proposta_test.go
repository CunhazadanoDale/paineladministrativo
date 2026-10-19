//go:build e2e

package http

import (
	"net/http"
	"testing"
	"time"

	pedidodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/pedido"
)

func TestPropostaComValidadeCondicoesFreteEPrecoDeTabela(t *testing.T) {
	servidor, banco := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	produtoID := produtoDoTeste(t, servidor)
	clienteID := clienteDoPedido(t, servidor)

	brasilia := time.FixedZone("BRT", -3*60*60)
	hoje := time.Now().In(brasilia)
	hoje = time.Date(hoje.Year(), hoje.Month(), hoje.Day(), 0, 0, 0, 0, time.UTC)
	conferirStatus(t, envia(t, servidor, http.MethodPost, "/api/v1/tabela-precos", map[string]any{
		"produto_id": produtoID, "modelo": "importacao", "preco_centavos": 1800, "vigente_de": hoje.Format("2006-01-02"),
	}), http.StatusCreated)

	corpo := corpoDePedido(mia, clienteID, produtoID, "importacao", 10)
	corpo["entrada_bp"] = 5000
	corpo["prazo_entrega_dias"] = 90
	corpo["frete_tipo"] = "cif"
	corpo["frete_estimado_centavos"] = 25000
	corpo["validade"] = hoje.AddDate(0, 0, 7).Format("2006-01-02")
	pedido := criarPedidoE2E(t, servidor, corpo, tokenDoTeste)
	if pedido.Revisao != 1 || pedido.Validade == nil || *pedido.Validade != hoje.AddDate(0, 0, 7).Format("2006-01-02") || pedido.Vencida {
		t.Fatalf("proposta = revisão %d, validade %v, vencida %v", pedido.Revisao, pedido.Validade, pedido.Vencida)
	}
	if pedido.Condicao.EntradaBP != 5000 || *pedido.Condicao.PrazoEntregaDias != 90 || *pedido.FreteTipo != "cif" ||
		*pedido.FreteEstimadoCentavos != 25000 || pedido.ValorTotalCentavos != 15000 {
		t.Fatalf("condição e frete = %+v / %v %v / total %d", pedido.Condicao, pedido.FreteTipo, pedido.FreteEstimadoCentavos, pedido.ValorTotalCentavos)
	}
	item := pedido.Itens[0]
	if item.PrecoTabelaCentavos == nil || *item.PrecoTabelaCentavos != 1800 || *item.DescontoCentavos != 300 {
		t.Fatalf("item = %+v", item)
	}

	caminho := "/api/v1/pedidos/" + pedido.ID.String()
	conferirStatus(t, envia(t, servidor, http.MethodPost, "/api/v1/pedidos", map[string]any{
		"empresa_id": mia, "cliente_id": clienteID, "modelo": "pronta_entrega", "prazo_entrega_dias": 10,
		"itens": []map[string]any{{"produto_id": produtoID, "quantidade": 1, "preco_unitario_centavos": 100}},
	}), http.StatusBadRequest)
	conferirStatus(t, envia(t, servidor, http.MethodPost, "/api/v1/pedidos", map[string]any{
		"empresa_id": mia, "cliente_id": clienteID, "modelo": "pronta_entrega", "validade": hoje.AddDate(0, 0, -1).Format("2006-01-02"),
		"itens": []map[string]any{{"produto_id": produtoID, "quantidade": 1, "preco_unitario_centavos": 100}},
	}), http.StatusBadRequest)

	alteracao := corpoDePedido(mia, clienteID, produtoID, "importacao", 12)
	delete(alteracao, "empresa_id")
	alteracao["versao"] = pedido.Versao
	resposta := envia(t, servidor, http.MethodPut, caminho, alteracao)
	conferirStatus(t, resposta, http.StatusOK)
	alterado := decodificarEnvelope[pedidodto.PedidoResponse](t, resposta).Dados
	if alterado.Revisao != 2 || alterado.FreteTipo != nil || alterado.Condicao.EntradaBP != 3000 ||
		*alterado.Validade != hoje.AddDate(0, 0, 15).Format("2006-01-02") {
		t.Fatalf("alterado = revisão %d, frete %v, condição %+v, validade %v", alterado.Revisao, alterado.FreteTipo, alterado.Condicao, alterado.Validade)
	}

	if _, err := banco.Exec(`UPDATE pedido SET validade = $1 WHERE id = $2`, hoje.AddDate(0, 0, -1), pedido.ID); err != nil {
		t.Fatalf("vencer a proposta: %v", err)
	}
	resposta = envia(t, servidor, http.MethodGet, caminho, nil)
	conferirStatus(t, resposta, http.StatusOK)
	if vencida := decodificarEnvelope[pedidodto.PedidoDetalheResponse](t, resposta).Dados; !vencida.Vencida {
		t.Fatalf("a proposta deveria aparecer vencida: %+v", vencida.PedidoResponse)
	}
	resposta = envia(t, servidor, http.MethodPost, caminho+"/confirmar", map[string]any{"versao": alterado.Versao})
	conferirStatus(t, resposta, http.StatusConflict)
	if mensagem := decodificarErro(t, resposta).Erro.Mensagem; len(mensagem) < 16 || mensagem[:16] != "proposta vencida" {
		t.Fatalf("mensagem = %q", mensagem)
	}
}

func TestDocumentoDaPropostaEModelosDeVenda(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA")
	red := criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond").ID
	produtoID := produtoDoTeste(t, servidor)
	clienteID := clienteDoPedido(t, servidor)
	tokenRed := usuarioComercialDasEmpresas(t, servidor, "Rui", "rui@exemplo.com", red)

	corpo := corpoDePedido(mia.ID, clienteID, produtoID, "pronta_entrega", 10)
	corpo["frete_tipo"] = "fob"
	pedido := criarPedidoE2E(t, servidor, corpo, tokenDoTeste)

	resposta := envia(t, servidor, http.MethodGet, "/api/v1/pedidos/"+pedido.ID.String()+"/proposta", nil)
	conferirStatus(t, resposta, http.StatusOK)
	documento := decodificarEnvelope[pedidodto.DocumentoPropostaResponse](t, resposta).Dados
	if documento.Identificacao != "Proposta 1-R1" || documento.Empresa.ID != mia.ID || documento.Empresa.CNPJ != "11.222.333/0001-81" ||
		documento.Cliente.ID != clienteID || documento.Cliente.Nome != "Construtora Alfa" {
		t.Fatalf("documento = %+v", documento)
	}
	if len(documento.Itens) != 1 || documento.Itens[0].ProdutoID != produtoID || documento.Itens[0].Nome == "" || documento.TotalCentavos != 15000 {
		t.Fatalf("itens = %+v", documento.Itens)
	}
	esperadas := []string{
		"Pagamento em boleto com vencimento em 30 dias após a entrega.",
		"Frete FOB (por conta do cliente).",
		"Proposta válida até " + (*documento.Validade)[8:10] + "/" + (*documento.Validade)[5:7] + "/" + (*documento.Validade)[0:4] + ".",
	}
	if len(documento.Condicoes) != 3 || documento.Condicoes[0] != esperadas[0] || documento.Condicoes[1] != esperadas[1] || documento.Condicoes[2] != esperadas[2] {
		t.Fatalf("condições = %#v", documento.Condicoes)
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/pedidos/"+pedido.ID.String()+"/proposta", nil, tokenRed), http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/modelos-venda", nil, tokenRed)
	conferirStatus(t, resposta, http.StatusOK)
	if modelos := decodificarEnvelope[[]pedidodto.ModeloDeVendaResponse](t, resposta).Dados; len(modelos) != 2 ||
		modelos[0].Modelo != "pronta_entrega" || *modelos[0].Condicao.PrazoBoletoDias != 30 {
		t.Fatalf("modelos = %+v", modelos)
	}
}
