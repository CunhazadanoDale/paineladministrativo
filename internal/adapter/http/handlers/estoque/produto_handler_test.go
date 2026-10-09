package estoque

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	estoquedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/estoque"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	"github.com/google/uuid"
)

type produtoUseCaseFalso struct {
	erro          error
	produto       *domainestoque.Produto
	itens         []*domainestoque.Produto
	inputCriar    portsin.CriarProdutoInput
	inputListar   portsin.ListarProdutosInput
	inputAlterar  portsin.AlterarProdutoInput
	inputAtivo    portsin.AlternarAtivoProdutoInput
	inputDestaque portsin.AlternarDestaqueProdutoInput
}

func (f *produtoUseCaseFalso) Criar(_ context.Context, input portsin.CriarProdutoInput) (uuid.UUID, error) {
	if f.erro != nil {
		return uuid.Nil, f.erro
	}
	f.inputCriar = input

	return f.produto.ID, nil
}

func (f *produtoUseCaseFalso) Obter(_ context.Context, _ uuid.UUID) (*domainestoque.Produto, error) {
	if f.erro != nil {
		return nil, f.erro
	}

	return f.produto, nil
}

func (f *produtoUseCaseFalso) Listar(_ context.Context, input portsin.ListarProdutosInput) ([]*domainestoque.Produto, error) {
	if f.erro != nil {
		return nil, f.erro
	}
	f.inputListar = input

	return f.itens, nil
}

func (f *produtoUseCaseFalso) Alterar(_ context.Context, input portsin.AlterarProdutoInput) error {
	f.inputAlterar = input

	return f.erro
}

func (f *produtoUseCaseFalso) AlternarAtivo(_ context.Context, input portsin.AlternarAtivoProdutoInput) error {
	f.inputAtivo = input

	return f.erro
}

func (f *produtoUseCaseFalso) AlternarDestaque(_ context.Context, input portsin.AlternarDestaqueProdutoInput) error {
	f.inputDestaque = input

	return f.erro
}

var _ portsin.ProdutoUseCase = (*produtoUseCaseFalso)(nil)

func TestCriarProdutoValidoRespondeCreated(t *testing.T) {
	usecase := &produtoUseCaseFalso{produto: novoProdutoDeTeste(t)}
	categoriaID := uuid.New()

	resposta := executaProtegido(t, NewProdutoHandler(usecase).Criar, http.MethodPost, "/api/v1/produtos", "",
		`{"categoria_id":"`+categoriaID.String()+`","nome":"Cimento CP II 50","unidade_medida":"un","preco_centavos":2590}`)

	if resposta.Code != http.StatusCreated {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusCreated, resposta.Body.String())
	}
	if usecase.inputCriar.CategoriaID != categoriaID {
		t.Errorf("categoria enviada = %s, esperada %s", usecase.inputCriar.CategoriaID, categoriaID)
	}
	if usecase.inputCriar.UsuarioID == uuid.Nil {
		t.Error("usecase não recebeu o usuário autenticado")
	}
	if usecase.inputCriar.PrecoCentavos == nil || *usecase.inputCriar.PrecoCentavos != 2590 {
		t.Errorf("preço enviado = %v, esperado 2590", usecase.inputCriar.PrecoCentavos)
	}
}

func TestCriarProdutoComCorpoInvalidoResponde400(t *testing.T) {
	usecase := &produtoUseCaseFalso{produto: novoProdutoDeTeste(t)}

	resposta := executaProtegido(t, NewProdutoHandler(usecase).Criar, http.MethodPost, "/api/v1/produtos", "",
		`{"categoria_id":"nao-e-uuid"}`)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
	if usecase.inputCriar.UsuarioID != uuid.Nil {
		t.Error("não deveria chegar ao usecase com corpo inválido")
	}
}

func TestCriarProdutoSemPermissaoResponde403(t *testing.T) {
	usecase := &produtoUseCaseFalso{
		erro:    domain.ErroPermissao("perfil sem permissão para gerenciar produtos"),
		produto: novoProdutoDeTeste(t),
	}

	resposta := executaProtegido(t, NewProdutoHandler(usecase).Criar, http.MethodPost, "/api/v1/produtos", "",
		`{"nome":"Cimento CP II 50","unidade_medida":"un"}`)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusForbidden)
}

func TestListarProdutosRepassaFiltros(t *testing.T) {
	categoriaID := uuid.New()
	usecase := &produtoUseCaseFalso{
		produto: novoProdutoDeTeste(t),
		itens:   []*domainestoque.Produto{novoProdutoDeTeste(t)},
	}

	caminho := "/api/v1/produtos?categoria_id=" + categoriaID.String() +
		"&busca=cimento&ativo=true&destaque=false&estoque_baixo=true&tamanho=5"
	resposta := executaProtegido(t, NewProdutoHandler(usecase).Listar, http.MethodGet, caminho, "", "")

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	if usecase.inputListar.CategoriaID == nil || *usecase.inputListar.CategoriaID != categoriaID {
		t.Errorf("categoria do filtro = %v, esperada %s", usecase.inputListar.CategoriaID, categoriaID)
	}
	if usecase.inputListar.Busca != "cimento" {
		t.Errorf("busca = %q, esperado %q", usecase.inputListar.Busca, "cimento")
	}
	if usecase.inputListar.Ativo == nil || !*usecase.inputListar.Ativo {
		t.Error("filtro ativo não repassou true")
	}
	if usecase.inputListar.Destaque == nil || *usecase.inputListar.Destaque {
		t.Error("filtro destaque não repassou false")
	}
	if !usecase.inputListar.EstoqueBaixo {
		t.Error("filtro estoque_baixo não repassou true")
	}
	if usecase.inputListar.Filtro.Size != 5 {
		t.Errorf("tamanho = %d, esperado 5", usecase.inputListar.Filtro.Size)
	}
}

