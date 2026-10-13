package cliente

import (
	"net/http"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	clientedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/cliente"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/cliente"
	"github.com/google/uuid"
)

type ClienteHandler struct {
	usecase portsin.ClienteUseCase
}

func NewClienteHandler(usecase portsin.ClienteUseCase) *ClienteHandler {
	return &ClienteHandler{usecase: usecase}
}

func (h *ClienteHandler) Criar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	var requisicao clientedto.CriarClienteRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	cliente, err := h.usecase.Criar(r.Context(), requisicao.ParaInput(usuario.ID))
	escreverItem(w, http.StatusCreated, cliente, err, clientedto.NovoClienteResponse)
}

func (h *ClienteHandler) Listar(w http.ResponseWriter, r *http.Request) {
	usuario, ok := resposta.UsuarioDoContexto(w, r)
	if !ok {
		return
	}

	input := portsin.ListarClientesInput{
		UsuarioID: usuario.ID,
		Busca:     resposta.ConsultaTexto(r, "busca"),
		Tipo:      resposta.ConsultaTexto(r, "tipo"),
		UF:        resposta.ConsultaTexto(r, "uf"),
		Filtro:    resposta.ConsultaPaginacao(r),
	}

	var err error
	for nome, destino := range map[string]**uuid.UUID{"grupo": &input.GrupoID, "representante": &input.RepresentanteID, "lead": &input.LeadID} {
		if *destino, err = resposta.ConsultaUUID(r, nome); err != nil {
			resposta.ResponderErro(w, err)
			return
		}
	}
	if input.Ativo, err = resposta.ConsultaBooleanaOpcional(r, "ativo"); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	itens, err := h.usecase.Listar(r.Context(), input)
	escreverPagina(w, input.Filtro, itens, err, clientedto.NovoClienteResponses)
}

func (h *ClienteHandler) Obter(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	cliente, err := h.usecase.Obter(r.Context(), id, usuario.ID)
	escreverItem(w, http.StatusOK, cliente, err, clientedto.NovoClienteResponse)
}

func (h *ClienteHandler) Alterar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	var requisicao clientedto.AlterarClienteRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	cliente, err := h.usecase.Alterar(r.Context(), requisicao.ParaInput(usuario.ID, id))
	escreverItem(w, http.StatusOK, cliente, err, clientedto.NovoClienteResponse)
}

func (h *ClienteHandler) Ativar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	cliente, err := h.usecase.Ativar(r.Context(), id, usuario.ID)
	escreverItem(w, http.StatusOK, cliente, err, clientedto.NovoClienteResponse)
}

func (h *ClienteHandler) Desativar(w http.ResponseWriter, r *http.Request) {
	usuario, id, ok := resposta.UsuarioEId(w, r)
	if !ok {
		return
	}

	cliente, err := h.usecase.Desativar(r.Context(), id, usuario.ID)
	escreverItem(w, http.StatusOK, cliente, err, clientedto.NovoClienteResponse)
}

func (h *ClienteHandler) Remover(w http.ResponseWriter, r *http.Request) {
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
