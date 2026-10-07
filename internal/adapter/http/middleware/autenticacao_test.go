package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsinautenticacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/autenticacao"
	portsinusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/usuarios"
	"github.com/google/uuid"
)

var _ portsinautenticacao.TokenService = (*tokensDeTeste)(nil)

var _ portsinusuarios.UsuarioUseCase = (*usuariosDeTeste)(nil)

type tokensDeTeste struct {
	usuarioID uuid.UUID
	erro      error
}

func (t *tokensDeTeste) Gerar(usuarioID uuid.UUID) (string, time.Time, error) {
	return "token", time.Now().UTC().Add(time.Hour), nil
}

func (t *tokensDeTeste) Validar(token string) (uuid.UUID, error) {
	if t.erro != nil {
		return uuid.Nil, t.erro
	}
	if token == "" {
		return uuid.Nil, errors.New("token vazio")
	}

	return t.usuarioID, nil
}

type usuariosDeTeste struct {
	portsinusuarios.UsuarioUseCase
	usuario       *domainusuarios.Usuario
	administrador bool
	erro          error
}

func (u *usuariosDeTeste) GetByID(_ context.Context, id uuid.UUID) (*domainusuarios.Usuario, error) {
	if u.erro != nil {
		return nil, u.erro
	}
	if u.usuario == nil || u.usuario.ID != id {
		return nil, nil
	}

	copia := *u.usuario

	return &copia, nil
}

func (u *usuariosDeTeste) EhAdministrador(_ context.Context, usuario *domainusuarios.Usuario) (bool, error) {
	if u.erro != nil {
		return false, u.erro
	}

	return u.administrador, nil
}

func novoUsuario() *domainusuarios.Usuario {
	return &domainusuarios.Usuario{
		ID:      uuid.New(),
		Nome:    "Ana Souza",
		Email:   "ana@exemplo.com",
		Ativo:   true,
		CargoID: uuid.New(),
	}
}

func rotaDeTeste(w http.ResponseWriter, r *http.Request) {
	usuario, ok := UsuarioDoContexto(r.Context())
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(usuario.Email))
}

func executar(autenticacao func(http.HandlerFunc) http.HandlerFunc, rota http.HandlerFunc, autorizacao string) *httptest.ResponseRecorder {
	requisicao := httptest.NewRequest(http.MethodGet, "/api/v1/leads", nil)
	if autorizacao != "" {
		requisicao.Header.Set("Authorization", autorizacao)
	}

	registrador := httptest.NewRecorder()
	autenticacao(rota).ServeHTTP(registrador, requisicao)

	return registrador
}

func verificarNaoAutenticado(t *testing.T, registrador *httptest.ResponseRecorder) {
	t.Helper()

	if registrador.Code != http.StatusUnauthorized {
		t.Errorf("status %d, esperado %d: %s", registrador.Code, http.StatusUnauthorized, registrador.Body.String())
	}
	if registrador.Header().Get("WWW-Authenticate") == "" {
		t.Error("resposta 401 sem cabeçalho WWW-Authenticate")
	}
	if !strings.Contains(registrador.Body.String(), `"erro"`) {
		t.Errorf("corpo %q, esperado envelope de erro", registrador.Body.String())
	}
}

func TestAutenticarSemTokenDevolveNaoAutenticado(t *testing.T) {
	tokens := &tokensDeTeste{usuarioID: uuid.New()}
	usuarios := &usuariosDeTeste{usuario: novoUsuario()}

	registrador := executar(func(rota http.HandlerFunc) http.HandlerFunc {
		return Autenticar(tokens, usuarios, rota)
	}, rotaDeTeste, "")

	verificarNaoAutenticado(t, registrador)
}

func TestAutenticarComTokenInvalidoDevolveNaoAutenticado(t *testing.T) {
	tokens := &tokensDeTeste{usuarioID: uuid.New(), erro: errors.New("token vencido")}
	usuarios := &usuariosDeTeste{usuario: novoUsuario()}

	registrador := executar(func(rota http.HandlerFunc) http.HandlerFunc {
		return Autenticar(tokens, usuarios, rota)
	}, rotaDeTeste, "Bearer qualquer")

	verificarNaoAutenticado(t, registrador)
}

func TestAutenticarComUsuarioInexistenteDevolveNaoAutenticado(t *testing.T) {
	usuario := novoUsuario()
	tokens := &tokensDeTeste{usuarioID: uuid.New()}
	usuarios := &usuariosDeTeste{usuario: usuario}

	registrador := executar(func(rota http.HandlerFunc) http.HandlerFunc {
		return Autenticar(tokens, usuarios, rota)
	}, rotaDeTeste, "Bearer qualquer")

	verificarNaoAutenticado(t, registrador)
}

func TestAutenticarComUsuarioDesativadoDevolveNaoAutenticado(t *testing.T) {
	usuario := novoUsuario()
	usuario.Ativo = false

	tokens := &tokensDeTeste{usuarioID: usuario.ID}
	usuarios := &usuariosDeTeste{usuario: usuario}

	registrador := executar(func(rota http.HandlerFunc) http.HandlerFunc {
		return Autenticar(tokens, usuarios, rota)
	}, rotaDeTeste, "Bearer qualquer")

	verificarNaoAutenticado(t, registrador)
}

