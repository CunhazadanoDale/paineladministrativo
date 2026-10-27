//go:build e2e

package http

import (
	"net/http"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	fornecedordto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/fornecedor"
)

func TestImportarFornecedoresPorCSV(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	_, tokenComercial := usuarioComercial(t, servidor, "Vera Nunes", "vera@exemplo.com")
	tokenFinanceiro := tokenFinanceiroDasEmpresas(t, servidor, "Fabi", "fabi@exemplo.com")
	criarFornecedorE2E(t, servidor, map[string]any{"documento": "33.400.004/0001-16", "nome": "Já cadastrado"}, tokenFinanceiro)

	csv := strings.Join([]string{
		"pais;documento;identificacao_estrangeira;nome;nome_fantasia;moeda_padrao;uf;cidade;contato_nome;email;telefone;observacoes",
		"BR;11.222.333/0001-81;;Cerâmica Paraíba LTDA;Cerâmica PB;BRL;PB;João Pessoa;Maria;vendas@ceramicapb.com.br;83 3333-0000;",
		"cn;;91330200ma2x;Foshan Ceramics Co., Ltd.;;usd;;;Li Wei;sales@exemplo.cn;;\"pagar; em USD\"",
		";33400004000116;;Duplicado;;;;;;;;",
		"BR;;;Sem documento;;;;;;;;",
	}, "\r\n")

	conferirStatus(t, enviarCSV(t, servidor, "/api/v1/fornecedores/importacao", csv, tokenComercial), http.StatusForbidden)

	resposta := enviarCSV(t, servidor, "/api/v1/fornecedores/importacao?simular=true", csv, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if simulacao := decodificarEnvelope[dto.RelatorioImportacaoResponse](t, resposta).Dados; !simulacao.Simulacao || simulacao.Importadas != 2 || len(simulacao.Rejeitadas) != 2 {
		t.Fatalf("simulação = %+v", simulacao)
	}

	resposta = enviarCSV(t, servidor, "/api/v1/fornecedores/importacao", csv, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	relatorio := decodificarEnvelope[dto.RelatorioImportacaoResponse](t, resposta).Dados
	if relatorio.Importadas != 2 || len(relatorio.Rejeitadas) != 2 || relatorio.Rejeitadas[0].Motivo != "já existe um fornecedor com este documento" {
		t.Fatalf("relatório = %+v", relatorio)
	}

	resposta = envia(t, servidor, http.MethodGet, "/api/v1/fornecedores?pais=CN", nil)
	estrangeiros := decodificarPagina[fornecedordto.FornecedorResponse](t, resposta).Dados
	if len(estrangeiros) != 1 || estrangeiros[0].MoedaPadrao != "USD" || estrangeiros[0].IdentificacaoEstrangeira == nil ||
		*estrangeiros[0].IdentificacaoEstrangeira != "91330200MA2X" || estrangeiros[0].Observacoes != "pagar; em USD" {
		t.Errorf("fornecedor estrangeiro = %+v", estrangeiros)
	}

	resposta = enviarCSV(t, servidor, "/api/v1/fornecedores/importacao", csv, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if novamente := decodificarEnvelope[dto.RelatorioImportacaoResponse](t, resposta).Dados; novamente.Importadas != 0 {
		t.Errorf("reimportação gravou %d fornecedores", novamente.Importadas)
	}

	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/fornecedores/importacao/modelo", nil, tokenFinanceiro), http.StatusOK)
}