func TestListarProdutosComCategoriaInvalidaResponde400(t *testing.T) {
	usecase := &produtoUseCaseFalso{produto: novoProdutoDeTeste(t)}

	resposta := executaProtegido(t, NewProdutoHandler(usecase).Listar, http.MethodGet,
		"/api/v1/produtos?categoria_id=nao-e-uuid", "", "")

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
	if usecase.inputListar.CategoriaID != nil {
		t.Error("não deveria chegar ao usecase com filtro inválido")
	}
}

func TestAlterarProdutoRepassaIdEUsuario(t *testing.T) {
	usecase := &produtoUseCaseFalso{produto: novoProdutoDeTeste(t)}
	id := uuid.New()
	categoriaID := uuid.New()

	resposta := executaProtegido(t, NewProdutoHandler(usecase).Alterar, http.MethodPut,
		"/api/v1/produtos/"+id.String(), id.String(),
		`{"categoria_id":"`+categoriaID.String()+`","nome":"Cimento CP II 40","unidade_medida":"sc"}`)

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	if usecase.inputAlterar.ProdutoID != id {
		t.Errorf("produto enviado = %s, esperado %s", usecase.inputAlterar.ProdutoID, id)
	}
	if usecase.inputAlterar.UsuarioID == uuid.Nil {
		t.Error("usecase não recebeu o usuário autenticado")
	}
	if usecase.inputAlterar.Nome != "Cimento CP II 40" {
		t.Errorf("nome enviado = %q, esperado %q", usecase.inputAlterar.Nome, "Cimento CP II 40")
	}
}

func TestAlterarProdutoNomeRepetidoResponde409(t *testing.T) {
	usecase := &produtoUseCaseFalso{
		erro:    domain.ErroConflito("já existe um produto com este nome"),
		produto: novoProdutoDeTeste(t),
	}
	id := uuid.New()

	resposta := executaProtegido(t, NewProdutoHandler(usecase).Alterar, http.MethodPut,
		"/api/v1/produtos/"+id.String(), id.String(),
		`{"nome":"Cimento CP II 50","unidade_medida":"un"}`)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusConflict)
	if apoioteste.MensagemDoEnvelope(t, resposta) != "já existe um produto com este nome" {
		t.Errorf("mensagem %q sem o detalhe do conflito", apoioteste.MensagemDoEnvelope(t, resposta))
	}
}

func TestAlternarDestaqueRepassaDestaque(t *testing.T) {
	usecase := &produtoUseCaseFalso{produto: novoProdutoDeTeste(t)}
	id := uuid.New()

	resposta := executaProtegido(t, NewProdutoHandler(usecase).AlternarDestaque, http.MethodPatch,
		"/api/v1/produtos/"+id.String(), id.String(), `{"destaque":true}`)

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	if !usecase.inputDestaque.Destaque {
		t.Error("usecase recebeu destaque false")
	}
	if usecase.inputDestaque.ProdutoID != id {
		t.Errorf("produto enviado = %s, esperado %s", usecase.inputDestaque.ProdutoID, id)
	}
}

func TestDesativarProdutoDevolveAtivoFalse(t *testing.T) {
	usecase := &produtoUseCaseFalso{produto: novoProdutoDeTeste(t)}
	id := uuid.New()

	resposta := executaProtegido(t, NewProdutoHandler(usecase).Desativar, http.MethodPatch,
		"/api/v1/produtos/"+id.String(), id.String(), "")

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	if usecase.inputAtivo.Ativo {
		t.Error("usecase recebeu ativo true no desativar")
	}

	var envelope dto.Resposta[estoquedto.ProdutoResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("corpo não é um envelope válido: %v", err)
	}
	if envelope.Dados.Saldo != 10 {
		t.Errorf("saldo devolvido = %d, esperado 10", envelope.Dados.Saldo)
	}
}

func TestSaldoDoProdutoDevolveQuantidade(t *testing.T) {
	produto := novoProdutoDeTeste(t)
	produto.Saldo = domainestoque.SaldoDe(7)
	usecase := &produtoUseCaseFalso{produto: produto}
	id := uuid.New()

	resposta := executaProtegido(t, NewProdutoHandler(usecase).Saldo, http.MethodGet,
		"/api/v1/produtos/"+id.String()+"/saldo", id.String(), "")

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}

	var envelope dto.Resposta[estoquedto.SaldoResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("corpo não é um envelope válido: %v", err)
	}
	if envelope.Dados.Saldo != 7 {
		t.Errorf("saldo devolvido = %d, esperado 7", envelope.Dados.Saldo)
	}
	if envelope.Dados.ProdutoID != produto.ID {
		t.Errorf("produto devolvido = %s, esperado %s", envelope.Dados.ProdutoID, produto.ID)
	}
}
