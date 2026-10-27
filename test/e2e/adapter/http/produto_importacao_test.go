//go:build e2e

package http

import (
	"net/http"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	estoquedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/estoque"
)

func TestImportarProdutosPorCSV(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	_, tokenComercial := usuarioComercial(t, servidor, "Vera Nunes", "vera@exemplo.com")
	raiz := criarCategoria(t, servidor, "Materiais", nil)
	cimentos := criarCategoria(t, servidor, "Cimentos", &raiz)
	criarProdutoComCodigo(t, servidor, cimentos, "Cimento antigo", "ANT-1")

	csv := strings.Join([]string{
		"nome;codigo;categoria;unidade_medida;preco;preco_promocional;estoque_minimo;peso_kg;descricao;destaque",
		"Cimento CP II 50kg;CIM-50;Cimentos;sc;\"R$ 1.239,90\";;20;\"50,5\";Saco de 50 kg;sim",
		"Argamassa AC-I;;cimentos;sc;;;;;;",
		"Cimento novo;ANT-1;Cimentos;sc;;;;;;",
		"Areia;;Agregados;m3;;;;;;",
	}, "\r\n")

	conferirStatus(t, enviarCSV(t, servidor, "/api/v1/produtos/importacao", csv, tokenComercial), http.StatusForbidden)

	resposta := enviarCSV(t, servidor, "/api/v1/produtos/importacao?simular=true", csv, tokenDoTeste)
	conferirStatus(t, resposta, http.StatusOK)
	if simulacao := decodificarEnvelope[dto.RelatorioImportacaoResponse](t, resposta).Dados; !simulacao.Simulacao || simulacao.Importadas != 2 || len(simulacao.Rejeitadas) != 2 {
		t.Fatalf("simulação = %+v", simulacao)
	}

	resposta = enviarCSV(t, servidor, "/api/v1/produtos/importacao", csv, tokenDoTeste)
	conferirStatus(t, resposta, http.StatusOK)
	relatorio := decodificarEnvelope[dto.RelatorioImportacaoResponse](t, resposta).Dados
	if relatorio.Importadas != 2 || len(relatorio.Rejeitadas) != 2 ||
		relatorio.Rejeitadas[0].Motivo != "já existe um produto com este código" || relatorio.Rejeitadas[1].Motivo != "categoria Agregados não encontrada" {
		t.Fatalf("relatório = %+v", relatorio)
	}

	resposta = envia(t, servidor, http.MethodGet, "/api/v1/produtos?busca=CIM-50", nil)
	produtos := decodificarPagina[estoquedto.ProdutoResponse](t, resposta).Dados
	if len(produtos) != 1 || produtos[0].PrecoCentavos == nil || *produtos[0].PrecoCentavos != 123990 ||
		produtos[0].PesoKg == nil || *produtos[0].PesoKg != 50.5 || !produtos[0].Destaque || produtos[0].CategoriaID != cimentos {
		t.Errorf("produto importado = %+v", produtos)
	}

	resposta = enviarCSV(t, servidor, "/api/v1/produtos/importacao", csv, tokenDoTeste)
	conferirStatus(t, resposta, http.StatusOK)
	if novamente := decodificarEnvelope[dto.RelatorioImportacaoResponse](t, resposta).Dados; novamente.Importadas != 0 {
		t.Errorf("reimportação gravou %d produtos", novamente.Importadas)
	}

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/produtos/importacao/modelo", nil, tokenDoTeste)
	conferirStatus(t, resposta, http.StatusOK)
}
