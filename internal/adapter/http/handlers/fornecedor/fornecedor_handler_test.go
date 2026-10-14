package fornecedor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	fornecedordto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/fornecedor"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/middleware"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsinautenticacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/autenticacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/fornecedor"
	portsinusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/usuarios"
	"github.com/google/uuid"
)

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

type fornecedorUseCaseFalso struct {
	portsin.FornecedorUseCase
	fornecedor     *domainfornecedor.Fornecedor
	erro           error
	inputCriar     portsin.CriarFornecedorInput
	inputAlterar   portsin.AlterarFornecedorInput
	inputSituacao  portsin.SituacaoFornecedorInput
	inputListar    portsin.ListarFornecedoresInput
	removido       uuid.UUID
	removidoPor    uuid.UUID
	desativacoes   int
	chamadasCriar  int
	chamadasListar int
}

func (f *fornecedorUseCaseFalso) Criar(_ context.Context, input portsin.CriarFornecedorInput) (*domainfornecedor.Fornecedor, error) {
	f.inputCriar = input
	f.chamadasCriar++
	return f.fornecedor, f.erro
}

func (f *fornecedorUseCaseFalso) Obter(_ context.Context, _ uuid.UUID) (*domainfornecedor.Fornecedor, error) {
	return f.fornecedor, f.erro
}

func (f *fornecedorUseCaseFalso) Listar(_ context.Context, input portsin.ListarFornecedoresInput) ([]*domainfornecedor.Fornecedor, error) {
	f.inputListar = input
	f.chamadasListar++
	return []*domainfornecedor.Fornecedor{f.fornecedor}, f.erro
}

func (f *fornecedorUseCaseFalso) Alterar(_ context.Context, input portsin.AlterarFornecedorInput) (*domainfornecedor.Fornecedor, error) {
	f.inputAlterar = input
	return f.fornecedor, f.erro
}

func (f *fornecedorUseCaseFalso) Desativar(_ context.Context, input portsin.SituacaoFornecedorInput) (*domainfornecedor.Fornecedor, error) {
	f.inputSituacao = input
	f.desativacoes++
	return f.fornecedor, f.erro
}

func (f *fornecedorUseCaseFalso) Remover(_ context.Context, fornecedorID, usuarioID uuid.UUID) error {
	f.removido = fornecedorID
	f.removidoPor = usuarioID
	return f.erro
}

func executaProtegido(t *testing.T, handler http.HandlerFunc, metodo, caminho, id, corpo string) (*httptest.ResponseRecorder, uuid.UUID) {
	t.Helper()

	usuario := &domainusuarios.Usuario{ID: uuid.New(), Ativo: true}
	protegido := middleware.Autenticar(&tokensFalso{usuarioID: usuario.ID}, &usuariosFalso{usuario: usuario}, handler)

	return apoioteste.ExecutarHandler(protegido, metodo, caminho, id, corpo), usuario.ID
}

func fornecedorNacional(t *testing.T) *domainfornecedor.Fornecedor {
	t.Helper()

	documento, err := domaincliente.NovoDocumento("11.222.333/0001-81")
	if err != nil {
		t.Fatalf("documento de teste inválido: %v", err)
	}
	fornecedor, err := domainfornecedor.NovoFornecedor(domainfornecedor.PaisBrasil, documento, domainfornecedor.DadosFornecedor{
		Nome:        "Cerâmica Paraíba",
		MoedaPadrao: domainfornecedor.MoedaReal,
	})
	if err != nil {
		t.Fatalf("fornecedor de teste inválido: %v", err)
	}

	return fornecedor
}

func fornecedorEstrangeiro(t *testing.T) *domainfornecedor.Fornecedor {
	t.Helper()

	identificacao := "91330200MA2"
	fornecedor, err := domainfornecedor.NovoFornecedor("CN", domaincliente.Documento{}, domainfornecedor.DadosFornecedor{
		IdentificacaoEstrangeira: &identificacao,
		Nome:                     "Foshan Ceramics",
		MoedaPadrao:              domainfornecedor.MoedaDolar,
	})
	if err != nil {
		t.Fatalf("fornecedor de teste inválido: %v", err)
	}

	return fornecedor
}

