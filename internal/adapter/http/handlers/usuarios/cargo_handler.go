package usuarios

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	usuariosdto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/usuarios"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
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
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	id, err := h.usecase.Create(r.Context(), requisicao.Nome, requisicao.Descricao, requisicao.Administrador, requisicao.Financeiro)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	item, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusCreated, dto.Resposta[usuariosdto.CargoResponse]{
		Dados: usuariosdto.NovaCargoResponse(item),
	})
}

func (h *CargoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	paginacao := resposta.ConsultaPaginacao(r)

	var (
		itens []*domainusuarios.Cargo
		err   error
	)

	if resposta.ConsultaBooleana(r, "ativos") {
		itens, err = h.usecase.ListAtivos(r.Context(), paginacao)
	} else {
		itens, err = h.usecase.List(r.Context(), paginacao)
	}

	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[usuariosdto.CargoResponse]{
		Dados:   usuariosdto.NovaCargoResponses(itens),
		Pagina:  paginacao.Page,
		Tamanho: paginacao.Size,
	})
}

func (h *CargoHandler) Obter(w http.ResponseWriter, r *http.Request) {
	id, ok := resposta.ParametroUUID(w, r, "id")
	if !ok {
		return
	}

	item, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[usuariosdto.CargoResponse]{
		Dados: usuariosdto.NovaCargoResponse(item),
	})
}

func (h *CargoHandler) ObterPorNome(w http.ResponseWriter, r *http.Request) {
	nome := resposta.ConsultaTexto(r, "nome")
	if nome == "" {
		resposta.ResponderErro(w, domain.ErroValidacao("parâmetro nome inválido"))
		return
	}

	item, err := h.usecase.GetByNome(r.Context(), nome)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[usuariosdto.CargoResponse]{
		Dados: usuariosdto.NovaCargoResponse(item),
	})
}

func (h *CargoHandler) Atualizar(w http.ResponseWriter, r *http.Request) {
	id, ok := resposta.ParametroUUID(w, r, "id")
	if !ok {
		return
	}

	var requisicao usuariosdto.AtualizarCargoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	if err := h.usecase.Update(r.Context(), requisicao.ParaCargo(id)); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	item, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[usuariosdto.CargoResponse]{
		Dados: usuariosdto.NovaCargoResponse(item),
	})
}

func (h *CargoHandler) Remover(w http.ResponseWriter, r *http.Request) {
	id, ok := resposta.ParametroUUID(w, r, "id")
	if !ok {
		return
	}

	if err := h.usecase.Delete(r.Context(), id); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverVazio(w, http.StatusNoContent)
}
