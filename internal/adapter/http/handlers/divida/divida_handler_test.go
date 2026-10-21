package divida

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/middleware"
	domaindivida "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/divida"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsinautenticacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/autenticacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/divida"
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

type dividaUseCaseFalso struct {
	portsin.DividaUseCase
	criar    portsin.CriarDividaInput
	listar   portsin.ListarDividasInput
	gerar    portsin.GerarContasInput
	chamadas int
}

func dividaDeTeste() *domaindivida.Divida {
	bp := domaindivida.TaxaDe(185, "mensal")
	divida := &domaindivida.Divida{
		ID:                   uuid.New(),
		CredorTipo:           domaindivida.CredorBanco,
		Descricao:            "Capital de giro",
		Taxa:                 &bp,
		Indexador:            domaindivida.IndexadorCDI,
		SaldoInicialCentavos: 200000,
		DataSaldo:            time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
		Status:               domaindivida.StatusAtiva,
		Versao:               1,
	}
	divida.Parcelas = []*domaindivida.Parcela{
		{
			ID: uuid.New(), Numero: 1, Vencimento: time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC), AmortizacaoCentavos: 100000,
			JurosCentavos: 2000, Status: domaindivida.StatusParcelaGerada,
			ContaAmortizacao: &domaindivida.Conta{SolicitacaoID: uuid.New(), Status: domainsolicitacao.StatusPago},
			ContaJuros:       &domaindivida.Conta{SolicitacaoID: uuid.New(), Status: domainsolicitacao.StatusAprovado},
		},
		{
			ID: uuid.New(), Numero: 2, Vencimento: time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC), AmortizacaoCentavos: 100000,
			JurosCentavos: 1000, Status: domaindivida.StatusParcelaPrevista,
		},
	}

	return divida
}

func (f *dividaUseCaseFalso) visao() *portsin.Visao {
	return &portsin.Visao{Divida: dividaDeTeste(), Hoje: time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)}
}

func (f *dividaUseCaseFalso) Criar(_ context.Context, input portsin.CriarDividaInput) (*portsin.Visao, error) {
	f.criar = input
	f.chamadas++

	return f.visao(), nil
}

func (f *dividaUseCaseFalso) Obter(context.Context, uuid.UUID, uuid.UUID) (*portsin.Visao, error) {
	f.chamadas++

	return f.visao(), nil
}

func (f *dividaUseCaseFalso) Listar(_ context.Context, input portsin.ListarDividasInput) (*portsin.Lista, error) {
	f.listar = input
	f.chamadas++

	return &portsin.Lista{Dividas: []*domaindivida.Divida{dividaDeTeste()}, Hoje: time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)}, nil
}

func (f *dividaUseCaseFalso) GerarContas(_ context.Context, input portsin.GerarContasInput) (portsin.ResultadoGeracao, error) {
	f.gerar = input
	f.chamadas++

	return portsin.ResultadoGeracao{Competencia: input.Competencia}, nil
}

func TestCriarConverteDatasEParcelas(t *testing.T) {
	usecase := &dividaUseCaseFalso{}
	handler := NewDividaHandler(usecase)
	empresaID := uuid.New()

	registrador := executaProtegido(t, handler.Criar, http.MethodPost, "/api/v1/dividas", "", `{
		"empresa_id": "`+empresaID.String()+`", "fornecedor_id": "`+uuid.NewString()+`", "credor_tipo": "banco",
		"descricao": "Capital de giro", "taxa_bp": 185, "taxa_periodo": "mensal", "liberada_em": "",
		"saldo_inicial_centavos": 200000, "data_saldo": "2026-09-30",
		"categoria_amortizacao_id": "`+uuid.NewString()+`", "categoria_juros_id": "`+uuid.NewString()+`",
		"parcelas": [{"vencimento": "2026-10-10", "amortizacao_centavos": 100000, "juros_centavos": 2000}]
	}`)

	if registrador.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", registrador.Code, registrador.Body.String())
	}
	if usecase.criar.EmpresaID != empresaID || usecase.criar.DataSaldo.Format("2006-01-02") != "2026-09-30" ||
		usecase.criar.Dados.TaxaBP == nil || *usecase.criar.Dados.TaxaBP != 185 || usecase.criar.Dados.LiberadaEm != nil ||
		len(usecase.criar.Parcelas) != 1 || usecase.criar.Parcelas[0].Vencimento.Format("2006-01-02") != "2026-10-10" {
		t.Errorf("entrada do usecase = %+v", usecase.criar)
	}
	corpo := registrador.Body.String()
	for _, trecho := range []string{
		`"saldo_devedor_centavos":100000`, `"taxa_bp":185`, `"taxa_periodo":"mensal"`, `"situacao":"vencida"`,
		`"situacao":"a_vencer"`, `"conta_juros":{"solicitacao_id"`, `"conta_juros":null`, `"data_saldo":"2026-09-30"`,
	} {
		if !strings.Contains(corpo, trecho) {
			t.Errorf("corpo sem %s: %s", trecho, corpo)
		}
	}
}