func decodificar(t *testing.T, registrador *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var envelope dto.Resposta[map[string]any]
	if err := json.Unmarshal(registrador.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("corpo não é um envelope válido: %v: %s", err, registrador.Body.String())
	}

	return envelope.Dados
}

func TestCriarFornecedorRepassaCorpoEUsuario(t *testing.T) {
	usecase := &fornecedorUseCaseFalso{fornecedor: fornecedorNacional(t)}

	registrador, usuarioID := executaProtegido(t, NewFornecedorHandler(usecase).Criar, http.MethodPost, "/api/v1/fornecedores", "",
		`{"pais":"br","documento":"11.222.333/0001-81","nome":"Cerâmica Paraíba","moeda_padrao":"brl","uf":"PB","cidade":"João Pessoa","contato_nome":"Maria","email":"vendas@exemplo.com","telefone":"83","observacoes":"terças","nome_fantasia":"Cerâmica PB"}`)

	if registrador.Code != http.StatusCreated {
		t.Fatalf("status %d, esperado %d: %s", registrador.Code, http.StatusCreated, registrador.Body.String())
	}

	input := usecase.inputCriar
	if input.UsuarioID != usuarioID || input.Pais != "br" || input.Documento == nil || *input.Documento != "11.222.333/0001-81" {
		t.Errorf("input = %+v", input)
	}
	dados := input.Dados
	if dados.Nome != "Cerâmica Paraíba" || dados.MoedaPadrao != "brl" || *dados.UF != "PB" || *dados.Cidade != "João Pessoa" {
		t.Errorf("dados = %+v", dados)
	}
	if *dados.ContatoNome != "Maria" || *dados.Email != "vendas@exemplo.com" || *dados.Telefone != "83" || dados.Observacoes != "terças" || *dados.NomeFantasia != "Cerâmica PB" {
		t.Errorf("dados de contato = %+v", dados)
	}

	corpo := decodificar(t, registrador)
	if corpo["documento"] != "11222333000181" || corpo["documento_formatado"] != "11.222.333/0001-81" || corpo["tipo_pessoa"] != "pj" {
		t.Errorf("documento na resposta = %v, %v, %v", corpo["documento"], corpo["documento_formatado"], corpo["tipo_pessoa"])
	}
	if corpo["pais"] != "BR" || corpo["moeda_padrao"] != "BRL" || corpo["versao"] != float64(1) || corpo["ativo"] != true {
		t.Errorf("resposta = %v", corpo)
	}
	if _, existe := corpo["identificacao_estrangeira"]; !existe {
		t.Error("resposta deveria trazer identificacao_estrangeira, mesmo nula")
	}
}

func TestCriarFornecedorEstrangeiroRespondeSemDocumento(t *testing.T) {
	usecase := &fornecedorUseCaseFalso{fornecedor: fornecedorEstrangeiro(t)}

	registrador, _ := executaProtegido(t, NewFornecedorHandler(usecase).Criar, http.MethodPost, "/api/v1/fornecedores", "",
		`{"pais":"CN","identificacao_estrangeira":"91330200ma2","nome":"Foshan Ceramics","moeda_padrao":"USD"}`)

	if registrador.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", registrador.Code, registrador.Body.String())
	}
	if usecase.inputCriar.Documento != nil || *usecase.inputCriar.Dados.IdentificacaoEstrangeira != "91330200ma2" {
		t.Errorf("input = %+v", usecase.inputCriar)
	}

	corpo := decodificar(t, registrador)
	if corpo["documento"] != nil || corpo["documento_formatado"] != nil || corpo["tipo_pessoa"] != nil || corpo["uf"] != nil {
		t.Errorf("estrangeiro com documento ou uf na resposta: %v", corpo)
	}
	if corpo["identificacao_estrangeira"] != "91330200MA2" || corpo["pais"] != "CN" {
		t.Errorf("resposta = %v", corpo)
	}
}

func TestCriarFornecedorRecusaCorpoInvalido(t *testing.T) {
	usecase := &fornecedorUseCaseFalso{fornecedor: fornecedorNacional(t)}

	for nome, corpo := range map[string]string{
		"campo desconhecido": `{"nome":"X","versao":1}`,
		"json quebrado":      `{"nome":`,
		"corpo vazio":        ``,
	} {
		t.Run(nome, func(t *testing.T) {
			registrador, _ := executaProtegido(t, NewFornecedorHandler(usecase).Criar, http.MethodPost, "/api/v1/fornecedores", "", corpo)
			apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)
		})
	}
	if usecase.chamadasCriar != 0 {
		t.Errorf("usecase chamado %d vezes com corpo inválido", usecase.chamadasCriar)
	}
}

