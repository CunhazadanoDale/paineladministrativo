package financeiro

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	financeirodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/financeiro"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/financeiro"
	"github.com/google/uuid"
)

type categoriaUseCaseFalso struct {
	portsin.CategoriaFinanceiraUseCase
	categoria      *domainfinanceiro.CategoriaFinanceira
	plano          []domainfinanceiro.RamoDoPlano
	erro           error
	inputCriar     portsin.CriarCategoriaFinanceiraInput
	inputAlterar   portsin.AlterarCategoriaFinanceiraInput
	inputSituacao  portsin.MudarSituacaoCategoriaFinanceiraInput
	inputListar    portsin.ListarCategoriasFinanceirasInput
	somenteAtivas  bool
	removida       uuid.UUID
	removidaPor    uuid.UUID
	chamadas       int
	chamadasListar int
}

func (f *categoriaUseCaseFalso) Criar(_ context.Context, input portsin.CriarCategoriaFinanceiraInput) (*domainfinanceiro.CategoriaFinanceira, error) {
	f.inputCriar = input
	f.chamadas++
	return f.categoria, f.erro
}

func (f *categoriaUseCaseFalso) Obter(context.Context, uuid.UUID) (*domainfinanceiro.CategoriaFinanceira, error) {
	return f.categoria, f.erro
}

func (f *categoriaUseCaseFalso) Listar(_ context.Context, input portsin.ListarCategoriasFinanceirasInput) ([]*domainfinanceiro.CategoriaFinanceira, error) {
	f.inputListar = input
	f.chamadasListar++
	return []*domainfinanceiro.CategoriaFinanceira{f.categoria}, f.erro
}

func (f *categoriaUseCaseFalso) Plano(_ context.Context, somenteAtivas bool) ([]domainfinanceiro.RamoDoPlano, error) {
	f.somenteAtivas = somenteAtivas
	return f.plano, f.erro
}

func (f *categoriaUseCaseFalso) Alterar(_ context.Context, input portsin.AlterarCategoriaFinanceiraInput) (*domainfinanceiro.CategoriaFinanceira, error) {
	f.inputAlterar = input
	return f.categoria, f.erro
}

func (f *categoriaUseCaseFalso) Ativar(_ context.Context, input portsin.MudarSituacaoCategoriaFinanceiraInput) (*domainfinanceiro.CategoriaFinanceira, error) {
	f.inputSituacao = input
	return f.categoria, f.erro
}

func (f *categoriaUseCaseFalso) Desativar(_ context.Context, input portsin.MudarSituacaoCategoriaFinanceiraInput) (*domainfinanceiro.CategoriaFinanceira, error) {
	f.inputSituacao = input
	return f.categoria, f.erro
}

func (f *categoriaUseCaseFalso) Remover(_ context.Context, categoriaID, usuarioID uuid.UUID) error {
	f.removida = categoriaID
	f.removidaPor = usuarioID
	return f.erro
}

func raizDeTeste(t *testing.T) *domainfinanceiro.CategoriaFinanceira {
	t.Helper()

	classificacao, err := domainfinanceiro.NovaClassificacao("despesa", "despesa_fixa")
	if err != nil {
		t.Fatalf("classificação inválida: %v", err)
	}
	raiz, err := domainfinanceiro.NovaCategoriaFinanceira(domainfinanceiro.DadosCategoriaFinanceira{Nome: "Despesas", Codigo: texto("4")}, &classificacao, nil)
	if err != nil {
		t.Fatalf("raiz inválida: %v", err)
	}

	return raiz
}

func subDeTeste(t *testing.T, raiz *domainfinanceiro.CategoriaFinanceira) *domainfinanceiro.CategoriaFinanceira {
	t.Helper()

	sub, err := domainfinanceiro.NovaCategoriaFinanceira(domainfinanceiro.DadosCategoriaFinanceira{Nome: "Aluguel"}, nil, raiz)
	if err != nil {
		t.Fatalf("subcategoria inválida: %v", err)
	}

	return sub
}

