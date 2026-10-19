package tabelapreco

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	tabelaprecodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/tabelapreco"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/tabelapreco"
)

type TabelaPrecoHandler struct {
	usecase portsin.TabelaPrecoUseCase
}

func NewTabelaPrecoHandler(usecase portsin.TabelaPrecoUseCase) *TabelaPrecoHandler {
	return &TabelaPrecoHandler{usecase: usecase}
}

func (h *TabelaPrecoHandler) Criar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao tabelaprecodto.CriarItemRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}
	dados, err := requisicao.ParaDados()
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	item, err := h.usecase.Criar(r.Context(), portsin.CriarItemInput{UsuarioID: usuario.ID, Dados: dados})
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusCreated, dto.Resposta[tabelaprecodto.ItemResponse]{Dados: tabelaprecodto.NovoItemResponse(item)})
}

func (h *TabelaPrecoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.ListarItensInput{
		UsuarioID: usuario.ID,
		Modelo:    resposta.ConsultaTexto(r, "modelo"),
		Filtro:    resposta.ConsultaPaginacao(r),
	}
	var err error
	if input.ProdutoID, err = resposta.ConsultaUUID(r, "produto"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	itens, err := h.usecase.Listar(r.Context(), input)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[tabelaprecodto.ItemResponse]{
		Dados: tabelaprecodto.NovoItemResponses(itens), Pagina: input.Filtro.Page, Tamanho: input.Filtro.Size,
	})
}

func (h *TabelaPrecoHandler) Vigentes(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.VigentesInput{
		UsuarioID: usuario.ID,
		Modelo:    resposta.ConsultaTexto(r, "modelo"),
		Busca:     resposta.ConsultaTexto(r, "busca"),
		Filtro:    resposta.ConsultaPaginacao(r),
	}
	var err error
	if input.ProdutoID, err = resposta.ConsultaUUID(r, "produto"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	data, err := tabelaprecodto.ParaData(resposta.ConsultaTexto(r, "data"), "parâmetro data")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	if !data.IsZero() {
		input.Data = &data
	}

	vigentes, err := h.usecase.Vigentes(r.Context(), input)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[tabelaprecodto.PrecoVigenteResponse]{
		Dados: tabelaprecodto.NovoPrecoVigenteResponses(vigentes), Pagina: input.Filtro.Page, Tamanho: input.Filtro.Size,
	})
}

func (h *TabelaPrecoHandler) Remover(w http.ResponseWriter, r *http.Request) {
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