func TestCriarFornecedorTraduzErrosDoUsecase(t *testing.T) {
	casos := []struct {
		erro     error
		status   int
		mensagem string
	}{
		{domain.ErroConflito("já existe um fornecedor com este documento"), http.StatusConflict, "já existe um fornecedor com este documento"},
		{domain.ErroPermissao("perfil sem permissão para gerenciar fornecedores"), http.StatusForbidden, "perfil sem permissão para gerenciar fornecedores"},
		{domain.ErroValidacao("documento é obrigatório para fornecedor do Brasil"), http.StatusBadRequest, "erro de validação: documento é obrigatório para fornecedor do Brasil"},
	}

	for _, caso := range casos {
		t.Run(caso.mensagem, func(t *testing.T) {
			usecase := &fornecedorUseCaseFalso{erro: caso.erro}
			registrador, _ := executaProtegido(t, NewFornecedorHandler(usecase).Criar, http.MethodPost, "/api/v1/fornecedores", "", `{"nome":"X"}`)

			apoioteste.VerificarEnvelopeDeErro(t, registrador, caso.status)
			if mensagem := apoioteste.MensagemDoEnvelope(t, registrador); mensagem != caso.mensagem {
				t.Errorf("mensagem %q, esperada %q", mensagem, caso.mensagem)
			}
		})
	}
}

func TestListarFornecedoresRepassaFiltrosEPaginacao(t *testing.T) {
	usecase := &fornecedorUseCaseFalso{fornecedor: fornecedorEstrangeiro(t)}

	registrador, _ := executaProtegido(t, NewFornecedorHandler(usecase).Listar, http.MethodGet,
		"/api/v1/fornecedores?busca=+foshan+&pais=cn&moeda=usd&ativo=true&pagina=2&tamanho=5", "", "")

	if registrador.Code != http.StatusOK {
		t.Fatalf("status %d: %s", registrador.Code, registrador.Body.String())
	}

	input := usecase.inputListar
	if input.Busca != "foshan" || input.Pais != "cn" || input.Moeda != "usd" || input.Ativo == nil || !*input.Ativo {
		t.Errorf("input = %+v", input)
	}
	if input.Filtro.Page != 2 || input.Filtro.Size != 5 {
		t.Errorf("paginação = %+v", input.Filtro)
	}

	var pagina dto.Paginado[fornecedordto.FornecedorResponse]
	if err := json.Unmarshal(registrador.Body.Bytes(), &pagina); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	if pagina.Pagina != 2 || pagina.Tamanho != 5 || len(pagina.Dados) != 1 || pagina.Dados[0].Nome != "Foshan Ceramics" {
		t.Errorf("página = %+v", pagina)
	}
}

func TestListarFornecedoresRecusaAtivoInvalido(t *testing.T) {
	usecase := &fornecedorUseCaseFalso{fornecedor: fornecedorNacional(t)}

	registrador, _ := executaProtegido(t, NewFornecedorHandler(usecase).Listar, http.MethodGet, "/api/v1/fornecedores?ativo=talvez", "", "")

	apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)
	if usecase.chamadasListar != 0 {
		t.Error("usecase não deveria ser chamado com filtro inválido")
	}
}

func TestObterFornecedorNaoEncontrado(t *testing.T) {
	usecase := &fornecedorUseCaseFalso{erro: domain.ErroNaoEncontrado("fornecedor não encontrado")}
	id := uuid.NewString()

	registrador, _ := executaProtegido(t, NewFornecedorHandler(usecase).Obter, http.MethodGet, "/api/v1/fornecedores/"+id, id, "")
	apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusNotFound)

	registrador, _ = executaProtegido(t, NewFornecedorHandler(usecase).Obter, http.MethodGet, "/api/v1/fornecedores/abc", "abc", "")
	apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)
}

