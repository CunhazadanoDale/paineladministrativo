package financeiro

import (
	"context"
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	financeirodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/financeiro"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/financeiro"
)

type CategoriaFinanceiraHandler struct {
	usecase portsin.CategoriaFinanceiraUseCase
}

func NewCategoriaFinanceiraHandler(usecase portsin.CategoriaFinanceiraUseCase) *CategoriaFinanceiraHandler {
	return &CategoriaFinanceiraHandler{usecase: usecase}
}

func (h *CategoriaFinanceiraHandler) Criar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao financeirodto.CriarCategoriaFinanceiraRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	categoria, err := h.usecase.Criar(r.Context(), requisicao.ParaInput(usuario.ID))
	escreverCategoria(w, http.StatusCreated, categoria, err)
}

func (h *CategoriaFinanceiraHandler) Listar(w http.ResponseWriter, r *http.Request) {
	pai, err := resposta.ConsultaUUID(r, "pai")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	somenteRaiz, err := resposta.ConsultaBooleanaOpcional(r, "raiz")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	ativa, err := resposta.ConsultaBooleanaOpcional(r, "ativa")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	paginacao := resposta.ConsultaPaginacao(r)
	categorias, err := h.usecase.Listar(r.Context(), portsin.ListarCategoriasFinanceirasInput{
		Pai:         pai,
		SomenteRaiz: somenteRaiz != nil && *somenteRaiz,
		Natureza:    resposta.ConsultaTexto(r, "natureza"),
		LinhaDRE:    resposta.ConsultaTexto(r, "linha_dre"),
		Ativa:       ativa,
		Busca:       resposta.ConsultaTexto(r, "busca"),
		Filtro:      paginacao,
	})
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[financeirodto.CategoriaFinanceiraResponse]{
		Dados:   financeirodto.NovasCategoriaFinanceiraResponses(categorias),
		Pagina:  paginacao.Page,
		Tamanho: paginacao.Size,
	})
}

func (h *CategoriaFinanceiraHandler) Arvore(w http.ResponseWriter, r *http.Request) {
	somenteAtivas, err := resposta.ConsultaBooleanaOpcional(r, "ativas")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	plano, err := h.usecase.Plano(r.Context(), somenteAtivas != nil && *somenteAtivas)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[[]financeirodto.RamoDoPlanoResponse]{Dados: financeirodto.NovoPlanoResponse(plano)})
}

func (h *CategoriaFinanceiraHandler) Obter(w http.ResponseWriter, r *http.Request) {
	id, ok := resposta.ParametroUUID(w, r, "id")
	if !ok {
		return
	}

	categoria, err := h.usecase.Obter(r.Context(), id)
	escreverCategoria(w, http.StatusOK, categoria, err)
}

func (h *CategoriaFinanceiraHandler) Alterar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao financeirodto.AlterarCategoriaFinanceiraRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	categoria, err := h.usecase.Alterar(r.Context(), portsin.AlterarCategoriaFinanceiraInput{
		UsuarioID:   usuario.ID,
		CategoriaID: id,
		Dados:       requisicao.ParaDados(),
		Versao:      requisicao.Versao,
	})
	escreverCategoria(w, http.StatusOK, categoria, err)
}

func (h *CategoriaFinanceiraHandler) Ativar(w http.ResponseWriter, r *http.Request) {
	h.mudarSituacao(w, r, h.usecase.Ativar)
}

func (h *CategoriaFinanceiraHandler) Desativar(w http.ResponseWriter, r *http.Request) {
	h.mudarSituacao(w, r, h.usecase.Desativar)
}

func (h *CategoriaFinanceiraHandler) Remover(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	if err := h.usecase.Remover(r.Context(), id, usuario.ID); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverVazio(w, http.StatusNoContent)
}

func (h *CategoriaFinanceiraHandler) mudarSituacao(
	w http.ResponseWriter,
	r *http.Request,
	mudar func(context.Context, portsin.MudarSituacaoCategoriaFinanceiraInput) (*domainfinanceiro.CategoriaFinanceira, error),
) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao financeirodto.SituacaoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	categoria, err := mudar(r.Context(), portsin.MudarSituacaoCategoriaFinanceiraInput{
		UsuarioID:   usuario.ID,
		CategoriaID: id,
		Versao:      requisicao.Versao,
	})
	escreverCategoria(w, http.StatusOK, categoria, err)
}

func escreverCategoria(w http.ResponseWriter, status int, categoria *domainfinanceiro.CategoriaFinanceira, err error) {
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, status, dto.Resposta[financeirodto.CategoriaFinanceiraResponse]{
		Dados: financeirodto.NovaCategoriaFinanceiraResponse(categoria),
	})
}