func TestAutenticarComUsuarioJaExcluidoDevolveNaoAutenticado(t *testing.T) {
	tokens := &tokensDeTeste{usuarioID: uuid.New()}
	usuarios := &usuariosDeTeste{erro: domain.ErrNotFound}

	registrador := executar(func(rota http.HandlerFunc) http.HandlerFunc {
		return Autenticar(tokens, usuarios, rota)
	}, rotaDeTeste, "Bearer qualquer")

	verificarNaoAutenticado(t, registrador)
}

func TestAutenticarComErroDeRepositorioDevolveErroInterno(t *testing.T) {
	tokens := &tokensDeTeste{usuarioID: uuid.New()}
	usuarios := &usuariosDeTeste{erro: errors.New("banco fora")}

	registrador := executar(func(rota http.HandlerFunc) http.HandlerFunc {
		return Autenticar(tokens, usuarios, rota)
	}, rotaDeTeste, "Bearer qualquer")

	if registrador.Code != http.StatusInternalServerError {
		t.Errorf("status %d, esperado %d: %s", registrador.Code, http.StatusInternalServerError, registrador.Body.String())
	}
}

func TestAutenticarComTokenValidoSegueParaARota(t *testing.T) {
	usuario := novoUsuario()
	tokens := &tokensDeTeste{usuarioID: usuario.ID}
	usuarios := &usuariosDeTeste{usuario: usuario}

	for _, autorizacao := range []string{"Bearer qualquer", "bearer qualquer"} {
		registrador := executar(func(rota http.HandlerFunc) http.HandlerFunc {
			return Autenticar(tokens, usuarios, rota)
		}, rotaDeTeste, autorizacao)

		if registrador.Code != http.StatusOK {
			t.Errorf("%q: status %d, esperado %d: %s", autorizacao, registrador.Code, http.StatusOK, registrador.Body.String())
		}
		if registrador.Body.String() != usuario.Email {
			t.Errorf("%q: rota recebeu %q, esperado %q", autorizacao, registrador.Body.String(), usuario.Email)
		}
	}
}

func TestExigirAdministradorSemUsuarioNoContextoDevolveNaoAutenticado(t *testing.T) {
	usuarios := &usuariosDeTeste{}

	registrador := httptest.NewRecorder()
	ExigirAdministrador(usuarios, rotaDeTeste).ServeHTTP(registrador, httptest.NewRequest(http.MethodGet, "/api/v1/usuarios", nil))

	verificarNaoAutenticado(t, registrador)
}

func TestExigirAdministradorSemPermissaoDevolveProibido(t *testing.T) {
	usuarios := &usuariosDeTeste{usuario: novoUsuario()}

	registrador := httptest.NewRecorder()
	requisicao := httptest.NewRequest(http.MethodGet, "/api/v1/usuarios", nil)
	requisicao = requisicao.WithContext(context.WithValue(requisicao.Context(), chaveUsuario, usuarios.usuario))
	ExigirAdministrador(usuarios, rotaDeTeste).ServeHTTP(registrador, requisicao)

	if registrador.Code != http.StatusForbidden {
		t.Errorf("status %d, esperado %d: %s", registrador.Code, http.StatusForbidden, registrador.Body.String())
	}
	if !strings.Contains(registrador.Body.String(), `"erro"`) {
		t.Errorf("corpo %q, esperado envelope de erro", registrador.Body.String())
	}
}

func TestExigirAdministradorComPermissaoSegueParaARota(t *testing.T) {
	usuarios := &usuariosDeTeste{usuario: novoUsuario(), administrador: true}

	registrador := httptest.NewRecorder()
	requisicao := httptest.NewRequest(http.MethodGet, "/api/v1/usuarios", nil)
	requisicao = requisicao.WithContext(context.WithValue(requisicao.Context(), chaveUsuario, usuarios.usuario))
	ExigirAdministrador(usuarios, rotaDeTeste).ServeHTTP(registrador, requisicao)

	if registrador.Code != http.StatusOK {
		t.Errorf("status %d, esperado %d: %s", registrador.Code, http.StatusOK, registrador.Body.String())
	}
}

func TestExtrairToken(t *testing.T) {
	casos := []struct {
		autorizacao string
		esperado    string
	}{
		{"Bearer abc.def", "abc.def"},
		{"bearer abc.def", "abc.def"},
		{"BEARER  abc.def ", "abc.def"},
		{"", ""},
		{"abc.def", ""},
		{"Basic abc.def", ""},
		{"Bearer", ""},
	}

	for _, caso := range casos {
		if obtido := extrairToken(caso.autorizacao); obtido != caso.esperado {
			t.Errorf("%q extraiu %q, esperado %q", caso.autorizacao, obtido, caso.esperado)
		}
	}
}
