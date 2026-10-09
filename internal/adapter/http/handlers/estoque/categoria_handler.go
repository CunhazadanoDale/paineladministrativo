package estoque

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	estoquedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/estoque"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	"github.com/google/uuid"
)

type CategoriaHandler struct {
	usecase portsin.CategoriaUseCase
}

func NewCategoriaHandler(usecase portsin.CategoriaUseCase) *CategoriaHandler {
	return &CategoriaHandler{usecase: usecase}
}

func (h *CategoriaHandler) Criar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao estoquedto.CriarCategoriaRequest
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

	dto.EscreverJSON(w, http.StatusCreated, dto.Resposta[estoquedto.CategoriaResponse]{
		Dados: estoquedto.NovaCategoriaResponse(item),
	})
}

func (h *CategoriaHandler) Listar(w http.ResponseWriter, r *http.Request) {
	paginacao := resposta.ConsultaPaginacao(r)

	ativo, err := resposta.ConsultaBooleanaOpcional(r, "ativo")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	itens, err := h.usecase.Listar(r.Context(), portsin.ListarCategoriasInput{
		Ativo:  ativo,
		Filtro: paginacao,
	})
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[estoquedto.CategoriaResponse]{
		Dados:   estoquedto.NovaCategoriaResponses(itens),
		Pagina:  paginacao.Page,
		Tamanho: paginacao.Size,
	})
}

func (h *CategoriaHandler) Obter(w http.ResponseWriter, r *http.Request) {
	id, ok := resposta.ParametroUUID(w, r, "id")
	if !ok {
		return
	}

	item, err := h.usecase.Obter(r.Context(), id)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[estoquedto.CategoriaResponse]{
		Dados: estoquedto.NovaCategoriaResponse(item),
	})
}

func (h *CategoriaHandler) Alterar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao estoquedto.AlterarCategoriaRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	if err := h.usecase.Alterar(r.Context(), requisicao.ParaInput(usuario.ID, id)); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	h.escreverAtualizada(w, r, id)
}

func (h *CategoriaHandler) Ativar(w http.ResponseWriter, r *http.Request) {
	h.alternarAtivo(w, r, true)
}

func (h *CategoriaHandler) Desativar(w http.ResponseWriter, r *http.Request) {
	h.alternarAtivo(w, r, false)
}

func (h *CategoriaHandler) alternarAtivo(w http.ResponseWriter, r *http.Request, ativo bool) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	if err := h.usecase.AlternarAtivo(r.Context(), portsin.AlternarAtivoCategoriaInput{
		UsuarioID:   usuario.ID,
		CategoriaID: id,
		Ativo:       ativo,
	}); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	h.escreverAtualizada(w, r, id)
}

func (h *CategoriaHandler) escreverAtualizada(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	item, err := h.usecase.Obter(r.Context(), id)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[estoquedto.CategoriaResponse]{
		Dados: estoquedto.NovaCategoriaResponse(item),
	})
}