func TestDataInvalidaResponde400SemChamarOUsecase(t *testing.T) {
	usecase := &dividaUseCaseFalso{}
	handler := NewDividaHandler(usecase)
	id := uuid.NewString()

	casos := []struct {
		nome    string
		handler http.HandlerFunc
		metodo  string
		caminho string
		id      string
		corpo   string
	}{
		{"data do saldo", handler.Criar, http.MethodPost, "/api/v1/dividas", "", `{"data_saldo": "30/09/2026"}`},
		{"vencimento", handler.Criar, http.MethodPost, "/api/v1/dividas", "", `{"parcelas": [{"vencimento": "amanhã"}]}`},
		{"liberação", handler.Alterar, http.MethodPut, "/api/v1/dividas/" + id, id, `{"versao": 1, "liberada_em": "2026-13-01"}`},
		{"renegociação", handler.Renegociar, http.MethodPost, "/api/v1/dividas/" + id + "/renegociar", id, `{"versao": 1, "nova": {"data_saldo": "ontem"}}`},
		{"campo desconhecido", handler.Renegociar, http.MethodPost, "/api/v1/dividas/" + id + "/renegociar", id, `{"versao": 1, "nova": {"empresa_id": "x"}}`},
		{"empresa", handler.Listar, http.MethodGet, "/api/v1/dividas?empresa=x", "", ""},
		{"empresa da geração", handler.GerarContas, http.MethodPost, "/api/v1/dividas/gerar?competencia=2026-10&empresa=x", "", ""},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			registrador := executaProtegido(t, caso.handler, caso.metodo, caso.caminho, caso.id, caso.corpo)
			apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)
		})
	}

	if usecase.chamadas != 0 {
		t.Errorf("usecase chamado %d vezes", usecase.chamadas)
	}
}

func TestListarEGerarRepassamOsFiltros(t *testing.T) {
	usecase := &dividaUseCaseFalso{}
	handler := NewDividaHandler(usecase)

	registrador := executaProtegido(t, handler.Listar, http.MethodGet, "/api/v1/dividas?credor_tipo=fisco&status=ativa&vencidas=true", "", "")
	if registrador.Code != http.StatusOK || usecase.listar.CredorTipo != "fisco" || usecase.listar.Status != "ativa" ||
		usecase.listar.Vencidas == nil || !*usecase.listar.Vencidas || usecase.listar.Quitada != nil {
		t.Errorf("lista: status %d, entrada %+v", registrador.Code, usecase.listar)
	}
	if corpo := registrador.Body.String(); strings.Contains(corpo, "parcelas") || !strings.Contains(corpo, `"saldo_devedor_centavos":100000`) ||
		!strings.Contains(corpo, `"vencido_centavos":2000`) {
		t.Errorf("lista: %s", corpo)
	}
	registrador = executaProtegido(t, handler.Listar, http.MethodGet, "/api/v1/dividas?quitada=talvez", "", "")
	apoioteste.VerificarEnvelopeDeErro(t, registrador, http.StatusBadRequest)

	registrador = executaProtegido(t, handler.GerarContas, http.MethodPost, "/api/v1/dividas/gerar?competencia=2026-11", "", "")
	if registrador.Code != http.StatusOK || usecase.gerar.Competencia != "2026-11" ||
		!strings.Contains(registrador.Body.String(), `"geradas":[]`) {
		t.Errorf("geração: status %d, corpo %s", registrador.Code, registrador.Body.String())
	}
}
