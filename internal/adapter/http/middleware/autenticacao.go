package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsinautenticacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/autenticacao"
	portsinusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/usuarios"
)

type contextoUsuario string

const chaveUsuario contextoUsuario = "usuario"

func UsuarioDoContexto(ctx context.Context) (*domainusuarios.Usuario, bool) {
	usuario, ok := ctx.Value(chaveUsuario).(*domainusuarios.Usuario)

	return usuario, ok
}

func Autenticar(tokens portsinautenticacao.TokenService, usuarios portsinusuarios.UsuarioUseCase, proximo http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		usuarioID, versaoSessao, err := tokens.Validar(extrairToken(r.Header.Get("Authorization")))
		if err != nil {
			responderNaoAutenticado(w)
			return
		}

		usuario, err := usuarios.GetByID(r.Context(), usuarioID)
		if err != nil && !errors.Is(err, domain.ErrNotFound) && !errors.Is(err, domain.ErrValidacao) {
			dto.EscreverErro(w, http.StatusInternalServerError, "erro interno do servidor")
			return
		}
		if usuario == nil || !usuario.Ativo || usuario.VersaoSessao != versaoSessao {
			responderNaoAutenticado(w)
			return
		}

		proximo(w, r.WithContext(context.WithValue(r.Context(), chaveUsuario, usuario)))
	}
}

func ExigirAdministrador(usuarios portsinusuarios.UsuarioUseCase, proximo http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		usuario, ok := UsuarioDoContexto(r.Context())
		if !ok {
			responderNaoAutenticado(w)
			return
		}

		administrador, err := usuarios.EhAdministrador(r.Context(), usuario)
		if err != nil {
			dto.EscreverErro(w, http.StatusInternalServerError, "erro interno do servidor")
			return
		}
		if !administrador {
			dto.EscreverErro(w, http.StatusForbidden, "perfil sem permissão para esta operação")
			return
		}

		proximo(w, r)
	}
}

func extrairToken(authorizacao string) string {
	const prefixo = "Bearer "

	if len(authorizacao) <= len(prefixo) || !strings.EqualFold(authorizacao[:len(prefixo)], prefixo) {
		return ""
	}

	return strings.TrimSpace(authorizacao[len(prefixo):])
}

func responderNaoAutenticado(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	dto.EscreverErro(w, http.StatusUnauthorized, "token ausente ou inválido")
}