func TestAlterarFornecedorRepassaVersaoERecusaDocumento(t *testing.T) {
	usecase := &fornecedorUseCaseFalso{fornecedor: fornecedorNacional(t)}
	id := uuid.New()

	registrador, usuarioID := executaProtegido(t, NewFornecedorHandler(usecase).Alterar, http.MethodPut, "/api/v1/fornecedores/"+id.String(), id.String(),
		`{"versao":3,"nome":"Cerâmica Nordeste","moeda_padrao":"EUR"}`)

	if registrador.Code != http.StatusOK {
		t.Fatalf("status %d: %s", registrador.Code, registrador.Body.String())
	}
	input := usecase.inputAlterar
	if input.FornecedorID != id || input.UsuarioID != usuarioID || input.Versao != 3 || input.Dados.Nome != "Cerâmica Nordeste" || input.Dados.MoedaPadrao != "EUR" {
		t.Errorf("input = %+v", input)
	}

	for _, campo := range []string{`"documento":"11222333000181"`, `"pais":"CN"`} {
		registrador, _ = executaProtegido(t, NewFornecedorHandler(usecase).Alterar, http.MethodPut, "/api/v1/fornecedores/"+id.String(), id.String(),
			`{"versao":3,"nome":"X",`+campo+`}`)
		apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)
		if mensagem := apoioteste.MensagemDoEnvelope(t, registrador); !strings.Contains(mensagem, "unknown field") {
			t.Errorf("mensagem %q deveria apontar o campo desconhecido", mensagem)
		}
	}
}

func TestAlterarFornecedorConflitoDeVersao(t *testing.T) {
	usecase := &fornecedorUseCaseFalso{erro: domainfornecedor.ErroVersaoDivergente()}
	id := uuid.NewString()

	registrador, _ := executaProtegido(t, NewFornecedorHandler(usecase).Alterar, http.MethodPut, "/api/v1/fornecedores/"+id, id, `{"versao":1,"nome":"X"}`)

	apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusConflict)
	if mensagem := apoioteste.MensagemDoEnvelope(t, registrador); mensagem != "o fornecedor foi alterado por outra pessoa; recarregue e tente de novo" {
		t.Errorf("mensagem %q", mensagem)
	}
}

func TestDesativarFornecedorExigeCorpoComVersao(t *testing.T) {
	usecase := &fornecedorUseCaseFalso{fornecedor: fornecedorNacional(t)}
	id := uuid.New()
	caminho := "/api/v1/fornecedores/" + id.String() + "/desativar"

	registrador, usuarioID := executaProtegido(t, NewFornecedorHandler(usecase).Desativar, http.MethodPatch, caminho, id.String(), `{"versao":2}`)
	if registrador.Code != http.StatusOK {
		t.Fatalf("status %d: %s", registrador.Code, registrador.Body.String())
	}
	if usecase.inputSituacao != (portsin.SituacaoFornecedorInput{UsuarioID: usuarioID, FornecedorID: id, Versao: 2}) {
		t.Errorf("input = %+v", usecase.inputSituacao)
	}

	registrador, _ = executaProtegido(t, NewFornecedorHandler(usecase).Desativar, http.MethodPatch, caminho, id.String(), ``)
	apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)
	if usecase.desativacoes != 1 {
		t.Errorf("desativações = %d, esperado 1", usecase.desativacoes)
	}
}

func TestRemoverFornecedor(t *testing.T) {
	usecase := &fornecedorUseCaseFalso{}
	id := uuid.New()

	registrador, usuarioID := executaProtegido(t, NewFornecedorHandler(usecase).Remover, http.MethodDelete, "/api/v1/fornecedores/"+id.String(), id.String(), "")

	if registrador.Code != http.StatusNoContent || registrador.Body.Len() != 0 {
		t.Errorf("status %d e corpo %q, esperado 204 vazio", registrador.Code, registrador.Body.String())
	}
	if usecase.removido != id || usecase.removidoPor != usuarioID {
		t.Errorf("removido %s por %s", usecase.removido, usecase.removidoPor)
	}

	usecase.erro = domain.ErroValidacao("registro em uso por outros dados e não pode ser excluído")
	registrador, _ = executaProtegido(t, NewFornecedorHandler(usecase).Remover, http.MethodDelete, "/api/v1/fornecedores/"+id.String(), id.String(), "")
	apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)
}