func TestCriarCategoriaRaizRepassaClassificacao(t *testing.T) {
	usecase := &categoriaUseCaseFalso{categoria: raizDeTeste(t)}

	registrador, usuarioID := executaProtegido(t, NewCategoriaFinanceiraHandler(usecase).Criar, http.MethodPost, "/api/v1/categorias-financeiras", "",
		`{"nome":"Despesas","natureza":"despesa","linha_dre":"despesa_fixa","codigo":"4","ordem":2}`)

	if registrador.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", registrador.Code, registrador.Body.String())
	}
	input := usecase.inputCriar
	if input.UsuarioID != usuarioID || input.CategoriaPaiID != nil || *input.Natureza != "despesa" || *input.LinhaDRE != "despesa_fixa" {
		t.Errorf("input = %+v", input)
	}
	if input.Dados.Nome != "Despesas" || *input.Dados.Codigo != "4" || input.Dados.Ordem != 2 {
		t.Errorf("dados = %+v", input.Dados)
	}

	corpo := decodificar(t, registrador)
	if corpo["natureza"] != "despesa" || corpo["linha_dre"] != "despesa_fixa" || corpo["lancavel"] != false || corpo["categoria_pai_id"] != nil {
		t.Errorf("resposta = %v", corpo)
	}
	if corpo["codigo"] != "4" || corpo["ativa"] != true || corpo["versao"] != float64(1) {
		t.Errorf("resposta = %v", corpo)
	}
}

func TestCriarSubcategoriaRepassaPaiSemClassificacao(t *testing.T) {
	raiz := raizDeTeste(t)
	usecase := &categoriaUseCaseFalso{categoria: subDeTeste(t, raiz)}

	registrador, _ := executaProtegido(t, NewCategoriaFinanceiraHandler(usecase).Criar, http.MethodPost, "/api/v1/categorias-financeiras", "",
		`{"nome":"Aluguel","categoria_pai_id":"`+raiz.ID.String()+`"}`)

	if registrador.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", registrador.Code, registrador.Body.String())
	}
	if input := usecase.inputCriar; *input.CategoriaPaiID != raiz.ID || input.Natureza != nil || input.LinhaDRE != nil {
		t.Errorf("input = %+v", input)
	}
	if corpo := decodificar(t, registrador); corpo["lancavel"] != true || corpo["categoria_pai_id"] != raiz.ID.String() {
		t.Errorf("resposta = %v", corpo)
	}
}

func TestCriarCategoriaRecusaCorpoInvalido(t *testing.T) {
	usecase := &categoriaUseCaseFalso{categoria: raizDeTeste(t)}

	for nome, corpo := range map[string]string{
		"campo desconhecido": `{"nome":"X","versao":1}`,
		"pai inválido":       `{"nome":"X","categoria_pai_id":"abc"}`,
		"ordem em texto":     `{"nome":"X","ordem":"1"}`,
		"corpo vazio":        ``,
	} {
		t.Run(nome, func(t *testing.T) {
			registrador, _ := executaProtegido(t, NewCategoriaFinanceiraHandler(usecase).Criar, http.MethodPost, "/api/v1/categorias-financeiras", "", corpo)
			apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)
		})
	}
	if usecase.chamadas != 0 {
		t.Errorf("usecase chamado %d vezes com corpo inválido", usecase.chamadas)
	}
}

func TestCriarCategoriaTraduzErrosDoUsecase(t *testing.T) {
	casos := []struct {
		erro     error
		status   int
		mensagem string
	}{
		{domainfinanceiro.ErroNomeDeCategoriaRepetido(), http.StatusConflict, "já existe uma categoria com este nome neste nível"},
		{domainfinanceiro.ErroSubcategoriaHerdaClassificacao(), http.StatusBadRequest, "erro de validação: subcategoria herda natureza e linha da DRE da categoria pai"},
		{domain.ErroPermissao("perfil sem permissão para o financeiro"), http.StatusForbidden, "perfil sem permissão para o financeiro"},
	}

	for _, caso := range casos {
		t.Run(caso.mensagem, func(t *testing.T) {
			usecase := &categoriaUseCaseFalso{erro: caso.erro}
			registrador, _ := executaProtegido(t, NewCategoriaFinanceiraHandler(usecase).Criar, http.MethodPost, "/api/v1/categorias-financeiras", "", `{"nome":"X"}`)

			apoioteste.VerificarEnvelopeDeErro(t, registrador, caso.status)
			if mensagem := apoioteste.MensagemDoEnvelope(t, registrador); mensagem != caso.mensagem {
				t.Errorf("mensagem %q, esperada %q", mensagem, caso.mensagem)
			}
		})
	}
}

