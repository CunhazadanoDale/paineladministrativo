package solicitacao

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
	solicitacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/solicitacao"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/middleware"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
	"github.com/google/uuid"
)

type solicitacaoUseCaseFalso struct {
	erro        error
	solicitacao *domainsolicitacao.Solicitacao
	itens       []*domainsolicitacao.Solicitacao
	pagamento   *domainsolicitacao.Pagamento

	inputCriar     portsin.CriarSolicitacaoInput
	inputListar    portsin.ListarSolicitacoesInput
	inputPagamento portsin.RegistrarPagamentoInput
	aprovadoID     uuid.UUID
	rejeitadoID    uuid.UUID
	canceladoID    uuid.UUID
	motivo         string
}

func (f *solicitacaoUseCaseFalso) Criar(_ context.Context, input portsin.CriarSolicitacaoInput) (uuid.UUID, error) {
	if f.erro != nil {
		return uuid.Nil, f.erro
	}
	f.inputCriar = input

	return f.solicitacao.ID, nil
}

func (f *solicitacaoUseCaseFalso) Obter(_ context.Context, _, _ uuid.UUID) (*domainsolicitacao.Solicitacao, error) {
	if f.erro != nil {
		return nil, f.erro
	}

	return f.solicitacao, nil
}

func (f *solicitacaoUseCaseFalso) Listar(_ context.Context, input portsin.ListarSolicitacoesInput) ([]*domainsolicitacao.Solicitacao, error) {
	if f.erro != nil {
		return nil, f.erro
	}
	f.inputListar = input

	return f.itens, nil
}

func (f *solicitacaoUseCaseFalso) Aprovar(_ context.Context, solicitacaoID, aprovadorID uuid.UUID) error {
	f.aprovadoID = solicitacaoID
	_ = aprovadorID

	return f.erro
}

func (f *solicitacaoUseCaseFalso) Rejeitar(_ context.Context, solicitacaoID, _ uuid.UUID, motivo string) error {
	f.rejeitadoID = solicitacaoID
	f.motivo = motivo

	return f.erro
}

func (f *solicitacaoUseCaseFalso) Cancelar(_ context.Context, solicitacaoID, _ uuid.UUID) error {
	f.canceladoID = solicitacaoID

	return f.erro
}

func (f *solicitacaoUseCaseFalso) RegistrarPagamento(_ context.Context, input portsin.RegistrarPagamentoInput) error {
	f.inputPagamento = input

	return f.erro
}

func (f *solicitacaoUseCaseFalso) ListarHistorico(_ context.Context, _, _ uuid.UUID) ([]*domainsolicitacao.Historico, error) {
	if f.erro != nil {
		return nil, f.erro
	}

	return []*domainsolicitacao.Historico{}, nil
}

func (f *solicitacaoUseCaseFalso) ListarArquivos(_ context.Context, _, _ uuid.UUID) ([]*domainsolicitacao.Arquivo, error) {
	if f.erro != nil {
		return nil, f.erro
	}

	return []*domainsolicitacao.Arquivo{}, nil
}

func (f *solicitacaoUseCaseFalso) ObterPagamento(_ context.Context, _, _ uuid.UUID) (*domainsolicitacao.Pagamento, error) {
	if f.erro != nil {
		return nil, f.erro
	}

	return f.pagamento, nil
}

var _ portsin.SolicitacaoUseCase = (*solicitacaoUseCaseFalso)(nil)

func novaSolicitacaoDeTeste() *domainsolicitacao.Solicitacao {
	agora := time.Now().UTC()

	return &domainsolicitacao.Solicitacao{
		ID:            uuid.New(),
		SolicitanteID: uuid.New(),
		Status:        domainsolicitacao.StatusPendenteAprovacao,
		CriadoEm:      agora,
		AtualizadoEm:  agora,
	}
}

func executaProtegido(t *testing.T, handler http.HandlerFunc, metodo, caminho, id, corpo string) *httptest.ResponseRecorder {
	t.Helper()

	usuario := &domainusuarios.Usuario{ID: uuid.New(), Ativo: true}
	protegido := middleware.Autenticar(&tokensFalso{usuarioID: usuario.ID}, &usuariosFalso{usuario: usuario}, handler)

	return apoioteste.ExecutarHandler(protegido, metodo, caminho, id, corpo)
}

func TestCriarSolicitacaoValidaRespondeCreated(t *testing.T) {
	usecase := &solicitacaoUseCaseFalso{solicitacao: novaSolicitacaoDeTeste()}

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).Criar, http.MethodPost, "/api/v1/solicitacoes", "", `{
		"valor_centavos": 150000,
		"prazo_pagamento": "2030-01-15",
		"observacao": "Compra de cimento",
		"forma_pagamento": "cartao"
	}`)

	if resposta.Code != http.StatusCreated {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusCreated, resposta.Body.String())
	}
	if usecase.inputCriar.ValorCentavos != 150000 {
		t.Errorf("valor repassado = %d, esperado 150000", usecase.inputCriar.ValorCentavos)
	}
	if usecase.inputCriar.FormaPagamento != "cartao" {
		t.Errorf("forma de pagamento = %q, esperada cartao", usecase.inputCriar.FormaPagamento)
	}
	if usecase.inputCriar.SolicitanteID == uuid.Nil {
		t.Error("solicitante não veio do usuário autenticado")
	}
}

func TestCriarSolicitacaoComCorpoInvalidoResponde400(t *testing.T) {
	usecase := &solicitacaoUseCaseFalso{solicitacao: novaSolicitacaoDeTeste()}

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).Criar, http.MethodPost, "/api/v1/solicitacoes", "", `{"valor_centavos":`)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
}

