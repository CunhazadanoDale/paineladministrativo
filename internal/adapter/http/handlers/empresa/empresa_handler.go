package empresa

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	empresadto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/empresa"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/empresa"
	"github.com/google/uuid"
)

type EmpresaHandler struct {
	usecase portsin.EmpresaUseCase
}

func NewEmpresaHandler(usecase portsin.EmpresaUseCase) *EmpresaHandler {
	return &EmpresaHandler{usecase: usecase}
}

func (h *EmpresaHandler) Criar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao empresadto.CriarEmpresaRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	id, err := h.usecase.Criar(r.Context(), requisicao.ParaInput(usuario.ID))
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	h.escreverEmpresa(w, r, http.StatusCreated, id, usuario.ID)
}

func (h *EmpresaHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	ativa, err := resposta.ConsultaBooleanaOpcional(r, "ativas")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	paginacao := resposta.ConsultaPaginacao(r)

	itens, err := h.usecase.Listar(r.Context(), portsin.ListarEmpresasInput{
		UsuarioID: usuario.ID,
		Ativa:     ativa,
		Filtro:    paginacao,
	})
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[empresadto.EmpresaResponse]{
		Dados:   empresadto.NovaEmpresaResponses(itens),
		Pagina:  paginacao.Page,
		Tamanho: paginacao.Size,
	})
}

func (h *EmpresaHandler) Obter(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	h.escreverEmpresa(w, r, http.StatusOK, id, usuario.ID)
}

func (h *EmpresaHandler) Alterar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao empresadto.AlterarEmpresaRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	if err := h.usecase.Alterar(r.Context(), requisicao.ParaInput(usuario.ID, id)); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	h.escreverEmpresa(w, r, http.StatusOK, id, usuario.ID)
}

func (h *EmpresaHandler) Ativar(w http.ResponseWriter, r *http.Request) {
	h.alternarAtiva(w, r, true)
}

func (h *EmpresaHandler) Desativar(w http.ResponseWriter, r *http.Request) {
	h.alternarAtiva(w, r, false)
}

func (h *EmpresaHandler) Remover(w http.ResponseWriter, r *http.Request) {
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

func (h *EmpresaHandler) ObterVinculos(w http.ResponseWriter, r *http.Request) {
	usuario, alvoID, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	vinculos, err := h.usecase.ObterVinculos(r.Context(), alvoID, usuario.ID)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[empresadto.VinculosResponse]{
		Dados: empresadto.NovaVinculosResponse(vinculos),
	})
}

func (h *EmpresaHandler) DefinirVinculos(w http.ResponseWriter, r *http.Request) {
	usuario, alvoID, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao empresadto.VinculosRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	if err := h.usecase.DefinirVinculos(r.Context(), requisicao.ParaInput(usuario.ID, alvoID)); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	vinculos, err := h.usecase.ObterVinculos(r.Context(), alvoID, usuario.ID)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[empresadto.VinculosResponse]{
		Dados: empresadto.NovaVinculosResponse(vinculos),
	})
}

func (h *EmpresaHandler) alternarAtiva(w http.ResponseWriter, r *http.Request, ativa bool) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	if err := h.usecase.AlternarAtiva(r.Context(), portsin.AlternarAtivaEmpresaInput{
		UsuarioID: usuario.ID,
		EmpresaID: id,
		Ativa:     ativa,
	}); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	h.escreverEmpresa(w, r, http.StatusOK, id, usuario.ID)
}

func (h *EmpresaHandler) escreverEmpresa(w http.ResponseWriter, r *http.Request, status int, empresaID, usuarioID uuid.UUID) {
	item, err := h.usecase.Obter(r.Context(), empresaID, usuarioID)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, status, dto.Resposta[empresadto.EmpresaResponse]{
		Dados: empresadto.NovaEmpresaResponse(item),
	})
}