func TestListarCategoriasRepassaFiltros(t *testing.T) {
	usecase := &categoriaUseCaseFalso{categoria: raizDeTeste(t)}
	pai := uuid.New()

	registrador, _ := executaProtegido(t, NewCategoriaFinanceiraHandler(usecase).Listar, http.MethodGet,
		"/api/v1/categorias-financeiras?pai="+pai.String()+"&raiz=true&natureza=despesa&linha_dre=despesa_fixa&ativa=true&busca=+alu+&pagina=2&tamanho=10", "", "")

	if registrador.Code != http.StatusOK {
		t.Fatalf("status %d: %s", registrador.Code, registrador.Body.String())
	}
	input := usecase.inputListar
	if *input.Pai != pai || !input.SomenteRaiz || input.Natureza != "despesa" || input.LinhaDRE != "despesa_fixa" || !*input.Ativa || input.Busca != "alu" {
		t.Errorf("input = %+v", input)
	}
	if input.Filtro.Page != 2 || input.Filtro.Size != 10 {
		t.Errorf("paginação = %+v", input.Filtro)
	}

	registrador, _ = executaProtegido(t, NewCategoriaFinanceiraHandler(usecase).Listar, http.MethodGet, "/api/v1/categorias-financeiras", "", "")
	if registrador.Code != http.StatusOK || usecase.inputListar.SomenteRaiz || usecase.inputListar.Pai != nil || usecase.inputListar.Ativa != nil {
		t.Errorf("sem filtros: status %d, input %+v", registrador.Code, usecase.inputListar)
	}

	chamadas := usecase.chamadasListar
	for _, consulta := range []string{"pai=abc", "raiz=talvez", "ativa=x"} {
		registrador, _ = executaProtegido(t, NewCategoriaFinanceiraHandler(usecase).Listar, http.MethodGet, "/api/v1/categorias-financeiras?"+consulta, "", "")
		apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)
	}
	if usecase.chamadasListar != chamadas {
		t.Error("usecase não deveria ser chamado com consulta inválida")
	}
}

func TestArvoreDevolveRaizesComSubcategorias(t *testing.T) {
	raiz := raizDeTeste(t)
	sub := subDeTeste(t, raiz)
	usecase := &categoriaUseCaseFalso{plano: []domainfinanceiro.RamoDoPlano{
		{Categoria: raiz, Subcategorias: []*domainfinanceiro.CategoriaFinanceira{sub}},
	}}

	registrador, _ := executaProtegido(t, NewCategoriaFinanceiraHandler(usecase).Arvore, http.MethodGet, "/api/v1/categorias-financeiras/arvore?ativas=true", "", "")

	if registrador.Code != http.StatusOK || !usecase.somenteAtivas {
		t.Fatalf("status %d, somente ativas %v", registrador.Code, usecase.somenteAtivas)
	}
	var envelope dto.Resposta[[]financeirodto.RamoDoPlanoResponse]
	if err := json.Unmarshal(registrador.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	if len(envelope.Dados) != 1 || envelope.Dados[0].ID != raiz.ID || len(envelope.Dados[0].Subcategorias) != 1 || !envelope.Dados[0].Subcategorias[0].Lancavel {
		t.Errorf("árvore = %+v", envelope.Dados)
	}

	usecase.plano = []domainfinanceiro.RamoDoPlano{{Categoria: raiz, Subcategorias: []*domainfinanceiro.CategoriaFinanceira{}}}
	registrador, _ = executaProtegido(t, NewCategoriaFinanceiraHandler(usecase).Arvore, http.MethodGet, "/api/v1/categorias-financeiras/arvore", "", "")
	if usecase.somenteAtivas {
		t.Error("sem a consulta, a árvore deveria trazer também as inativas")
	}
	var bruto map[string][]map[string]any
	if err := json.Unmarshal(registrador.Body.Bytes(), &bruto); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	if subcategorias, ok := bruto["dados"][0]["subcategorias"].([]any); !ok || len(subcategorias) != 0 {
		t.Errorf("raiz sem filhas deveria trazer lista vazia, veio %v", bruto["dados"][0]["subcategorias"])
	}

	registrador, _ = executaProtegido(t, NewCategoriaFinanceiraHandler(usecase).Arvore, http.MethodGet, "/api/v1/categorias-financeiras/arvore?ativas=talvez", "", "")
	apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)
}

