package fornecedor

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	fornecedordto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/fornecedor"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/fornecedor"
)

type ImportacaoHandler struct {
	usecase portsin.ImportacaoUseCase
}

func NewImportacaoHandler(usecase portsin.ImportacaoUseCase) *ImportacaoHandler {
	return &ImportacaoHandler{usecase: usecase}
}

func (h *ImportacaoHandler) Importar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}
	simular, err := resposta.ConsultaBooleanaOpcional(r, "simular")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	linhas, ok := resposta.CorpoCSV(w, r, fornecedordto.CabecalhoImportacao)
	if !ok {
		return
	}

	relatorio, err := h.usecase.Importar(r.Context(), portsin.ImportarFornecedoresInput{
		UsuarioID: usuario.ID,
		Linhas:    fornecedordto.NovasLinhasImportacao(linhas),
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

func (h *ImportacaoHandler) Modelo(w http.ResponseWriter, _ *http.Request) {
	resposta.ModeloCSV(w, "modelo-fornecedores.csv", fornecedordto.CabecalhoImportacao, fornecedordto.ExemploImportacao)
}