func TestCriarSolicitacaoComPrazoInvalidoResponde400(t *testing.T) {
	usecase := &solicitacaoUseCaseFalso{solicitacao: novaSolicitacaoDeTeste()}

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).Criar, http.MethodPost, "/api/v1/solicitacoes", "", `{
		"valor_centavos": 150000,
		"prazo_pagamento": "ontem",
		"forma_pagamento": "pix"
	}`)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
	if usecase.inputCriar.SolicitanteID != uuid.Nil {
		t.Error("não deveria chegar ao usecase com prazo inválido")
	}
}

func TestCriarSolicitacaoSemTokenRespondeUnauthorized(t *testing.T) {
	usecase := &solicitacaoUseCaseFalso{solicitacao: novaSolicitacaoDeTeste()}

	resposta := apoioteste.ExecutarHandler(
		NewSolicitacaoHandler(usecase).Criar, http.MethodPost, "/api/v1/solicitacoes", "", `{}`,
	)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusUnauthorized)
}

func TestListarSolicitacoesRepassaEscopoStatusEPaginacao(t *testing.T) {
	usecase := &solicitacaoUseCaseFalso{solicitacao: novaSolicitacaoDeTeste()}

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).Listar, http.MethodGet,
		"/api/v1/solicitacoes?escopo=financeiro&status=aprovado&pagina=2&tamanho=5", "", "")

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	if usecase.inputListar.Escopo != portsin.EscopoFinanceiro {
		t.Errorf("escopo = %q, esperado %q", usecase.inputListar.Escopo, portsin.EscopoFinanceiro)
	}
	if usecase.inputListar.Status != "aprovado" {
		t.Errorf("status = %q, esperado aprovado", usecase.inputListar.Status)
	}
	if usecase.inputListar.Filtro.Page != 2 || usecase.inputListar.Filtro.Size != 5 {
		t.Errorf("paginação = %+v, esperada {pagina: 2, tamanho: 5}", usecase.inputListar.Filtro)
	}

	var pagina dto.Paginado[solicitacaodto.SolicitacaoResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &pagina); err != nil {
		t.Fatalf("corpo não é uma página válida: %v", err)
	}
	if pagina.Pagina != 2 || pagina.Tamanho != 5 {
		t.Errorf("página respondida = {%d, %d}, esperada {2, 5}", pagina.Pagina, pagina.Tamanho)
	}
}

func TestObterSolicitacaoInexistenteMantemPrefixoDeNaoEncontrado(t *testing.T) {
	usecase := &solicitacaoUseCaseFalso{
		erro: domain.ErroNaoEncontrado("solicitação não encontrada"),
	}

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).Obter, http.MethodGet,
		"/api/v1/solicitacoes/"+uuid.NewString(), uuid.NewString(), "")

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusNotFound)
	if !strings.HasPrefix(apoioteste.MensagemDoEnvelope(t, resposta), "registro não encontrado: ") {
		t.Errorf("mensagem %q sem o prefixo registro não encontrado", apoioteste.MensagemDoEnvelope(t, resposta))
	}
}

func TestAprovarSemPermissaoResponde403SemPrefixo(t *testing.T) {
	usecase := &solicitacaoUseCaseFalso{
		erro: domain.ErroPermissao("perfil sem permissão para aprovar ou rejeitar solicitações"),
	}

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).Aprovar, http.MethodPost,
		"/api/v1/solicitacoes/"+uuid.NewString()+"/aprovar", uuid.NewString(), "")

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusForbidden)
	if strings.Contains(apoioteste.MensagemDoEnvelope(t, resposta), "perfil sem permissão para esta operação: ") {
		t.Errorf("mensagem %q veio com prefixo da sentinela", apoioteste.MensagemDoEnvelope(t, resposta))
	}
}

func TestRegistrarPagamentoEmConflitoResponde409SemPrefixo(t *testing.T) {
	usecase := &solicitacaoUseCaseFalso{
		erro: domain.ErroConflito("solicitação já paga"),
	}

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).RegistrarPagamento, http.MethodPost,
		"/api/v1/solicitacoes/"+uuid.NewString()+"/pagamento", uuid.NewString(), `{"valor_centavos": 100}`)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusConflict)
	if strings.Contains(apoioteste.MensagemDoEnvelope(t, resposta), "operação em conflito com o estado atual: ") {
		t.Errorf("mensagem %q veio com prefixo da sentinela", apoioteste.MensagemDoEnvelope(t, resposta))
	}
}

func TestObterPagamentoSemRegistroResponde404(t *testing.T) {
	usecase := &solicitacaoUseCaseFalso{
		erro: domain.ErroNaoEncontrado("pagamento não encontrado"),
	}

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).ObterPagamento, http.MethodGet,
		"/api/v1/solicitacoes/"+uuid.NewString()+"/pagamento", uuid.NewString(), "")

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusNotFound)
}

func TestRejeitarRepassaMotivoParaOUsecase(t *testing.T) {
	usecase := &solicitacaoUseCaseFalso{solicitacao: novaSolicitacaoDeTeste()}

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).Rejeitar, http.MethodPost,
		"/api/v1/solicitacoes/"+uuid.NewString()+"/rejeitar", uuid.NewString(), `{"motivo": "orçamento acima do teto"}`)

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	if usecase.motivo != "orçamento acima do teto" {
		t.Errorf("motivo = %q, esperado o do corpo", usecase.motivo)
	}
	if usecase.rejeitadoID == uuid.Nil {
		t.Error("id da solicitação não veio do caminho")
	}
}
