package receber

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
	receberdto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/receber"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/middleware"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsinautenticacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/autenticacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/receber"
	portsinusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/usuarios"
	"github.com/google/uuid"
)

func executaProtegido(t *testing.T, handler http.HandlerFunc, metodo, caminho, id, corpo string) *httptest.ResponseRecorder {
	t.Helper()

	usuario := &domainusuarios.Usuario{ID: uuid.New(), Ativo: true}
	protegido := middleware.Autenticar(&tokensFalso{usuarioID: usuario.ID}, &usuariosFalso{usuario: usuario}, handler)

	return apoioteste.ExecutarHandler(protegido, metodo, caminho, id, corpo)
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

type tituloUseCaseFalso struct {
	portsin.TituloUseCase
	titulo       *domainreceber.Titulo
	inputCriar   portsin.CriarTituloInput
	inputListar  portsin.ListarTitulosInput
	inputBoleto  portsin.RegistrarEnvioBoletoInput
	inputProrrog portsin.ProrrogarTituloInput
}

func (f *tituloUseCaseFalso) Criar(_ context.Context, input portsin.CriarTituloInput) (*domainreceber.Titulo, error) {
	f.inputCriar = input
	return f.titulo, nil
}

func (f *tituloUseCaseFalso) Listar(_ context.Context, input portsin.ListarTitulosInput) ([]*domainreceber.Titulo, error) {
	f.inputListar = input
	return []*domainreceber.Titulo{f.titulo}, nil
}

func (f *tituloUseCaseFalso) RegistrarEnvioBoleto(_ context.Context, input portsin.RegistrarEnvioBoletoInput) (*domainreceber.Titulo, error) {
	f.inputBoleto = input
	return f.titulo, nil
}

func (f *tituloUseCaseFalso) Prorrogar(_ context.Context, input portsin.ProrrogarTituloInput) (*domainreceber.Titulo, error) {
	f.inputProrrog = input
	return f.titulo, nil
}

func tituloDeTeste() *domainreceber.Titulo {
	return &domainreceber.Titulo{
		ID:            uuid.New(),
		Origem:        domainreceber.OrigemManual,
		Numero:        "ALG-10",
		Parcela:       1,
		Vencimento:    time.Date(2020, 1, 10, 0, 0, 0, 0, time.UTC),
		ValorCentavos: 150000,
		Competencia:   domainreceber.CompetenciaDe(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)),
		Status:        domainreceber.StatusCarteira,
		Versao:        1,
	}
}

func TestCriarTituloConverteOCorpo(t *testing.T) {
	usecase := &tituloUseCaseFalso{titulo: tituloDeTeste()}
	empresa, cliente := uuid.New(), uuid.New()
	corpo := `{"empresa_id":"` + empresa.String() + `","cliente_id":"` + cliente.String() +
		`","numero":"ALG-10","parcela":1,"vencimento":"2026-11-10","valor_centavos":150000,"competencia":"2026-10"}`

	resposta := executaProtegido(t, NewTituloHandler(usecase).Criar, http.MethodPost, "/api/v1/titulos-receber", "", corpo)

	if resposta.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", resposta.Code, resposta.Body.String())
	}
	input := usecase.inputCriar
	if input.EmpresaID != empresa || input.ClienteID != cliente || input.Dados.Competencia != "2026-10" {
		t.Fatalf("input = %+v", input)
	}
	if !input.Dados.Vencimento.Equal(time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("vencimento = %s", input.Dados.Vencimento)
	}

	var corpoResposta dto.Resposta[receberdto.TituloResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &corpoResposta); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	if corpoResposta.Dados.Vencimento != "2020-01-10" || corpoResposta.Dados.Competencia != "2020-01" || !corpoResposta.Dados.Vencido {
		t.Fatalf("resposta = %+v", corpoResposta.Dados)
	}
	if corpoResposta.Dados.SaldoCentavos != 150000 {
		t.Fatalf("saldo = %d", corpoResposta.Dados.SaldoCentavos)
	}
}

func TestCriarTituloRecusaDataInvalida(t *testing.T) {
	usecase := &tituloUseCaseFalso{titulo: tituloDeTeste()}

	resposta := executaProtegido(t, NewTituloHandler(usecase).Criar, http.MethodPost, "/api/v1/titulos-receber", "", `{"vencimento":"10/11/2026"}`)

	if resposta.Code != http.StatusBadRequest || !strings.Contains(resposta.Body.String(), "AAAA-MM-DD") {
		t.Fatalf("status %d: %s", resposta.Code, resposta.Body.String())
	}
}

