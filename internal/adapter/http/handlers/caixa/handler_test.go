package caixa

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/middleware"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsinautenticacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/autenticacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
	portsinusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/usuarios"
	"github.com/google/uuid"
)

var _ portsinautenticacao.TokenService = (*tokensFalso)(nil)

type tokensFalso struct {
	usuarioID uuid.UUID
}

func (t *tokensFalso) Gerar(uuid.UUID, int) (string, time.Time, error) {
	return "token", time.Now().UTC().Add(time.Hour), nil
}

func (t *tokensFalso) Validar(string) (uuid.UUID, int, error) {
	return t.usuarioID, 0, nil
}

type usuariosFalso struct {
	portsinusuarios.UsuarioUseCase
	usuario *domainusuarios.Usuario
}

func (u *usuariosFalso) Obter(context.Context, uuid.UUID) (*domainusuarios.Usuario, error) {
	return u.usuario, nil
}

func executaProtegido(t *testing.T, handler http.HandlerFunc, metodo, caminho, id, corpo string) *httptest.ResponseRecorder {
	t.Helper()

	usuario := &domainusuarios.Usuario{ID: uuid.New(), Ativo: true}
	protegido := middleware.Autenticar(&tokensFalso{usuarioID: usuario.ID}, &usuariosFalso{usuario: usuario}, handler)

	return apoioteste.ExecutarHandler(protegido, metodo, caminho, id, corpo)
}

type aberturaUseCaseFalso struct {
	portsin.AberturaUseCase
	input portsin.DefinirAberturaInput
}

func (f *aberturaUseCaseFalso) Definir(_ context.Context, input portsin.DefinirAberturaInput) (*domaincaixa.Abertura, error) {
	f.input = input

	return &domaincaixa.Abertura{
		ContaBancariaID: input.ContaBancariaID,
		DataCorte:       input.DataCorte,
		SaldoCentavos:   input.SaldoCentavos,
		UsuarioID:       input.UsuarioID,
		Versao:          1,
	}, nil
}

type extratoUseCaseFalso struct {
	portsin.ExtratoUseCase
	importar    portsin.ImportarExtratoInput
	lancamentos portsin.ListarLancamentosInput
}

func (f *extratoUseCaseFalso) Importar(_ context.Context, input portsin.ImportarExtratoInput) (*portsin.RelatorioImportacao, error) {
	f.importar = input

	return &portsin.RelatorioImportacao{
		Simulacao: input.Simular,
		Importacao: &domaincaixa.ImportacaoExtrato{
			ID: uuid.New(), ContaBancariaID: uuid.New(), ArquivoID: input.ArquivoID,
			PeriodoInicio: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), PeriodoFim: time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC),
			LinhasNovas: 2,
		},
		Total: 2,
	}, nil
}

func (f *extratoUseCaseFalso) ListarLancamentos(_ context.Context, input portsin.ListarLancamentosInput) ([]*domaincaixa.LancamentoExtrato, error) {
	f.lancamentos = input

	return []*domaincaixa.LancamentoExtrato{{
		ID: uuid.New(), ContaBancariaID: input.ContaBancariaID, Data: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC),
		ValorCentavos: -1290, Descricao: "TARIFA",
	}}, nil
}

func TestDefinirAberturaLeOCorpo(t *testing.T) {
	usecase := &aberturaUseCaseFalso{}
	handler := NewAberturaHandler(usecase)
	contaID := uuid.New()

	registrador := executaProtegido(t, handler.Definir, http.MethodPut, "/api/v1/caixa/contas/x/abertura", contaID.String(),
		`{"data_corte":"2026-09-30","saldo_centavos":-2500,"versao":3}`)
	if registrador.Code != http.StatusOK || !strings.Contains(registrador.Body.String(), `"data_corte":"2026-09-30"`) {
		t.Fatalf("status %d: %s", registrador.Code, registrador.Body.String())
	}
	if usecase.input.ContaBancariaID != contaID || usecase.input.SaldoCentavos != -2500 || usecase.input.Versao == nil || *usecase.input.Versao != 3 {
		t.Errorf("input = %+v", usecase.input)
	}

	for nome, corpo := range map[string]string{
		"sem saldo":          `{"data_corte":"2026-09-30"}`,
		"data inválida":      `{"data_corte":"30/09/2026","saldo_centavos":1}`,
		"campo desconhecido": `{"data_corte":"2026-09-30","saldo_centavos":1,"saldo":1}`,
	} {
		resposta := executaProtegido(t, handler.Definir, http.MethodPut, "/api/v1/caixa/contas/x/abertura", contaID.String(), corpo)
		if resposta.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d", nome, resposta.Code)
		}
		apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
	}
}

func TestImportarExtratoRespondeCriadoOuSimulado(t *testing.T) {
	usecase := &extratoUseCaseFalso{}
	handler := NewExtratoHandler(usecase)
	arquivoID := uuid.New()

	criado := executaProtegido(t, handler.Importar, http.MethodPost, "/api/v1/extratos/importacao", "", `{"arquivo_id":"`+arquivoID.String()+`"}`)
	if criado.Code != http.StatusCreated || !strings.Contains(criado.Body.String(), `"importacao_id":"`) ||
		!strings.Contains(criado.Body.String(), `"ignoradas":[]`) {
		t.Fatalf("status %d: %s", criado.Code, criado.Body.String())
	}
	if usecase.importar.ArquivoID != arquivoID || usecase.importar.Simular || usecase.importar.ContaBancariaID != nil {
		t.Errorf("input = %+v", usecase.importar)
	}

	simulado := executaProtegido(t, handler.Importar, http.MethodPost, "/api/v1/extratos/importacao", "",
		`{"arquivo_id":"`+arquivoID.String()+`","simular":true}`)
	if simulado.Code != http.StatusOK || !strings.Contains(simulado.Body.String(), `"importacao_id":null`) {
		t.Fatalf("simulação: status %d: %s", simulado.Code, simulado.Body.String())
	}
}

func TestListarLancamentosExigeContaEDatasValidas(t *testing.T) {
	usecase := &extratoUseCaseFalso{}
	handler := NewExtratoHandler(usecase)
	contaID := uuid.New()

	semConta := executaProtegido(t, handler.ListarLancamentos, http.MethodGet, "/api/v1/extratos/lancamentos", "", "")
	apoioteste.VerificarEnvelopeDeErro(t, semConta, http.StatusBadRequest)

	dataInvalida := executaProtegido(t, handler.ListarLancamentos, http.MethodGet,
		"/api/v1/extratos/lancamentos?conta="+contaID.String()+"&de=01/10/2026", "", "")
	apoioteste.VerificarEnvelopeDeErro(t, dataInvalida, http.StatusBadRequest)

	resposta := executaProtegido(t, handler.ListarLancamentos, http.MethodGet,
		"/api/v1/extratos/lancamentos?conta="+contaID.String()+"&de=2026-10-01&ate=2026-10-31", "", "")
	if resposta.Code != http.StatusOK || !strings.Contains(resposta.Body.String(), `"tipo":"debito"`) {
		t.Fatalf("status %d: %s", resposta.Code, resposta.Body.String())
	}
	if usecase.lancamentos.ContaBancariaID != contaID || usecase.lancamentos.De == nil || usecase.lancamentos.Ate == nil ||
		usecase.lancamentos.Ate.Day() != 31 {
		t.Errorf("input = %+v", usecase.lancamentos)
	}
}
