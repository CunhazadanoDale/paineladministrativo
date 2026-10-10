package usuarios

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	usuariosdto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/usuarios"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/handlers/resposta"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/middleware"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsinautenticacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/autenticacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/usuarios"
	"github.com/google/uuid"
)

type UsuarioHandler struct {
	usecase    portsin.UsuarioUseCase
	tokens     portsinautenticacao.TokenService
	tentativas *middleware.LimitadorDeTentativas
}

func NewUsuarioHandler(usecase portsin.UsuarioUseCase, tokens portsinautenticacao.TokenService, tentativas *middleware.LimitadorDeTentativas) *UsuarioHandler {
	return &UsuarioHandler{usecase: usecase, tokens: tokens, tentativas: tentativas}
}

func (h *UsuarioHandler) Criar(w http.ResponseWriter, r *http.Request) {
	var requisicao usuariosdto.CriarUsuarioRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	id, err := h.usecase.Criar(r.Context(), requisicao.Nome, requisicao.Email, requisicao.Senha, requisicao.CargoID)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	item, err := h.usecase.Obter(r.Context(), id)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusCreated, dto.Resposta[usuariosdto.UsuarioResponse]{
		Dados: usuariosdto.NovaUsuarioResponse(item),
	})
}

func (h *UsuarioHandler) Listar(w http.ResponseWriter, r *http.Request) {
	paginacao := resposta.ConsultaPaginacao(r)
	busca := resposta.ConsultaTexto(r, "q")

	var (
		itens []*domainusuarios.Usuario
		err   error
	)

	switch {
	case busca != "":
		itens, err = h.usecase.Buscar(r.Context(), busca, paginacao)
	case resposta.ConsultaBooleana(r, "ativos"):
		itens, err = h.usecase.ListarAtivos(r.Context(), paginacao)
	default:
		itens, err = h.usecase.Listar(r.Context(), paginacao)
	}

	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Paginado[usuariosdto.UsuarioResponse]{
		Dados:   usuariosdto.NovaUsuarioResponses(itens),
		Pagina:  paginacao.Page,
		Tamanho: paginacao.Size,
	})
}

func (h *UsuarioHandler) Obter(w http.ResponseWriter, r *http.Request) {
	id, ok := resposta.ParametroUUID(w, r, "id")
	if !ok {
		return
	}

	item, err := h.usecase.Obter(r.Context(), id)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[usuariosdto.UsuarioResponse]{
		Dados: usuariosdto.NovaUsuarioResponse(item),
	})
}

func (h *UsuarioHandler) Atualizar(w http.ResponseWriter, r *http.Request) {
	id, ok := resposta.ParametroUUID(w, r, "id")
	if !ok {
		return
	}

	var requisicao usuariosdto.AtualizarUsuarioRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	if err := h.usecase.Atualizar(r.Context(), requisicao.ParaUsuario(id)); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	item, err := h.usecase.Obter(r.Context(), id)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[usuariosdto.UsuarioResponse]{
		Dados: usuariosdto.NovaUsuarioResponse(item),
	})
}

func (h *UsuarioHandler) Remover(w http.ResponseWriter, r *http.Request) {
	id, ok := resposta.ParametroUUID(w, r, "id")
	if !ok {
		return
	}

	if err := h.usecase.Remover(r.Context(), id); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverVazio(w, http.StatusNoContent)
}

func (h *UsuarioHandler) Autenticar(w http.ResponseWriter, r *http.Request) {
	var requisicao usuariosdto.AutenticarUsuarioRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	if restante, bloqueado := h.tentativas.Bloqueio(r, requisicao.Email); bloqueado {
		responderTentativasEsgotadas(w, restante)
		return
	}

	usuario, err := h.usecase.Autenticar(r.Context(), requisicao.Email, requisicao.Senha)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrValidacao) {
			h.tentativas.RegistrarFalha(r, requisicao.Email)
		}
		if errors.Is(err, domain.ErrNotFound) {
			dto.EscreverErro(w, http.StatusNotFound, "email ou senha inválidos")
			return
		}

		resposta.ResponderErro(w, err)
		return
	}

	login := time.Now().UTC()
	if err := h.usecase.AtualizarUltimoLogin(r.Context(), usuario.ID, login); err != nil {
		resposta.ResponderErro(w, err)
		return
	}
	usuario.UltimoLogin = login

	token, expiraEm, err := h.tokens.Gerar(usuario.ID, usuario.VersaoSessao)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	administrador, err := h.usecase.EhAdministrador(r.Context(), usuario)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	comercial, err := h.usecase.TemAcessoComercial(r.Context(), usuario)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	h.tentativas.RegistrarSucesso(r, requisicao.Email)

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[usuariosdto.SessaoResponse]{
		Dados: usuariosdto.SessaoResponse{
			Token:         token,
			ExpiraEm:      expiraEm,
			Administrador: administrador,
			Comercial:     comercial,
			Usuario:       usuariosdto.NovaUsuarioResponse(usuario),
		},
	})
}

func responderTentativasEsgotadas(w http.ResponseWriter, restante time.Duration) {
	minutos := int(math.Ceil(restante.Minutes()))

	w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(restante.Seconds()))))
	dto.EscreverErro(w, http.StatusTooManyRequests, fmt.Sprintf("muitas tentativas de login; tente novamente em %d minuto(s)", minutos))
}

func (h *UsuarioHandler) TrocarSenha(w http.ResponseWriter, r *http.Request) {
	id, ok := resposta.ParametroUUID(w, r, "id")
	if !ok {
		return
	}

	var requisicao usuariosdto.TrocarSenhaRequest
	if !resposta.CorpoJSON(w, r, &requisicao) {
		return
	}

	if err := h.usecase.AtualizarSenha(r.Context(), id, requisicao.NovaSenha); err != nil {
		resposta.ResponderErro(w, err)
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
	id, ok := resposta.ParametroUUID(w, r, "id")
	if !ok {
		return
	}

	if err := acao(r.Context(), id); err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	item, err := h.usecase.Obter(r.Context(), id)
	if err != nil {
		resposta.ResponderErro(w, err)
		return
	}

	dto.EscreverJSON(w, http.StatusOK, dto.Resposta[usuariosdto.UsuarioResponse]{
		Dados: usuariosdto.NovaUsuarioResponse(item),
	})
}
