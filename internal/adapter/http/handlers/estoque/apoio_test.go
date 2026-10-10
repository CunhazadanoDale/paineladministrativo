package estoque

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/middleware"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsinautenticacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/autenticacao"
	portsinusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/usuarios"
	"github.com/google/uuid"
)

func executaProtegido(t *testing.T, handler http.HandlerFunc, metodo, caminho, id, corpo string) *httptest.ResponseRecorder {
	t.Helper()

	usuario := &domainusuarios.Usuario{ID: uuid.New(), Ativo: true}
	protegido := middleware.Autenticar(&tokensFalso{usuarioID: usuario.ID}, &usuariosFalso{usuario: usuario}, handler)

	return apoioteste.ExecutarHandler(protegido, metodo, caminho, id, corpo)
}

func executaComParametros(
	t *testing.T,
	handler http.HandlerFunc,
	metodo, caminho string,
	parametros map[string]string,
	corpo string,
	autenticado bool,
) *httptest.ResponseRecorder {
	t.Helper()

	requisicao := httptest.NewRequest(metodo, caminho, strings.NewReader(corpo))
	for nome, valor := range parametros {
		requisicao.SetPathValue(nome, valor)
	}

	proximo := http.HandlerFunc(handler)
	if autenticado {
		usuario := &domainusuarios.Usuario{ID: uuid.New(), Ativo: true}
		proximo = middleware.Autenticar(&tokensFalso{usuarioID: usuario.ID}, &usuariosFalso{usuario: usuario}, proximo)
	}

	registrador := httptest.NewRecorder()
	proximo(registrador, requisicao)

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

func (u *usuariosFalso) GetByID(context.Context, uuid.UUID) (*domainusuarios.Usuario, error) {
	return u.usuario, nil
}

var _ portsinusuarios.UsuarioUseCase = (*usuariosFalso)(nil)

func novaCategoriaDeTeste(t *testing.T) *domainestoque.Categoria {
	t.Helper()

	slug, err := domainestoque.NovoSlug("Alvenaria")
	if err != nil {
		t.Fatalf("não gerei o slug da categoria: %v", err)
	}

	categoria, err := domainestoque.NovaCategoria("Alvenaria", slug, nil, 1, "brick-wall")
	if err != nil {
		t.Fatalf("não criei a categoria de teste: %v", err)
	}

	return categoria
}

func novaSubcategoriaDeTeste(t *testing.T) *domainestoque.Categoria {
	t.Helper()

	slug, err := domainestoque.NovoSlug("Cimento")
	if err != nil {
		t.Fatalf("não gerei o slug da subcategoria: %v", err)
	}

	subcategoria, err := domainestoque.NovaCategoria("Cimento", slug, novaCategoriaDeTeste(t), 1, "")
	if err != nil {
		t.Fatalf("não criei a subcategoria de teste: %v", err)
	}

	return subcategoria
}

func novoProdutoDeTeste(t *testing.T) *domainestoque.Produto {
	t.Helper()

	slug, err := domainestoque.NovoSlug("Cimento CP II 50")
	if err != nil {
		t.Fatalf("não gerei o slug do produto: %v", err)
	}

	produto, err := domainestoque.NovoProduto(domainestoque.ProdutoInput{
		Categoria:     novaSubcategoriaDeTeste(t),
		Nome:          "Cimento CP II 50",
		Slug:          slug,
		UnidadeMedida: domainestoque.UnidadeMedidaUn,
	})
	if err != nil {
		t.Fatalf("não criei o produto de teste: %v", err)
	}

	produto.Saldo = domainestoque.SaldoDe(10)

	return produto
}

func novoMovimentoDeTeste(t *testing.T) *domainestoque.Movimento {
	t.Helper()

	movimento, err := domainestoque.NovoMovimento(
		uuid.New(),
		domainestoque.TipoMovimentoEntrada,
		domainestoque.QuantidadeDe(5),
		domainestoque.SaldoDe(5),
		uuid.New(),
		nil,
		"",
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("não criei o movimento de teste: %v", err)
	}

	return movimento
}
