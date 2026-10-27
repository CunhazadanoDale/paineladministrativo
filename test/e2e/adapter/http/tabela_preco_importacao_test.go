//go:build e2e

package http

import (
	"net/http"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	tabelaprecodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/tabelapreco"
)

func TestImportarTabelaDePrecosPorCSV(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	_, tokenComercial := usuarioComercial(t, servidor, "Caio Lima", "caio@exemplo.com")
	raiz := criarCategoria(t, servidor, "Revestimentos", nil)
	produtoID := criarProdutoComCodigo(t, servidor, criarCategoria(t, servidor, "Porcelanatos", &raiz), "Porcelanato 60x60", "POR-60")
	conferirStatus(t, envia(t, servidor, http.MethodPost, "/api/v1/tabela-precos", map[string]any{
		"produto_id": produtoID, "modelo": "pronta_entrega", "preco_centavos": 12000, "vigente_de": "2026-11-01",
	}), http.StatusCreated)

	csv := strings.Join([]string{
		"codigo;modelo;preco;vigente_de;observacao",
		"POR-60;pronta_entrega;\"139,90\";01/12/2026;reajuste",
		"POR-60;importacao;\"99,90\";01/12/2026;",
		"POR-60;pronta_entrega;\"120,00\";01/11/2026;",
		"SEM-CODIGO;pronta_entrega;10;01/12/2026;",
	}, "\r\n")

	conferirStatus(t, enviarCSV(t, servidor, "/api/v1/tabela-precos/importacao", csv, tokenComercial), http.StatusForbidden)

	resposta := enviarCSV(t, servidor, "/api/v1/tabela-precos/importacao?simular=true", csv, tokenDoTeste)
	conferirStatus(t, resposta, http.StatusOK)
	if simulacao := decodificarEnvelope[dto.RelatorioImportacaoResponse](t, resposta).Dados; !simulacao.Simulacao || simulacao.Importadas != 2 || len(simulacao.Rejeitadas) != 2 {
		t.Fatalf("simulação = %+v", simulacao)
	}

	resposta = enviarCSV(t, servidor, "/api/v1/tabela-precos/importacao", csv, tokenDoTeste)
	conferirStatus(t, resposta, http.StatusOK)
	relatorio := decodificarEnvelope[dto.RelatorioImportacaoResponse](t, resposta).Dados
	if relatorio.Importadas != 2 || len(relatorio.Rejeitadas) != 2 ||
		relatorio.Rejeitadas[0].Motivo != "já existe preço deste produto e modelo com início nesta data" ||
		relatorio.Rejeitadas[1].Motivo != "produto com código SEM-CODIGO não encontrado" {
		t.Fatalf("relatório = %+v", relatorio)
	}

	resposta = envia(t, servidor, http.MethodGet, "/api/v1/tabela-precos?produto="+produtoID.String(), nil)
	itens := decodificarPagina[tabelaprecodto.ItemResponse](t, resposta).Dados
	if len(itens) != 3 || itens[0].VigenteDe != "2026-12-01" {
		t.Fatalf("histórico = %+v", itens)
	}
	precos := map[string]int64{}
	for _, item := range itens {
		precos[item.Modelo+"|"+item.VigenteDe] = item.PrecoCentavos
	}
	if precos["pronta_entrega|2026-12-01"] != 13990 || precos["importacao|2026-12-01"] != 9990 || precos["pronta_entrega|2026-11-01"] != 12000 {
		t.Errorf("preços = %v", precos)
	}

	resposta = enviarCSV(t, servidor, "/api/v1/tabela-precos/importacao", csv, tokenDoTeste)
	conferirStatus(t, resposta, http.StatusOK)
	if novamente := decodificarEnvelope[dto.RelatorioImportacaoResponse](t, resposta).Dados; novamente.Importadas != 0 {
		t.Errorf("reimportação gravou %d lançamentos", novamente.Importadas)
	}

	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/tabela-precos/importacao/modelo", nil, tokenDoTeste), http.StatusOK)
}
