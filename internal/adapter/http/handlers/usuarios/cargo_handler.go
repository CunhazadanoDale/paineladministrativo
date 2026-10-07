package usuarios

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	usuariosdto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/usuarios"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/usuarios"
)

type CargoHandler struct {
	usecase portsin.CargoUseCase
}

func NewCargoHandler(usecase portsin.CargoUseCase) *CargoHandler {
	return &CargoHandler{usecase: usecase}
}

func (h *CargoHandler) Criar(w http.ResponseWriter, r *http.Request) {
	var requisicao usuariosdto.CriarCargoRequest
	if !corpoJSON(w, r, &requisicao) {
		return
	}

	id, err := h.usecase.Create(r.Context(), requisicao.Nome, requisicao.Descricao, requisicao.Administrador, requisicao.Financeiro)
	if err != nil {
		responderErro(w, err)
		return
	}

	item, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		responderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusCreated, dto.Resposta[usuariosdto.CargoResponse]{
		Dados: usuariosdto.NovaCargoResponse(item),
	})
}

func (h *CargoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	paginacao := consultaPaginacao(r)

	var (
		itens []*domainusuarios.Cargo
		err   error
	)

	if consultaBooleana(r, "ativos") {
		itens, err = h.usecase.ListAtivos(r.Context(), paginacao)
	} else {
		itens, err = h.usecase.List(r.Context(), paginacao)
	}

	if err != nil {
		responderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[usuariosdto.CargoResponse]{
		Dados:   usuariosdto.NovaCargoResponses(itens),
		Pagina:  paginacao.Page,
		Tamanho: paginacao.Size,
	})
}

func (h *CargoHandler) Obter(w http.ResponseWriter, r *http.Request) {
	id, ok := parametroUUID(w, r, "id")
	if !ok {
		return
	}

	item, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		responderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[usuariosdto.CargoResponse]{
		Dados: usuariosdto.NovaCargoResponse(item),
	})
}

func (h *CargoHandler) ObterPorNome(w http.ResponseWriter, r *http.Request) {
	nome := consultaTexto(r, "nome")
	if nome == "" {
		responderErro(w, domain.ErroValidacao("parâmetro nome inválido"))
		return
	}

	item, err := h.usecase.GetByNome(r.Context(), nome)
	if err != nil {
		responderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[usuariosdto.CargoResponse]{
		Dados: usuariosdto.NovaCargoResponse(item),
	})
}

func (h *CargoHandler) Atualizar(w http.ResponseWriter, r *http.Request) {
	id, ok := parametroUUID(w, r, "id")
	if !ok {
		return
	}

	var requisicao usuariosdto.AtualizarCargoRequest
	if !corpoJSON(w, r, &requisicao) {
		return
	}

	if err := h.usecase.Update(r.Context(), requisicao.ParaCargo(id)); err != nil {
		responderErro(w, err)
		return
	}

	item, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		responderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[usuariosdto.CargoResponse]{
		Dados: usuariosdto.NovaCargoResponse(item),
	})
}

func (h *CargoHandler) Remover(w http.ResponseWriter, r *http.Request) {
	id, ok := parametroUUID(w, r, "id")
	if !ok {
		return
	}

	if err := h.usecase.Delete(r.Context(), id); err != nil {
		responderErro(w, err)
		return
	}

	dto.EscreverVazio(w, http.StatusNoContent)
}
