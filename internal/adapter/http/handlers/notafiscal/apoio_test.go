package notafiscal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/middleware"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsinautenticacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/autenticacao"
	portsinusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/usuarios"
	"github.com/google/uuid"
)

func executaProtegido(t *testing.T, handler http.HandlerFunc, metodo, caminho string, parametros map[string]string, corpo string) *httptest.ResponseRecorder {
	t.Helper()

	usuario := &domainusuarios.Usuario{ID: uuid.New(), Ativo: true}
	protegido := middleware.Autenticar(&tokensFalso{usuarioID: usuario.ID}, &usuariosFalso{usuario: usuario}, handler)

	requisicao := httptest.NewRequest(metodo, caminho, strings.NewReader(corpo))
	for nome, valor := range parametros {
		requisicao.SetPathValue(nome, valor)
	}

	registrador := httptest.NewRecorder()
	protegido(registrador, requisicao)

	return registrador
}

type tokensFalso struct {
	usuarioID uuid.UUID
}

func (t *tokensFalso) Gerar(uuid.UUID, int) (string, time.Time, error) {
	return "token", time.Now().UTC().Add(time.Hour), nil
}

func (t *tokensFalso) Validar(string) (uuid.UUID, int, error) {
	return t.usuarioID, 0, nil
}

var _ portsinautenticacao.TokenService = (*tokensFalso)(nil)

type usuariosFalso struct {
	portsinusuarios.UsuarioUseCase
	usuario *domainusuarios.Usuario
}

func (u *usuariosFalso) Obter(context.Context, uuid.UUID) (*domainusuarios.Usuario, error) {
	return u.usuario, nil
}
