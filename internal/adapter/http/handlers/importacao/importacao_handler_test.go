package importacao

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/middleware"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsinautenticacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/autenticacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/importacao"
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

type importacaoUseCaseFalso struct {
	portsin.ImportacaoUseCase
	criar      portsin.CriarImportacaoInput
	embarcar   portsin.EmbarcarInput
	listar     portsin.ListarImportacoesInput
	comValores bool
	chamadas   int
}

func importacaoDeTeste() *domainimportacao.Importacao {
	chegada := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)

	return &domainimportacao.Importacao{
		ID:              uuid.New(),
		Numero:          7,
		Moeda:           domainfornecedor.MoedaDolar,
		Status:          domainimportacao.StatusEmbarcada,
		ValorTotal:      domainimportacao.ValorMoedaDe(450000),
		ChegadaPrevista: &chegada,
		Versao:          2,
		Itens: []*domainimportacao.Item{
			{ID: uuid.New(), ProdutoID: uuid.New(), Quantidade: 1000, Valor: domainimportacao.ValorMoedaDe(450000)},
		},
	}
}

func (f *importacaoUseCaseFalso) Criar(_ context.Context, input portsin.CriarImportacaoInput) (*portsin.Visao, error) {
	f.criar = input
	f.chamadas++

	return &portsin.Visao{Importacao: importacaoDeTeste(), ComValores: true}, nil
}

func (f *importacaoUseCaseFalso) Obter(context.Context, uuid.UUID, uuid.UUID) (*portsin.Visao, error) {
	f.chamadas++

	return &portsin.Visao{Importacao: importacaoDeTeste(), ComValores: f.comValores}, nil
}

func (f *importacaoUseCaseFalso) Listar(_ context.Context, input portsin.ListarImportacoesInput) (*portsin.Lista, error) {
	f.listar = input
	f.chamadas++

	return &portsin.Lista{Importacoes: []*domainimportacao.Importacao{importacaoDeTeste()}, ComValores: f.comValores}, nil
}

func (f *importacaoUseCaseFalso) Embarcar(_ context.Context, input portsin.EmbarcarInput) (*portsin.Visao, error) {
	f.embarcar = input
	f.chamadas++

	return &portsin.Visao{Importacao: importacaoDeTeste(), ComValores: true}, nil
}

func TestCriarConverteAsDatasEResponde201(t *testing.T) {
	usecase := &importacaoUseCaseFalso{}
	handler := NewImportacaoHandler(usecase)
	empresaID := uuid.New()

	registrador := executaProtegido(t, handler.Criar, http.MethodPost, "/api/v1/importacoes", "", `{
		"empresa_id": "`+empresaID.String()+`", "fornecedor_id": "`+uuid.NewString()+`", "moeda": "USD",
		"chegada_prevista": "2026-11-20", "embarque_previsto": "",
		"itens": [{"produto_id": "`+uuid.NewString()+`", "quantidade": 1000, "valor_moeda_centesimos": 450000}]
	}`)

	if registrador.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", registrador.Code, registrador.Body.String())
	}
	if usecase.criar.EmpresaID != empresaID || usecase.criar.Dados.ChegadaPrevista == nil ||
		usecase.criar.Dados.EmbarquePrevisto != nil || usecase.criar.Dados.Itens[0].ValorMoedaCentesimos != 450000 {
		t.Errorf("entrada do usecase = %+v", usecase.criar)
	}
	if corpo := registrador.Body.String(); !strings.Contains(corpo, `"chegada_prevista":"2026-11-20"`) ||
		!strings.Contains(corpo, `"valor_total_moeda_centesimos":450000`) {
		t.Errorf("corpo = %s", corpo)
	}
}

