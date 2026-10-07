package usuarios

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	usuariosdto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/usuarios"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsinautenticacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/autenticacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/usuarios"
	"github.com/google/uuid"
)

type UsuarioHandler struct {
	usecase portsin.UsuarioUseCase
	tokens  portsinautenticacao.TokenService
}

func NewUsuarioHandler(usecase portsin.UsuarioUseCase, tokens portsinautenticacao.TokenService) *UsuarioHandler {
	return &UsuarioHandler{usecase: usecase, tokens: tokens}
}

func (h *UsuarioHandler) Criar(w http.ResponseWriter, r *http.Request) {
	var requisicao usuariosdto.CriarUsuarioRequest
	if !corpoJSON(w, r, &requisicao) {
		return
	}

	id, err := h.usecase.Create(r.Context(), requisicao.Nome, requisicao.Email, requisicao.Senha, requisicao.CargoID)
	if err != nil {
		responderErro(w, err)
		return
	}

	item, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		responderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusCreated, dto.Resposta[usuariosdto.UsuarioResponse]{
		Dados: usuariosdto.NovaUsuarioResponse(item),
	})
}

func (h *UsuarioHandler) Listar(w http.ResponseWriter, r *http.Request) {
	paginacao := consultaPaginacao(r)
	busca := consultaTexto(r, "q")

	var (
		itens []*domainusuarios.Usuario
		err   error
	)

	switch {
	case busca != "":
		itens, err = h.usecase.Search(r.Context(), busca, paginacao)
	case consultaBooleana(r, "ativos"):
		itens, err = h.usecase.ListAtivos(r.Context(), paginacao)
	default:
		itens, err = h.usecase.List(r.Context(), paginacao)
	}

	if err != nil {
		responderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[usuariosdto.UsuarioResponse]{
		Dados:   usuariosdto.NovaUsuarioResponses(itens),
		Pagina:  paginacao.Page,
		Tamanho: paginacao.Size,
	})
}

func (h *UsuarioHandler) Obter(w http.ResponseWriter, r *http.Request) {
	id, ok := parametroUUID(w, r, "id")
	if !ok {
		return
	}

	item, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		responderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[usuariosdto.UsuarioResponse]{
		Dados: usuariosdto.NovaUsuarioResponse(item),
	})
}

func (h *UsuarioHandler) Atualizar(w http.ResponseWriter, r *http.Request) {
	id, ok := parametroUUID(w, r, "id")
	if !ok {
		return
	}

	var requisicao usuariosdto.AtualizarUsuarioRequest
	if !corpoJSON(w, r, &requisicao) {
		return
	}

	if err := h.usecase.Update(r.Context(), requisicao.ParaUsuario(id)); err != nil {
		responderErro(w, err)
		return
	}

	item, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		responderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[usuariosdto.UsuarioResponse]{
		Dados: usuariosdto.NovaUsuarioResponse(item),
	})
}

func (h *UsuarioHandler) Remover(w http.ResponseWriter, r *http.Request) {
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

func (h *UsuarioHandler) Autenticar(w http.ResponseWriter, r *http.Request) {
	var requisicao usuariosdto.AutenticarUsuarioRequest
	if !corpoJSON(w, r, &requisicao) {
		return
	}

	usuario, err := h.usecase.Authenticate(r.Context(), requisicao.Email, requisicao.Senha)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			dto.EscreverErro(w, http.StatusNotFound, "email ou senha inválidos")
			return
		}

		responderErro(w, err)
		return
	}

	login := time.Now().UTC()
	if err := h.usecase.UpdateUltimoLogin(r.Context(), usuario.ID, login); err != nil {
		responderErro(w, err)
		return
	}
	usuario.UltimoLogin = login

	token, expiraEm, err := h.tokens.Gerar(usuario.ID)
	if err != nil {
		responderErro(w, err)
		return
	}

	administrador, err := h.usecase.EhAdministrador(r.Context(), usuario)
	if err != nil {
		responderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[usuariosdto.SessaoResponse]{
		Dados: usuariosdto.SessaoResponse{
			Token:         token,
			ExpiraEm:      expiraEm,
			Administrador: administrador,
			Usuario:       usuariosdto.NovaUsuarioResponse(usuario),
		},
	})
}

func (h *UsuarioHandler) TrocarSenha(w http.ResponseWriter, r *http.Request) {
	id, ok := parametroUUID(w, r, "id")
	if !ok {
		return
	}

	var requisicao usuariosdto.TrocarSenhaRequest
	if !corpoJSON(w, r, &requisicao) {
		return
	}

	if err := h.usecase.UpdateSenha(r.Context(), id, requisicao.NovaSenha); err != nil {
		responderErro(w, err)
		return
	}

	dto.EscreverVazio(w, http.StatusNoContent)
}

func (h *UsuarioHandler) Ativar(w http.ResponseWriter, r *http.Request) {
	h.alterarAtivo(w, r, h.usecase.Ativar)
}

func (h *UsuarioHandler) Desativar(w http.ResponseWriter, r *http.Request) {
	h.alterarAtivo(w, r, h.usecase.Desativar)
}

func (h *UsuarioHandler) alterarAtivo(w http.ResponseWriter, r *http.Request, acao func(context.Context, uuid.UUID) error) {
	id, ok := parametroUUID(w, r, "id")
	if !ok {
		return
	}

	if err := acao(r.Context(), id); err != nil {
		responderErro(w, err)
		return
	}

	item, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		responderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[usuariosdto.UsuarioResponse]{
		Dados: usuariosdto.NovaUsuarioResponse(item),
	})
}
