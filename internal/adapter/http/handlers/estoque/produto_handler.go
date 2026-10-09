package estoque

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	estoquedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/estoque"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	"github.com/google/uuid"
)

type ProdutoHandler struct {
	usecase portsin.ProdutoUseCase
}

func NewProdutoHandler(usecase portsin.ProdutoUseCase) *ProdutoHandler {
	return &ProdutoHandler{usecase: usecase}
}

func (h *ProdutoHandler) Criar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao estoquedto.CriarProdutoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	id, err := h.usecase.Criar(r.Context(), requisicao.ParaInput(usuario.ID))
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	item, err := h.usecase.Obter(r.Context(), id)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusCreated, dto.Resposta[estoquedto.ProdutoResponse]{
		Dados: estoquedto.NovaProdutoResponse(item),
	})
}

func (h *ProdutoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	paginacao := resposta.ConsultaPaginacao(r)

	categoriaID, err := resposta.ConsultaUUID(r, "categoria_id")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	ativo, err := resposta.ConsultaBooleanaOpcional(r, "ativo")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	destaque, err := resposta.ConsultaBooleanaOpcional(r, "destaque")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	itens, err := h.usecase.Listar(r.Context(), portsin.ListarProdutosInput{
		CategoriaID:  categoriaID,
		Ativo:        ativo,
		Destaque:     destaque,
		EstoqueBaixo: resposta.ConsultaBooleana(r, "estoque_baixo"),
		Busca:        resposta.ConsultaTexto(r, "busca"),
		Filtro:       paginacao,
	})
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[estoquedto.ProdutoResponse]{
		Dados:   estoquedto.NovaProdutoResponses(itens),
		Pagina:  paginacao.Page,
		Tamanho: paginacao.Size,
	})
}

func (h *ProdutoHandler) Obter(w http.ResponseWriter, r *http.Request) {
	id, ok := resposta.ParametroUUID(w, r, "id")
	if !ok {
		return
	}

	item, err := h.usecase.Obter(r.Context(), id)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[estoquedto.ProdutoResponse]{
		Dados: estoquedto.NovaProdutoResponse(item),
	})
}

func (h *ProdutoHandler) Saldo(w http.ResponseWriter, r *http.Request) {
	id, ok := resposta.ParametroUUID(w, r, "id")
	if !ok {
		return
	}

	item, err := h.usecase.Obter(r.Context(), id)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[estoquedto.SaldoResponse]{
		Dados: estoquedto.NovoSaldoResponse(item),
	})
}

func (h *ProdutoHandler) Alterar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao estoquedto.AlterarProdutoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	if err := h.usecase.Alterar(r.Context(), requisicao.ParaInput(usuario.ID, id)); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	h.escreverAtualizado(w, r, id)
}

func (h *ProdutoHandler) Ativar(w http.ResponseWriter, r *http.Request) {
	h.alternarAtivo(w, r, true)
}

func (h *ProdutoHandler) Desativar(w http.ResponseWriter, r *http.Request) {
	h.alternarAtivo(w, r, false)
}

func (h *ProdutoHandler) alternarAtivo(w http.ResponseWriter, r *http.Request, ativo bool) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	if err := h.usecase.AlternarAtivo(r.Context(), portsin.AlternarAtivoProdutoInput{
		UsuarioID: usuario.ID,
		ProdutoID: id,
		Ativo:     ativo,
	}); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	h.escreverAtualizado(w, r, id)
}

func (h *ProdutoHandler) AlternarDestaque(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao estoquedto.AlternarDestaqueRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	if err := h.usecase.AlternarDestaque(r.Context(), portsin.AlternarDestaqueProdutoInput{
		UsuarioID: usuario.ID,
		ProdutoID: id,
		Destaque:  requisicao.Destaque,
	}); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	h.escreverAtualizado(w, r, id)
}

func (h *ProdutoHandler) escreverAtualizado(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	item, err := h.usecase.Obter(r.Context(), id)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[estoquedto.ProdutoResponse]{
		Dados: estoquedto.NovaProdutoResponse(item),
	})
}
