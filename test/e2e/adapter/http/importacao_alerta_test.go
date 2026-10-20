//go:build e2e

package http

import (
	"net/http"
	"testing"
	"time"

	importacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/importacao"
	pedidodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/pedido"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	"github.com/google/uuid"
)

func TestAlertasEImportacoesDoPedido(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	red := criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond").ID
	tokenComercial := usuarioComercialDasEmpresas(t, servidor, "Carla MIA", "carla@exemplo.com", mia)
	tokenComercialRed := usuarioComercialDasEmpresas(t, servidor, "Rui Red", "rui@exemplo.com", red)
	fornecedor := criarFornecedorE2E(t, servidor, map[string]any{
		"pais": "CN", "identificacao_estrangeira": "91440600MA4UX", "nome": "Foshan Ceramics", "moeda_padrao": "USD",
	}, tokenDoTeste)
	produtoID := produtoDoTeste(t, servidor)
	clienteID := clienteDoPedido(t, servidor)

	corpo := corpoDePedido(mia, clienteID, produtoID, "importacao", 600)
	corpo["entrada_bp"] = 0
	pedido := criarPedidoE2E(t, servidor, corpo, tokenDoTeste)
	caminhoDoPedido := "/api/v1/pedidos/" + pedido.ID.String()
	resposta := envia(t, servidor, http.MethodPost, caminhoDoPedido+"/confirmar", map[string]any{"versao": pedido.Versao})
	conferirStatus(t, resposta, http.StatusOK)
	confirmado := decodificarEnvelope[pedidodto.PedidoResponse](t, resposta).Dados
	if confirmado.PrazoLimite == nil {
		t.Fatalf("pedido sem prazo-limite: %+v", confirmado)
	}
	resposta = envia(t, servidor, http.MethodGet, caminhoDoPedido, nil)
	conferirStatus(t, resposta, http.StatusOK)
	itemDoPedido := decodificarEnvelope[pedidodto.PedidoDetalheResponse](t, resposta).Dados.Itens[0]

	hoje := domainimportacao.Hoje(time.Now())
	tardia := corpoDeImportacao(mia, fornecedor.ID, produtoID)
	tardia["chegada_prevista"] = hoje.AddDate(0, 0, 200).Format("2006-01-02")
	semPrevisao := corpoDeImportacao(mia, fornecedor.ID, produtoID)
	alocar := func(corpoDaImportacao map[string]any, quantidade int) uuid.UUID {
		t.Helper()
		resposta := envia(t, servidor, http.MethodPost, "/api/v1/importacoes", corpoDaImportacao)
		conferirStatus(t, resposta, http.StatusCreated)
		importacao := decodificarEnvelope[importacaodto.ImportacaoDetalheResponse](t, resposta).Dados
		conferirStatus(t, envia(t, servidor, http.MethodPut, "/api/v1/importacoes/"+importacao.ID.String()+"/alocacoes", map[string]any{
			"versao": 1, "alocacoes": []map[string]any{{"importacao_item_id": importacao.Itens[0].ID, "pedido_item_id": itemDoPedido.ID, "quantidade": quantidade}},
		}), http.StatusOK)
		return importacao.ID
	}
	idTardia := alocar(tardia, 300)
	idSemPrevisao := alocar(semPrevisao, 100)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/importacoes/alertas", nil, tokenComercial)
	conferirStatus(t, resposta, http.StatusOK)
	alertas := decodificarEnvelope[importacaodto.AlertasResponse](t, resposta).Dados
	if alertas.FolgaDias != 10 || len(alertas.PrazoEmRisco) != 1 || alertas.PrazoEmRisco[0].ImportacaoID != idTardia ||
		alertas.PrazoEmRisco[0].PedidoID != pedido.ID || alertas.PrazoEmRisco[0].DiasDeAtraso <= 0 {
		t.Errorf("prazo em risco = %+v", alertas.PrazoEmRisco)
	}
	if len(alertas.SemPrevisao) != 1 || alertas.SemPrevisao[0].ImportacaoID != idSemPrevisao {
		t.Errorf("sem previsão = %+v", alertas.SemPrevisao)
	}
	if len(alertas.SemImportacao) != 1 || alertas.SemImportacao[0].SemCobertura != 200 || alertas.SemImportacao[0].Alocado != 400 {
		t.Errorf("sem importação = %+v", alertas.SemImportacao)
	}

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/importacoes/alertas", nil, tokenComercialRed)
	conferirStatus(t, resposta, http.StatusOK)
	if daRed := decodificarEnvelope[importacaodto.AlertasResponse](t, resposta).Dados; len(daRed.PrazoEmRisco)+len(daRed.SemPrevisao)+len(daRed.SemImportacao) != 0 {
		t.Errorf("alertas da Red = %+v", daRed)
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/importacoes/alertas?folga=91", nil, tokenComercial), http.StatusBadRequest)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/importacoes/alertas?empresa="+red.String(), nil, tokenComercial), http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodGet, caminhoDoPedido+"/importacoes", nil, tokenComercial)
	conferirStatus(t, resposta, http.StatusOK)
	if atendimentos := decodificarEnvelope[[]importacaodto.AtendimentoResponse](t, resposta).Dados; len(atendimentos) != 2 ||
		atendimentos[0].ImportacaoNumero != 1 || atendimentos[0].Quantidade != 300 || atendimentos[1].Quantidade != 100 {
		t.Errorf("importações do pedido = %+v", atendimentos)
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, caminhoDoPedido+"/importacoes", nil, tokenComercialRed), http.StatusForbidden)
	conferirStatus(t, envia(t, servidor, http.MethodGet, "/api/v1/pedidos/"+uuid.NewString()+"/importacoes", nil), http.StatusNotFound)
}
