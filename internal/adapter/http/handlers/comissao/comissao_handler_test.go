package comissao

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
	comissaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/comissao"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/middleware"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsinautenticacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/autenticacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/comissao"
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

type percentualUseCaseFalso struct {
	portsin.PercentualUseCase
	input portsin.DefinirPercentualInput
}

func (f *percentualUseCaseFalso) Definir(_ context.Context, input portsin.DefinirPercentualInput) (*domaincomissao.PercentualPedido, error) {
	f.input = input

	return &domaincomissao.PercentualPedido{
		PedidoID:        input.PedidoID,
		RepresentanteID: uuid.New(),
		Percentual:      domaincliente.PercentualDe(input.PontosBase),
		Origem:          domaincomissao.OrigemPercentualManual,
		UsuarioID:       input.UsuarioID,
		Versao:          3,
	}, nil
}

func (f *percentualUseCaseFalso) Obter(_ context.Context, pedidoID, usuarioID uuid.UUID) (*domaincomissao.PercentualPedido, error) {
	return &domaincomissao.PercentualPedido{
		PedidoID:   pedidoID,
		Percentual: domaincliente.PercentualDe(250),
		Origem:     domaincomissao.OrigemPercentualPadrao,
		UsuarioID:  usuarioID,
	}, nil
}

type comissaoUseCaseFalso struct {
	portsin.ComissaoUseCase
	inputApurar portsin.ApurarInput
	inputListar portsin.ListarComissoesInput
}

func (f *comissaoUseCaseFalso) Apurar(_ context.Context, input portsin.ApurarInput) (domaincomissao.ResultadoApuracao, error) {
	f.inputApurar = input

	return domaincomissao.ResultadoApuracao{}, nil
}

func (f *comissaoUseCaseFalso) Listar(_ context.Context, input portsin.ListarComissoesInput) ([]*domaincomissao.Comissao, error) {
	f.inputListar = input

	return []*domaincomissao.Comissao{}, nil
}

func TestDefinirPercentualExigePercentualELevaAVersao(t *testing.T) {
	usecase := &percentualUseCaseFalso{}
	handler := NewPercentualHandler(usecase)
	pedidoID := uuid.New()

	semPercentual := executaProtegido(t, handler.Definir, http.MethodPut, "/api/v1/pedidos/x/comissao", pedidoID.String(), `{"versao": 2}`)
	if semPercentual.Code != http.StatusBadRequest || !strings.Contains(apoioteste.MensagemDoEnvelope(t, semPercentual), "percentual_bp é obrigatório") {
		t.Fatalf("sem percentual: %d %s", semPercentual.Code, semPercentual.Body.String())
	}

	registrador := executaProtegido(t, handler.Definir, http.MethodPut, "/api/v1/pedidos/x/comissao", pedidoID.String(), `{"percentual_bp": 320, "versao": 2}`)
	if registrador.Code != http.StatusOK {
		t.Fatalf("status %d: %s", registrador.Code, registrador.Body.String())
	}
	if usecase.input.PedidoID != pedidoID || usecase.input.PontosBase != 320 || usecase.input.Versao == nil || *usecase.input.Versao != 2 {
		t.Fatalf("input = %+v", usecase.input)
	}

	var corpo dto.Resposta[comissaodto.PercentualResponse]
	if err := json.Unmarshal(registrador.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	if !corpo.Dados.Definido || corpo.Dados.Versao == nil || *corpo.Dados.Versao != 3 || corpo.Dados.PercentualBP != 320 {
		t.Fatalf("resposta = %+v", corpo.Dados)
	}
}

func TestObterPercentualSugeridoVemSemVersao(t *testing.T) {
	handler := NewPercentualHandler(&percentualUseCaseFalso{})

	registrador := executaProtegido(t, handler.Obter, http.MethodGet, "/api/v1/pedidos/x/comissao", uuid.NewString(), "")
	if registrador.Code != http.StatusOK || !strings.Contains(registrador.Body.String(), `"definido":false`) ||
		!strings.Contains(registrador.Body.String(), `"versao":null`) {
		t.Fatalf("status %d: %s", registrador.Code, registrador.Body.String())
	}
}

func TestApurarAceitaCorpoVazio(t *testing.T) {
	usecase := &comissaoUseCaseFalso{}
	handler := NewComissaoHandler(usecase)

	registrador := executaProtegido(t, handler.Apurar, http.MethodPost, "/api/v1/comissoes/apurar", "", "")
	if registrador.Code != http.StatusOK || !strings.Contains(registrador.Body.String(), `"apuradas":[]`) {
		t.Fatalf("status %d: %s", registrador.Code, registrador.Body.String())
	}
	if usecase.inputApurar.Empresa != nil || usecase.inputApurar.Competencia != "" {
		t.Fatalf("input = %+v", usecase.inputApurar)
	}

	empresaID := uuid.New()
	corpo := `{"empresa_id": "` + empresaID.String() + `", "competencia": "2026-09"}`
	if registrador := executaProtegido(t, handler.Apurar, http.MethodPost, "/api/v1/comissoes/apurar", "", corpo); registrador.Code != http.StatusOK {
		t.Fatalf("status %d: %s", registrador.Code, registrador.Body.String())
	}
	if *usecase.inputApurar.Empresa != empresaID || usecase.inputApurar.Competencia != "2026-09" {
		t.Fatalf("input = %+v", usecase.inputApurar)
	}
}

func TestListarComissoesLeOsFiltros(t *testing.T) {
	usecase := &comissaoUseCaseFalso{}
	handler := NewComissaoHandler(usecase)
	representanteID := uuid.New()

	registrador := executaProtegido(t, handler.Listar, http.MethodGet,
		"/api/v1/comissoes?representante="+representanteID.String()+"&competencia=2026-10&pagina=2", "", "")
	if registrador.Code != http.StatusOK {
		t.Fatalf("status %d: %s", registrador.Code, registrador.Body.String())
	}
	if *usecase.inputListar.RepresentanteID != representanteID || usecase.inputListar.Competencia != "2026-10" || usecase.inputListar.Filtro.Page != 2 {
		t.Fatalf("input = %+v", usecase.inputListar)
	}

	invalido := executaProtegido(t, handler.Listar, http.MethodGet, "/api/v1/comissoes?pedido=abc", "", "")
	apoioteste.VerificarEnvelopeDeErro(t, invalido, http.StatusBadRequest)
}
