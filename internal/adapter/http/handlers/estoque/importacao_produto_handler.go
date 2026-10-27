package estoque

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	estoquedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/estoque"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
)

type ImportacaoProdutoHandler struct {
	usecase portsin.ImportacaoProdutoUseCase
}

func NewImportacaoProdutoHandler(usecase portsin.ImportacaoProdutoUseCase) *ImportacaoProdutoHandler {
	return &ImportacaoProdutoHandler{usecase: usecase}
}

func (h *ImportacaoProdutoHandler) Importar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}
	simular, err := resposta.ConsultaBooleanaOpcional(r, "simular")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	linhas, ok := resposta.CorpoCSV(w, r, estoquedto.CabecalhoImportacaoProduto)
	if !ok {
		return
	}

	relatorio, err := h.usecase.Importar(r.Context(), portsin.ImportarProdutosInput{
		UsuarioID: usuario.ID,
		Linhas:    estoquedto.NovasLinhasImportacaoProduto(linhas),
		Simular:   simular != nil && *simular,
	})
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[dto.RelatorioImportacaoResponse]{
		Dados: dto.NovoRelatorioImportacaoResponse(relatorio),
	})
}

func (h *ImportacaoProdutoHandler) Modelo(w http.ResponseWriter, _ *http.Request) {
	resposta.ModeloCSV(w, "modelo-produtos.csv", estoquedto.CabecalhoImportacaoProduto, estoquedto.ExemploImportacaoProduto)
}