func TestListarTitulosRepassaOsFiltros(t *testing.T) {
	usecase := &tituloUseCaseFalso{titulo: tituloDeTeste()}
	empresa := uuid.New()

	resposta := executaProtegido(t, NewTituloHandler(usecase).Listar, http.MethodGet,
		"/api/v1/titulos-receber?empresa="+empresa.String()+"&status=em_banco&origem=pedido&vencidos=true&origem_cancelada=false&vencimento_de=2026-11-01&tamanho=5", "", "")

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d: %s", resposta.Code, resposta.Body.String())
	}
	input := usecase.inputListar
	if *input.Empresa != empresa || input.Status != "em_banco" || input.Origem != "pedido" || !input.Vencidos || input.OrigemCancelada {
		t.Fatalf("input = %+v", input)
	}
	if !input.VencimentoDe.Equal(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)) || input.Filtro.Size != 5 {
		t.Fatalf("data ou paginação = %v, %+v", input.VencimentoDe, input.Filtro)
	}

	invalido := executaProtegido(t, NewTituloHandler(usecase).Listar, http.MethodGet, "/api/v1/titulos-receber?vencidos=talvez", "", "")
	if invalido.Code != http.StatusBadRequest {
		t.Fatalf("vencidos inválido = %d", invalido.Code)
	}
}

func TestRegistrarEnvioBoletoExigeMomentoRFC3339(t *testing.T) {
	usecase := &tituloUseCaseFalso{titulo: tituloDeTeste()}
	id := uuid.New()
	handler := NewTituloHandler(usecase).RegistrarEnvioBoleto

	resposta := executaProtegido(t, handler, http.MethodPost, "/api/v1/titulos-receber/"+id.String()+"/boleto-enviado", id.String(),
		`{"versao":2,"canal":"email","enviado_em":"2026-10-15T09:30:00-03:00"}`)
	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d: %s", resposta.Code, resposta.Body.String())
	}
	if usecase.inputBoleto.TituloID != id || usecase.inputBoleto.Versao != 2 || !usecase.inputBoleto.EnviadoEm.Equal(time.Date(2026, 10, 15, 12, 30, 0, 0, time.UTC)) {
		t.Fatalf("input = %+v", usecase.inputBoleto)
	}

	invalido := executaProtegido(t, handler, http.MethodPost, "/api/v1/titulos-receber/"+id.String()+"/boleto-enviado", id.String(),
		`{"versao":2,"canal":"email","enviado_em":"15/10/2026"}`)
	if invalido.Code != http.StatusBadRequest {
		t.Fatalf("momento inválido = %d", invalido.Code)
	}
}

func TestProrrogarConverteAData(t *testing.T) {
	usecase := &tituloUseCaseFalso{titulo: tituloDeTeste()}
	id := uuid.New()

	resposta := executaProtegido(t, NewTituloHandler(usecase).Prorrogar, http.MethodPost, "/api/v1/titulos-receber/"+id.String()+"/prorrogar", id.String(),
		`{"versao":1,"vencimento":"2026-12-01","motivo":"cliente pediu"}`)

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d: %s", resposta.Code, resposta.Body.String())
	}
	if !usecase.inputProrrog.Vencimento.Equal(time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)) || usecase.inputProrrog.Motivo != "cliente pediu" {
		t.Fatalf("input = %+v", usecase.inputProrrog)
	}
}

type entradaUseCaseFalso struct {
	titulo *domainreceber.Titulo
	criado bool
	input  portsin.GerarTituloDeEntradaInput
}

func (f *entradaUseCaseFalso) GerarTituloDeEntrada(_ context.Context, input portsin.GerarTituloDeEntradaInput) (*domainreceber.Titulo, bool, error) {
	f.input = input
	return f.titulo, f.criado, nil
}

func TestGerarTituloDeEntradaResponde201SoQuandoCria(t *testing.T) {
	id, conta := uuid.New(), uuid.New()
	caminho := "/api/v1/pedidos/" + id.String() + "/titulo-entrada"
	corpo := `{"vencimento":"2026-10-20","conta_bancaria_id":"` + conta.String() + `"}`

	usecase := &entradaUseCaseFalso{titulo: tituloDeTeste(), criado: true}
	resposta := executaProtegido(t, NewEntradaDoPedidoHandler(usecase).GerarTitulo, http.MethodPost, caminho, id.String(), corpo)
	if resposta.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", resposta.Code, resposta.Body.String())
	}
	if usecase.input.PedidoID != id || *usecase.input.ContaBancariaID != conta || !usecase.input.Vencimento.Equal(time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("input = %+v", usecase.input)
	}

	usecase.criado = false
	if resposta := executaProtegido(t, NewEntradaDoPedidoHandler(usecase).GerarTitulo, http.MethodPost, caminho, id.String(), corpo); resposta.Code != http.StatusOK {
		t.Fatalf("título já existente = %d", resposta.Code)
	}
}