func TestDataInvalidaResponde400SemChamarOUsecase(t *testing.T) {
	usecase := &importacaoUseCaseFalso{}
	handler := NewImportacaoHandler(usecase)
	id := uuid.NewString()

	casos := []struct {
		nome    string
		handler http.HandlerFunc
		metodo  string
		caminho string
		id      string
		corpo   string
	}{
		{"criar", handler.Criar, http.MethodPost, "/api/v1/importacoes", "", `{"chegada_prevista": "20/11/2026"}`},
		{"embarcar", handler.Embarcar, http.MethodPost, "/api/v1/importacoes/" + id + "/embarcar", id, `{"versao": 1, "embarcada_em": "ontem"}`},
		{"desembaraçar", handler.Desembaracar, http.MethodPost, "/api/v1/importacoes/" + id + "/desembaracar", id, `{"versao": 1, "chegada_em": "2026-13-01"}`},
		{"listar", handler.Listar, http.MethodGet, "/api/v1/importacoes?chegada_ate=amanha", "", ""},
		{"número", handler.Listar, http.MethodGet, "/api/v1/importacoes?numero=0", "", ""},
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

func TestLeituraSemValoresOmiteOsCampos(t *testing.T) {
	usecase := &importacaoUseCaseFalso{}
	handler := NewImportacaoHandler(usecase)
	id := uuid.NewString()

	registrador := executaProtegido(t, handler.Obter, http.MethodGet, "/api/v1/importacoes/"+id, id, "")
	if registrador.Code != http.StatusOK || strings.Contains(registrador.Body.String(), "valor_") {
		t.Errorf("detalhe sem valores: status %d, corpo %s", registrador.Code, registrador.Body.String())
	}

	registrador = executaProtegido(t, handler.Listar, http.MethodGet, "/api/v1/importacoes?status=embarcada&numero=7&chegada_ate=2026-12-01", "", "")
	if registrador.Code != http.StatusOK || strings.Contains(registrador.Body.String(), "valor_") {
		t.Errorf("lista sem valores: status %d, corpo %s", registrador.Code, registrador.Body.String())
	}
	if usecase.listar.Status != "embarcada" || usecase.listar.Numero == nil || *usecase.listar.Numero != 7 ||
		usecase.listar.ChegadaAte == nil || usecase.listar.ChegadaAte.Format("2006-01-02") != "2026-12-01" {
		t.Errorf("entrada da lista = %+v", usecase.listar)
	}

	usecase.comValores = true
	registrador = executaProtegido(t, handler.Obter, http.MethodGet, "/api/v1/importacoes/"+id, id, "")
	var envelope struct {
		Dados struct {
			ValorTotal *int64 `json:"valor_total_moeda_centesimos"`
			Itens      []struct {
				Valor *int64 `json:"valor_moeda_centesimos"`
			} `json:"itens"`
		} `json:"dados"`
	}
	if err := json.Unmarshal(registrador.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	if envelope.Dados.ValorTotal == nil || *envelope.Dados.ValorTotal != 450000 || envelope.Dados.Itens[0].Valor == nil {
		t.Errorf("detalhe com valores = %s", registrador.Body.String())
	}
}

func TestEmbarcarRepassaOsCampos(t *testing.T) {
	usecase := &importacaoUseCaseFalso{}
	handler := NewImportacaoHandler(usecase)
	id := uuid.New()

	registrador := executaProtegido(t, handler.Embarcar, http.MethodPost, "/api/v1/importacoes/"+id.String()+"/embarcar", id.String(),
		`{"versao": 1, "embarcada_em": "2026-10-01", "chegada_prevista": "2026-11-20", "documento_embarque": "BL-778"}`)

	if registrador.Code != http.StatusOK {
		t.Fatalf("status %d: %s", registrador.Code, registrador.Body.String())
	}
	if usecase.embarcar.ImportacaoID != id || usecase.embarcar.Versao != 1 || usecase.embarcar.EmbarcadaEm.Format("2006-01-02") != "2026-10-01" ||
		usecase.embarcar.DocumentoEmbarque == nil || *usecase.embarcar.DocumentoEmbarque != "BL-778" {
		t.Errorf("entrada do embarque = %+v", usecase.embarcar)
	}
}
