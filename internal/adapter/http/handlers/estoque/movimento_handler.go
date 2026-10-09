package estoque

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	estoquedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/estoque"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
)

type MovimentoHandler struct {
	usecase portsin.MovimentoUseCase
}

func NewMovimentoHandler(usecase portsin.MovimentoUseCase) *MovimentoHandler {
	return &MovimentoHandler{usecase: usecase}
}

func (h *MovimentoHandler) Movimentar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao estoquedto.MovimentarEstoqueRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	movimento, err := h.usecase.Movimentar(r.Context(), requisicao.ParaInput(usuario.ID, id))
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusCreated, dto.Resposta[estoquedto.MovimentoResponse]{
		Dados: estoquedto.NovoMovimentoResponse(movimento),
	})
}

func (h *MovimentoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	id, ok := resposta.ParametroUUID(w, r, "id")
	if !ok {
		return
	}

	paginacao := resposta.ConsultaPaginacao(r)

	itens, err := h.usecase.Listar(r.Context(), portsin.ListarMovimentosInput{
		ProdutoID: &id,
		Tipo:      resposta.ConsultaTexto(r, "tipo"),
		Filtro:    paginacao,
	})
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[estoquedto.MovimentoResponse]{
		Dados:   estoquedto.NovoMovimentoResponses(itens),
		Pagina:  paginacao.Page,
		Tamanho: paginacao.Size,
	})
}
