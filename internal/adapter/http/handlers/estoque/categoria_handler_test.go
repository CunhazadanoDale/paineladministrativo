package estoque

import (
	"context"
	"net/http"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	"github.com/google/uuid"
)

type categoriaUseCaseFalso struct {
	erro          error
	categoria     *domainestoque.Categoria
	itens         []*domainestoque.Categoria
	inputCriar    portsin.CriarCategoriaInput
	inputListar   portsin.ListarCategoriasInput
	inputAlterar  portsin.AlterarCategoriaInput
	inputAlternar portsin.AlternarAtivoCategoriaInput
}

func (f *categoriaUseCaseFalso) Criar(_ context.Context, input portsin.CriarCategoriaInput) (uuid.UUID, error) {
	if f.erro != nil {
		return uuid.Nil, f.erro
	}
	f.inputCriar = input

	return f.categoria.ID, nil
}

func (f *categoriaUseCaseFalso) Obter(_ context.Context, _ uuid.UUID) (*domainestoque.Categoria, error) {
	if f.erro != nil {
		return nil, f.erro
	}

	return f.categoria, nil
}

func (f *categoriaUseCaseFalso) Listar(_ context.Context, input portsin.ListarCategoriasInput) ([]*domainestoque.Categoria, error) {
	if f.erro != nil {
		return nil, f.erro
	}
	f.inputListar = input

	return f.itens, nil
}

func (f *categoriaUseCaseFalso) Alterar(_ context.Context, input portsin.AlterarCategoriaInput) error {
	f.inputAlterar = input

	return f.erro
}

func (f *categoriaUseCaseFalso) AlternarAtivo(_ context.Context, input portsin.AlternarAtivoCategoriaInput) error {
	f.inputAlternar = input

	return f.erro
}

var _ portsin.CategoriaUseCase = (*categoriaUseCaseFalso)(nil)

func TestCriarCategoriaValidaRespondeCreated(t *testing.T) {
	usecase := &categoriaUseCaseFalso{categoria: novaCategoriaDeTeste(t)}

	resposta := executaProtegido(t, NewCategoriaHandler(usecase).Criar, http.MethodPost,
		"/api/v1/categorias", "", `{"nome":"Alvenaria","ordem":2}`)

	if resposta.Code != http.StatusCreated {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusCreated, resposta.Body.String())
	}
	if usecase.inputCriar.Nome != "Alvenaria" {
		t.Errorf("nome enviado ao usecase = %q, esperado %q", usecase.inputCriar.Nome, "Alvenaria")
	}
	if usecase.inputCriar.Ordem != 2 {
		t.Errorf("ordem enviada ao usecase = %d, esperada 2", usecase.inputCriar.Ordem)
	}
	if usecase.inputCriar.UsuarioID == uuid.Nil {
		t.Error("usecase não recebeu o usuário autenticado")
	}
}

func TestCriarCategoriaComCorpoInvalidoResponde400(t *testing.T) {
	usecase := &categoriaUseCaseFalso{categoria: novaCategoriaDeTeste(t)}

	resposta := executaProtegido(t, NewCategoriaHandler(usecase).Criar, http.MethodPost,
		"/api/v1/categorias", "", `{"nome":`)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
	if usecase.inputCriar.UsuarioID != uuid.Nil {
		t.Error("não deveria chegar ao usecase com corpo inválido")
	}
}

func TestCriarCategoriaSemPermissaoResponde403(t *testing.T) {
	usecase := &categoriaUseCaseFalso{
		erro:      domain.ErroPermissao("perfil sem permissão para gerenciar categorias"),
		categoria: novaCategoriaDeTeste(t),
	}

	resposta := executaProtegido(t, NewCategoriaHandler(usecase).Criar, http.MethodPost,
		"/api/v1/categorias", "", `{"nome":"Alvenaria"}`)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusForbidden)
	if apoioteste.MensagemDoEnvelope(t, resposta) != "perfil sem permissão para gerenciar categorias" {
		t.Errorf("mensagem %q sem o detalhe da permissão", apoioteste.MensagemDoEnvelope(t, resposta))
	}
}

