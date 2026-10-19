package cliente

import (
	"context"
	"net/http"

	clientedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/cliente"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/cliente"
	"github.com/google/uuid"
)

type GrupoHandler struct {
	usecase portsin.GrupoUseCase
}

func NewGrupoHandler(usecase portsin.GrupoUseCase) *GrupoHandler {
	return &GrupoHandler{usecase: usecase}
}

func (h *GrupoHandler) Criar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao clientedto.GrupoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	grupo, err := h.usecase.Criar(r.Context(), portsin.CriarGrupoInput{UsuarioID: usuario.ID, Nome: requisicao.Nome})
	escreverItem(w, http.StatusCreated, grupo, err, clientedto.NovoGrupoResponse)
}

func (h *GrupoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	ativo, err := resposta.ConsultaBooleanaOpcional(r, "ativos")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	paginacao := resposta.ConsultaPaginacao(r)
	itens, err := h.usecase.Listar(r.Context(), portsin.ListarGruposInput{
		Busca:  resposta.ConsultaTexto(r, "busca"),
		Ativo:  ativo,
		Filtro: paginacao,
	})
	escreverPagina(w, paginacao, itens, err, clientedto.NovoGrupoResponses)
}

func (h *GrupoHandler) Obter(w http.ResponseWriter, r *http.Request) {
	id, ok := resposta.ParametroUUID(w, r, "id")
	if !ok {
		return
	}

	grupo, err := h.usecase.Obter(r.Context(), id)
	escreverItem(w, http.StatusOK, grupo, err, clientedto.NovoGrupoResponse)
}

func (h *GrupoHandler) Renomear(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao clientedto.GrupoRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	grupo, err := h.usecase.Renomear(r.Context(), portsin.RenomearGrupoInput{UsuarioID: usuario.ID, GrupoID: id, Nome: requisicao.Nome})
	escreverItem(w, http.StatusOK, grupo, err, clientedto.NovoGrupoResponse)
}

func (h *GrupoHandler) Ativar(w http.ResponseWriter, r *http.Request) {
	alterarSituacao(w, r, h.usecase.Ativar, clientedto.NovoGrupoResponse)
}

func (h *GrupoHandler) Desativar(w http.ResponseWriter, r *http.Request) {
	alterarSituacao(w, r, h.usecase.Desativar, clientedto.NovoGrupoResponse)
}

type RepresentanteHandler struct {
	usecase portsin.RepresentanteUseCase
}

func NewRepresentanteHandler(usecase portsin.RepresentanteUseCase) *RepresentanteHandler {
	return &RepresentanteHandler{usecase: usecase}
}

func (h *RepresentanteHandler) Criar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao clientedto.RepresentanteRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	representante, err := h.usecase.Criar(r.Context(), portsin.CriarRepresentanteInput{UsuarioID: usuario.ID, Dados: requisicao.ParaDados()})
	escreverItem(w, http.StatusCreated, representante, err, clientedto.NovoRepresentanteResponse)
}

func (h *RepresentanteHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	ativo, err := resposta.ConsultaBooleanaOpcional(r, "ativos")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	paginacao := resposta.ConsultaPaginacao(r)
	itens, err := h.usecase.Listar(r.Context(), portsin.ListarRepresentantesInput{
		UsuarioID: usuario.ID,
		Busca:     resposta.ConsultaTexto(r, "busca"),
		Ativo:     ativo,
		Filtro:    paginacao,
	})
	escreverPagina(w, paginacao, itens, err, clientedto.NovoRepresentanteResponses)
}

func (h *RepresentanteHandler) Obter(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	representante, err := h.usecase.Obter(r.Context(), id, usuario.ID)
	escreverItem(w, http.StatusOK, representante, err, clientedto.NovoRepresentanteResponse)
}

func (h *RepresentanteHandler) Alterar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao clientedto.RepresentanteRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	representante, err := h.usecase.Alterar(r.Context(), portsin.AlterarRepresentanteInput{
		UsuarioID:       usuario.ID,
		RepresentanteID: id,
		Dados:           requisicao.ParaDados(),
	})
	escreverItem(w, http.StatusOK, representante, err, clientedto.NovoRepresentanteResponse)
}

func (h *RepresentanteHandler) Ativar(w http.ResponseWriter, r *http.Request) {
	alterarSituacao(w, r, h.usecase.Ativar, clientedto.NovoRepresentanteResponse)
}

func (h *RepresentanteHandler) Desativar(w http.ResponseWriter, r *http.Request) {
	alterarSituacao(w, r, h.usecase.Desativar, clientedto.NovoRepresentanteResponse)
}

func (h *RepresentanteHandler) DefinirFornecedor(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao clientedto.DefinirFornecedorRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	fornecedorID, err := requisicao.Fornecedor()
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	representante, err := h.usecase.DefinirFornecedor(r.Context(), portsin.DefinirFornecedorInput{
		UsuarioID: usuario.ID, RepresentanteID: id, FornecedorID: fornecedorID,
	})
	escreverItem(w, http.StatusOK, representante, err, clientedto.NovoRepresentanteResponse)
}

type ObraHandler struct {
	usecase portsin.ObraUseCase
}

func NewObraHandler(usecase portsin.ObraUseCase) *ObraHandler {
	return &ObraHandler{usecase: usecase}
}

func (h *ObraHandler) Criar(w http.ResponseWriter, r *http.Request) {
	usuario, clienteID, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao clientedto.ObraRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	obra, err := h.usecase.Criar(r.Context(), portsin.CriarObraInput{UsuarioID: usuario.ID, ClienteID: clienteID, Dados: requisicao.ParaDados()})
	escreverItem(w, http.StatusCreated, obra, err, clientedto.NovaObraResponse)
}

func (h *ObraHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuario, clienteID, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	ativo, err := resposta.ConsultaBooleanaOpcional(r, "ativas")
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	paginacao := resposta.ConsultaPaginacao(r)
	itens, err := h.usecase.Listar(r.Context(), portsin.ListarObrasInput{
		UsuarioID: usuario.ID,
		ClienteID: clienteID,
		Ativo:     ativo,
		Filtro:    paginacao,
	})
	escreverPagina(w, paginacao, itens, err, clientedto.NovaObraResponses)
}

func (h *ObraHandler) Obter(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	obra, err := h.usecase.Obter(r.Context(), id, usuario.ID)
	escreverItem(w, http.StatusOK, obra, err, clientedto.NovaObraResponse)
}

func (h *ObraHandler) Alterar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao clientedto.ObraRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	obra, err := h.usecase.Alterar(r.Context(), portsin.AlterarObraInput{UsuarioID: usuario.ID, ObraID: id, Dados: requisicao.ParaDados()})
	escreverItem(w, http.StatusOK, obra, err, clientedto.NovaObraResponse)
}

func (h *ObraHandler) Ativar(w http.ResponseWriter, r *http.Request) {
	alterarSituacao(w, r, h.usecase.Ativar, clientedto.NovaObraResponse)
}

func (h *ObraHandler) Desativar(w http.ResponseWriter, r *http.Request) {
	alterarSituacao(w, r, h.usecase.Desativar, clientedto.NovaObraResponse)
}

func alterarSituacao[T any, R any](
	w http.ResponseWriter,
	r *http.Request,
	alterar func(ctx context.Context, id, usuarioID uuid.UUID) (T, error),
	converter func(T) R,
) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	item, err := alterar(r.Context(), id, usuario.ID)
	escreverItem(w, http.StatusOK, item, err, converter)
}