func TestObterCategoria(t *testing.T) {
	usecase := &categoriaUseCaseFalso{categoria: raizDeTeste(t)}
	id := uuid.NewString()

	registrador, _ := executaProtegido(t, NewCategoriaFinanceiraHandler(usecase).Obter, http.MethodGet, "/api/v1/categorias-financeiras/"+id, id, "")
	if registrador.Code != http.StatusOK {
		t.Fatalf("status %d", registrador.Code)
	}

	usecase.erro = domainfinanceiro.ErroCategoriaFinanceiraNaoEncontrada()
	registrador, _ = executaProtegido(t, NewCategoriaFinanceiraHandler(usecase).Obter, http.MethodGet, "/api/v1/categorias-financeiras/"+id, id, "")
	apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusNotFound)

	registrador, _ = executaProtegido(t, NewCategoriaFinanceiraHandler(usecase).Obter, http.MethodGet, "/api/v1/categorias-financeiras/abc", "abc", "")
	apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)
}

func TestAlterarCategoriaRecusaClassificacaoNoCorpo(t *testing.T) {
	usecase := &categoriaUseCaseFalso{categoria: raizDeTeste(t)}
	id := uuid.New()
	caminho := "/api/v1/categorias-financeiras/" + id.String()

	registrador, usuarioID := executaProtegido(t, NewCategoriaFinanceiraHandler(usecase).Alterar, http.MethodPut, caminho, id.String(),
		`{"nome":"Fixas","codigo":"4.1","ordem":5,"versao":2}`)
	if registrador.Code != http.StatusOK {
		t.Fatalf("status %d: %s", registrador.Code, registrador.Body.String())
	}
	input := usecase.inputAlterar
	if input.UsuarioID != usuarioID || input.CategoriaID != id || input.Versao != 2 || input.Dados.Nome != "Fixas" || *input.Dados.Codigo != "4.1" || input.Dados.Ordem != 5 {
		t.Errorf("input = %+v", input)
	}

	for _, campo := range []string{`"natureza":"receita"`, `"linha_dre":"fora_dre"`, `"categoria_pai_id":"` + uuid.NewString() + `"`} {
		registrador, _ = executaProtegido(t, NewCategoriaFinanceiraHandler(usecase).Alterar, http.MethodPut, caminho, id.String(), `{"nome":"X","versao":2,`+campo+`}`)
		apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)
	}
}

func TestDesativarCategoriaComSubcategoriasAtivas(t *testing.T) {
	usecase := &categoriaUseCaseFalso{erro: domainfinanceiro.ErroSubcategoriasAtivas()}
	id := uuid.New()

	registrador, usuarioID := executaProtegido(t, NewCategoriaFinanceiraHandler(usecase).Desativar, http.MethodPatch,
		"/api/v1/categorias-financeiras/"+id.String()+"/desativar", id.String(), `{"versao":1}`)

	apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusConflict)
	if usecase.inputSituacao != (portsin.MudarSituacaoCategoriaFinanceiraInput{UsuarioID: usuarioID, CategoriaID: id, Versao: 1}) {
		t.Errorf("input = %+v", usecase.inputSituacao)
	}

	usecase.erro = nil
	usecase.categoria = raizDeTeste(t)
	registrador, _ = executaProtegido(t, NewCategoriaFinanceiraHandler(usecase).Ativar, http.MethodPatch,
		"/api/v1/categorias-financeiras/"+id.String()+"/ativar", id.String(), `{"versao":7}`)
	if registrador.Code != http.StatusOK || usecase.inputSituacao.Versao != 7 {
		t.Errorf("ativar: status %d, input %+v", registrador.Code, usecase.inputSituacao)
	}
}

func TestRemoverCategoria(t *testing.T) {
	usecase := &categoriaUseCaseFalso{}
	id := uuid.New()

	registrador, usuarioID := executaProtegido(t, NewCategoriaFinanceiraHandler(usecase).Remover, http.MethodDelete, "/api/v1/categorias-financeiras/"+id.String(), id.String(), "")
	if registrador.Code != http.StatusNoContent || usecase.removida != id || usecase.removidaPor != usuarioID {
		t.Errorf("status %d, removida %s por %s", registrador.Code, usecase.removida, usecase.removidaPor)
	}

	usecase.erro = domain.ErroValidacao("registro em uso por outros dados e não pode ser excluído")
	registrador, _ = executaProtegido(t, NewCategoriaFinanceiraHandler(usecase).Remover, http.MethodDelete, "/api/v1/categorias-financeiras/"+id.String(), id.String(), "")
	apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)
}
