//go:build e2e

package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	clientedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/cliente"
)

func enviarCSV(t *testing.T, servidor *httptest.Server, caminho, conteudo, token string) *http.Response {
	t.Helper()

	requisicao, err := http.NewRequest(http.MethodPost, servidor.URL+caminho, strings.NewReader(conteudo))
	if err != nil {
		t.Fatalf("não consegui montar a requisição: %v", err)
	}
	requisicao.Header.Set("Content-Type", "text/csv; charset=utf-8")
	requisicao.Header.Set("Authorization", "Bearer "+token)

	resposta, err := servidor.Client().Do(requisicao)
	if err != nil {
		t.Fatalf("requisição falhou: %v", err)
	}
	t.Cleanup(func() {
		_ = resposta.Body.Close()
	})

	return resposta
}

func TestConverterLeadEmCliente(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	_, tokenComercial := usuarioComercial(t, servidor, "Vera Nunes", "vera@exemplo.com")
	etapaID := criarEtapa(t, servidor, criarFunil(t, servidor, "Vendas"), "Contato")
	leadID := criarLead(t, servidor, etapaID, "Construtora Águia")
	caminho := "/api/v1/leads/" + leadID.String() + "/converter"
	corpo := map[string]any{"documento": "11.222.333/0001-81", "tipo": "construtora", "uf": "PB", "cidade": "João Pessoa"}

	resposta := enviaComToken(t, servidor, http.MethodPost, caminho, corpo, tokenComercial)
	conferirStatus(t, resposta, http.StatusCreated)
	cliente := decodificarEnvelope[clientedto.ClienteResponse](t, resposta).Dados
	if cliente.LeadID == nil || *cliente.LeadID != leadID || cliente.Nome != "Construtora Águia" {
		t.Errorf("cliente convertido = %+v", cliente)
	}

	resposta = enviaComToken(t, servidor, http.MethodPost, caminho, map[string]any{"documento": "33400004000116", "tipo": "loja", "uf": "PB", "cidade": "JP"}, tokenComercial)
	conferirStatus(t, resposta, http.StatusConflict)
	if mensagem := decodificarErro(t, resposta).Erro.Mensagem; mensagem != "lead já convertido em cliente" {
		t.Errorf("mensagem da segunda conversão = %q", mensagem)
	}

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/clientes?lead="+leadID.String(), nil, tokenComercial)
	conferirStatus(t, resposta, http.StatusOK)
	if clientes := decodificarPagina[clientedto.ClienteResponse](t, resposta).Dados; len(clientes) != 1 || clientes[0].ID != cliente.ID {
		t.Errorf("clientes do lead = %+v", clientes)
	}

	conferirStatus(t, enviaComToken(t, servidor, http.MethodDelete, "/api/v1/leads/"+leadID.String(), nil, tokenComercial), http.StatusBadRequest)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/leads/"+cliente.ID.String()+"/converter", corpo, tokenComercial), http.StatusNotFound)
}

func TestImportarClientesPorCSV(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	_, tokenComercial := usuarioComercial(t, servidor, "Vera Nunes", "vera@exemplo.com")
	conferirStatus(t, envia(t, servidor, http.MethodPost, "/api/v1/representantes", map[string]any{"nome": "José Vendas", "documento": "529.982.247-25"}), http.StatusCreated)

	csv := strings.Join([]string{
		"documento;nome;nome_fantasia;tipo;grupo;uf;cidade;email_nf;telefone;representante_documento;exige_numero_pedido_na_nf;proibe_antecipacao;observacoes",
		"11.222.333/0001-81;Construtora Águia;Águia;construtora;Grupo Águia;PB;João Pessoa;nf@aguia.com.br;;52998224725;sim;nao;",
		"33.400.004/0001-16;Construtora Ômega;;construtora;grupo águia;PB;Cabedelo;;;;;sim;\"pagar; dia 10\"",
		"11222333000181;Repetida;;loja;;PB;JP;;;;;;",
		"111.444.777-35;Maria;;construtora;;PB;JP;;;;;;",
	}, "\n")

	conferirStatus(t, enviarCSV(t, servidor, "/api/v1/clientes/importacao", csv, tokenComercial), http.StatusForbidden)

	resposta := enviarCSV(t, servidor, "/api/v1/clientes/importacao?simular=true", csv, tokenDoTeste)
	conferirStatus(t, resposta, http.StatusOK)
	simulacao := decodificarEnvelope[clientedto.RelatorioImportacaoResponse](t, resposta).Dados
	if !simulacao.Simulacao || simulacao.Total != 4 || simulacao.Importadas != 2 || len(simulacao.Rejeitadas) != 2 {
		t.Fatalf("simulação = %+v", simulacao)
	}
	resposta = envia(t, servidor, http.MethodGet, "/api/v1/clientes", nil)
	if clientes := decodificarPagina[clientedto.ClienteResponse](t, resposta).Dados; len(clientes) != 0 {
		t.Fatalf("simulação gravou %d clientes", len(clientes))
	}

	resposta = enviarCSV(t, servidor, "/api/v1/clientes/importacao", csv, tokenDoTeste)
	conferirStatus(t, resposta, http.StatusOK)
	relatorio := decodificarEnvelope[clientedto.RelatorioImportacaoResponse](t, resposta).Dados
	if relatorio.Simulacao || relatorio.Importadas != 2 || len(relatorio.Rejeitadas) != 2 {
		t.Fatalf("relatório = %+v", relatorio)
	}
	motivos := map[int]string{}
	for _, rejeitada := range relatorio.Rejeitadas {
		motivos[rejeitada.Linha] = rejeitada.Motivo
	}
	if motivos[4] != "documento repetido no arquivo (linha 2)" || motivos[5] != "cliente do tipo pf exige CPF, e CPF exige o tipo pf" {
		t.Errorf("motivos das rejeições = %v", motivos)
	}

	resposta = envia(t, servidor, http.MethodGet, "/api/v1/clientes?busca=construtora", nil)
	clientes := decodificarPagina[clientedto.ClienteResponse](t, resposta).Dados
	if len(clientes) != 2 || clientes[0].GrupoID == nil || clientes[1].GrupoID == nil || *clientes[0].GrupoID != *clientes[1].GrupoID {
		t.Fatalf("clientes importados = %+v, esperado os dois no mesmo grupo", clientes)
	}
	for _, cliente := range clientes {
		if cliente.Documento == "33400004000116" && (!cliente.ProibeAntecipacao || cliente.Observacoes != "pagar; dia 10") {
			t.Errorf("cliente Ômega importado = %+v", cliente)
		}
		if cliente.Documento == "11222333000181" && (cliente.RepresentanteID == nil || !cliente.ExigeNumeroPedidoNaNF) {
			t.Errorf("cliente Águia importado = %+v", cliente)
		}
	}

	resposta = enviarCSV(t, servidor, "/api/v1/clientes/importacao", csv, tokenDoTeste)
	conferirStatus(t, resposta, http.StatusOK)
	if novamente := decodificarEnvelope[clientedto.RelatorioImportacaoResponse](t, resposta).Dados; novamente.Importadas != 0 {
		t.Errorf("reimportação gravou %d clientes, esperado nenhum", novamente.Importadas)
	}

	conferirStatus(t, enviarCSV(t, servidor, "/api/v1/clientes/importacao", "documento;nome\n", tokenDoTeste), http.StatusBadRequest)
}
