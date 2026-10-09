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

type movimentoUseCaseFalso struct {
	erro            error
	movimento       *domainestoque.Movimento
	itens           []*domainestoque.Movimento
	inputMovimentar portsin.MovimentarEstoqueInput
	inputListar     portsin.ListarMovimentosInput
}

func (f *movimentoUseCaseFalso) Movimentar(_ context.Context, input portsin.MovimentarEstoqueInput) (*domainestoque.Movimento, error) {
	if f.erro != nil {
		return nil, f.erro
	}
	f.inputMovimentar = input

	return f.movimento, nil
}

func (f *movimentoUseCaseFalso) Listar(_ context.Context, input portsin.ListarMovimentosInput) ([]*domainestoque.Movimento, error) {
	if f.erro != nil {
		return nil, f.erro
	}
	f.inputListar = input

	return f.itens, nil
}

var _ portsin.MovimentoUseCase = (*movimentoUseCaseFalso)(nil)

func TestMovimentarRespondeCreatedERepassaInput(t *testing.T) {
	usecase := &movimentoUseCaseFalso{movimento: novoMovimentoDeTeste(t)}
	id := uuid.New()

	resposta := executaProtegido(t, NewMovimentoHandler(usecase).Movimentar, http.MethodPost,
		"/api/v1/produtos/"+id.String()+"/movimentos", id.String(),
		`{"tipo":"entrada","quantidade":10,"documento_ref":"NF-1","observacao":"reposicao"}`)

	if resposta.Code != http.StatusCreated {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusCreated, resposta.Body.String())
	}
	if usecase.inputMovimentar.ProdutoID != id {
		t.Errorf("produto enviado = %s, esperado %s", usecase.inputMovimentar.ProdutoID, id)
	}
	if usecase.inputMovimentar.UsuarioID == uuid.Nil {
		t.Error("usecase não recebeu o usuário autenticado")
	}
	if usecase.inputMovimentar.Tipo != "entrada" {
		t.Errorf("tipo enviado = %q, esperado %q", usecase.inputMovimentar.Tipo, "entrada")
	}
	if usecase.inputMovimentar.Quantidade != 10 {
		t.Errorf("quantidade enviada = %d, esperada 10", usecase.inputMovimentar.Quantidade)
	}
	if usecase.inputMovimentar.DocumentoRef == nil || *usecase.inputMovimentar.DocumentoRef != "NF-1" {
		t.Errorf("documento enviado = %v, esperado NF-1", usecase.inputMovimentar.DocumentoRef)
	}
}

func TestMovimentarComCorpoInvalidoResponde400(t *testing.T) {
	usecase := &movimentoUseCaseFalso{movimento: novoMovimentoDeTeste(t)}
	id := uuid.New()

	resposta := executaProtegido(t, NewMovimentoHandler(usecase).Movimentar, http.MethodPost,
		"/api/v1/produtos/"+id.String()+"/movimentos", id.String(), `{"quantidade":"dez"}`)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
	if usecase.inputMovimentar.ProdutoID != uuid.Nil {
		t.Error("não deveria chegar ao usecase com corpo inválido")
	}
}

func TestMovimentarComSaldoInsuficienteResponde400(t *testing.T) {
	usecase := &movimentoUseCaseFalso{
		erro:      domain.ErroValidacao("saldo do produto insuficiente para esta saída"),
		movimento: novoMovimentoDeTeste(t),
	}
	id := uuid.New()

	resposta := executaProtegido(t, NewMovimentoHandler(usecase).Movimentar, http.MethodPost,
		"/api/v1/produtos/"+id.String()+"/movimentos", id.String(),
		`{"tipo":"saida","quantidade":100}`)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
	if apoioteste.MensagemDoEnvelope(t, resposta) != "erro de validação: saldo do produto insuficiente para esta saída" {
		t.Errorf("mensagem %q sem o detalhe da validação", apoioteste.MensagemDoEnvelope(t, resposta))
	}
}

func TestMovimentarSemPermissaoResponde403(t *testing.T) {
	usecase := &movimentoUseCaseFalso{
		erro:      domain.ErroPermissao("perfil sem permissão para movimentar estoque"),
		movimento: novoMovimentoDeTeste(t),
	}
	id := uuid.New()

	resposta := executaProtegido(t, NewMovimentoHandler(usecase).Movimentar, http.MethodPost,
		"/api/v1/produtos/"+id.String()+"/movimentos", id.String(), `{"tipo":"entrada","quantidade":1}`)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusForbidden)
}

func TestListarMovimentosRepassaProdutoTipoEPaginacao(t *testing.T) {
	usecase := &movimentoUseCaseFalso{
		movimento: novoMovimentoDeTeste(t),
		itens:     []*domainestoque.Movimento{novoMovimentoDeTeste(t)},
	}
	id := uuid.New()

	resposta := executaProtegido(t, NewMovimentoHandler(usecase).Listar, http.MethodGet,
		"/api/v1/produtos/"+id.String()+"/movimentos?tipo=entrada&pagina=2&tamanho=5", id.String(), "")

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	if usecase.inputListar.ProdutoID == nil || *usecase.inputListar.ProdutoID != id {
		t.Errorf("produto do filtro = %v, esperado %s", usecase.inputListar.ProdutoID, id)
	}
	if usecase.inputListar.Tipo != "entrada" {
		t.Errorf("tipo do filtro = %q, esperado %q", usecase.inputListar.Tipo, "entrada")
	}
	if usecase.inputListar.Filtro.Page != 2 || usecase.inputListar.Filtro.Size != 5 {
		t.Errorf("paginação = %+v, esperada {pagina: 2, tamanho: 5}", usecase.inputListar.Filtro)
	}
}

func TestListarMovimentosComTipoInvalidoResponde400(t *testing.T) {
	usecase := &movimentoUseCaseFalso{
		erro:      domain.ErroValidacao("tipo de movimento deve ser entrada, saida, ajuste, perda ou devolucao"),
		movimento: novoMovimentoDeTeste(t),
	}
	id := uuid.New()

	resposta := executaProtegido(t, NewMovimentoHandler(usecase).Listar, http.MethodGet,
		"/api/v1/produtos/"+id.String()+"/movimentos?tipo=emprestimo", id.String(), "")

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
}
