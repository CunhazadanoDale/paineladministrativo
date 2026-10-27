package estoque

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	"github.com/google/uuid"
)

type importacaoProdutosFalso struct {
	input    portsin.ImportarProdutosInput
	chamadas int
}

func (f *importacaoProdutosFalso) Importar(_ context.Context, input portsin.ImportarProdutosInput) (*domain.RelatorioImportacao, error) {
	f.input = input
	f.chamadas++

	return domain.NovoRelatorioImportacao(input.Simular, len(input.Linhas))
}

const cabecalhoProdutos = "nome;codigo;categoria;unidade_medida;preco;preco_promocional;estoque_minimo;peso_kg;descricao;destaque"

func TestImportarProdutosRepassaUsuarioELinhas(t *testing.T) {
	usecase := &importacaoProdutosFalso{}
	corpo := cabecalhoProdutos + "\r\nCimento CP II;CIM-50;Cimentos;sc;\"39,90\";;20;50;\"Saco; 50 kg\";sim\r\n"

	resposta := executaProtegido(t, NewImportacaoProdutoHandler(usecase).Importar, http.MethodPost, "/api/v1/produtos/importacao?simular=true", "", corpo)

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d: %s", resposta.Code, resposta.Body.String())
	}
	if usecase.input.UsuarioID == uuid.Nil || !usecase.input.Simular || len(usecase.input.Linhas) != 1 {
		t.Fatalf("input = %+v", usecase.input)
	}
	if linha := usecase.input.Linhas[0]; linha.Numero != 2 || linha.Codigo != "CIM-50" || linha.Preco != "39,90" || linha.Descricao != "Saco; 50 kg" || linha.Destaque != "sim" {
		t.Errorf("linha = %+v", linha)
	}
}

func TestImportarProdutosRecusaCabecalhoErrado(t *testing.T) {
	usecase := &importacaoProdutosFalso{}

	resposta := executaProtegido(t, NewImportacaoProdutoHandler(usecase).Importar, http.MethodPost, "/api/v1/produtos/importacao", "", "nome;preco\n")

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
	if usecase.chamadas != 0 {
		t.Error("não deveria chegar ao usecase")
	}
}

func TestModeloDeProdutosDevolveCabecalho(t *testing.T) {
	resposta := executaProtegido(t, NewImportacaoProdutoHandler(&importacaoProdutosFalso{}).Modelo, http.MethodGet, "/api/v1/produtos/importacao/modelo", "", "")

	if resposta.Code != http.StatusOK || !strings.HasPrefix(resposta.Body.String(), "\xef\xbb\xbf"+cabecalhoProdutos+"\r\n") {
		t.Errorf("status %d, corpo %q", resposta.Code, resposta.Body.String())
	}
}