func TestListarCategoriasRepassaAtivoEPaginacao(t *testing.T) {
	usecase := &categoriaUseCaseFalso{
		categoria: novaCategoriaDeTeste(t),
		itens:     []*domainestoque.Categoria{novaCategoriaDeTeste(t)},
	}

	resposta := executaProtegido(t, NewCategoriaHandler(usecase).Listar, http.MethodGet,
		"/api/v1/categorias?ativo=false&pagina=2&tamanho=10", "", "")

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	if usecase.inputListar.Ativo == nil || *usecase.inputListar.Ativo {
		t.Error("filtro ativo não repassou o valor false")
	}
	if usecase.inputListar.Filtro.Page != 2 || usecase.inputListar.Filtro.Size != 10 {
		t.Errorf("paginação = %+v, esperada {pagina: 2, tamanho: 10}", usecase.inputListar.Filtro)
	}
}

func TestListarCategoriasSemFiltroDevolveAtivoNulo(t *testing.T) {
	usecase := &categoriaUseCaseFalso{categoria: novaCategoriaDeTeste(t)}

	resposta := executaProtegido(t, NewCategoriaHandler(usecase).Listar, http.MethodGet,
		"/api/v1/categorias", "", "")

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	if usecase.inputListar.Ativo != nil {
		t.Errorf("filtro ativo = %v, esperado ausente", *usecase.inputListar.Ativo)
	}
}

func TestListarCategoriasComAtivoInvalidoResponde400(t *testing.T) {
	usecase := &categoriaUseCaseFalso{categoria: novaCategoriaDeTeste(t)}

	resposta := executaProtegido(t, NewCategoriaHandler(usecase).Listar, http.MethodGet,
		"/api/v1/categorias?ativo=asim", "", "")

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
}

func TestAlterarCategoriaRepassaIdEUsuario(t *testing.T) {
	usecase := &categoriaUseCaseFalso{categoria: novaCategoriaDeTeste(t)}
	id := uuid.New()

	resposta := executaProtegido(t, NewCategoriaHandler(usecase).Alterar, http.MethodPut,
		"/api/v1/categorias/"+id.String(), id.String(), `{"nome":"Alvenaria nova","ordem":3,"icone":"brick"}`)

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	if usecase.inputAlterar.CategoriaID != id {
		t.Errorf("categoria enviada = %s, esperada %s", usecase.inputAlterar.CategoriaID, id)
	}
	if usecase.inputAlterar.UsuarioID == uuid.Nil {
		t.Error("usecase não recebeu o usuário autenticado")
	}
	if usecase.inputAlterar.Nome != "Alvenaria nova" {
		t.Errorf("nome enviado = %q, esperado %q", usecase.inputAlterar.Nome, "Alvenaria nova")
	}
}

func TestAlterarCategoriaInexistenteResponde404(t *testing.T) {
	usecase := &categoriaUseCaseFalso{
		erro:      domain.ErroNaoEncontrado("categoria não encontrada"),
		categoria: novaCategoriaDeTeste(t),
	}
	id := uuid.New()

	resposta := executaProtegido(t, NewCategoriaHandler(usecase).Alterar, http.MethodPut,
		"/api/v1/categorias/"+id.String(), id.String(), `{"nome":"Alvenaria"}`)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusNotFound)
}

func TestAtivarCategoriaDevolveAtivoVerdadeiro(t *testing.T) {
	usecase := &categoriaUseCaseFalso{categoria: novaCategoriaDeTeste(t)}
	id := uuid.New()

	resposta := executaProtegido(t, NewCategoriaHandler(usecase).Ativar, http.MethodPatch,
		"/api/v1/categorias/"+id.String(), id.String(), "")

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	if !usecase.inputAlternar.Ativo {
		t.Error("usecase recebeu ativo false no ativar")
	}
	if usecase.inputAlternar.CategoriaID != id {
		t.Errorf("categoria enviada = %s, esperada %s", usecase.inputAlternar.CategoriaID, id)
	}
}

func TestDesativarCategoriaComSubcategoriasResponde409(t *testing.T) {
	usecase := &categoriaUseCaseFalso{
		erro:      domain.ErroConflito("categoria possui subcategorias e não pode ser inativada"),
		categoria: novaCategoriaDeTeste(t),
	}
	id := uuid.New()

	resposta := executaProtegido(t, NewCategoriaHandler(usecase).Desativar, http.MethodPatch,
		"/api/v1/categorias/"+id.String(), id.String(), "")

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusConflict)
	if usecase.inputAlternar.Ativo {
		t.Error("usecase recebeu ativo true no desativar")
	}
}
