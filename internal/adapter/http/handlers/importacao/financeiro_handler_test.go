package importacao

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/middleware"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/importacao"
	"github.com/google/uuid"
)

type financeiroUseCaseFalso struct {
	portsin.FinanceiroUseCase
	custo    portsin.LancarCustoInput
	remover  portsin.ParcelaInput
	cambio   portsin.FecharCambioInput
	previsto *int64
	chamadas int
}

func (f *financeiroUseCaseFalso) visao() *portsin.Visao {
	return &portsin.Visao{Importacao: importacaoDeTeste(), ComValores: true}
}

func (f *financeiroUseCaseFalso) LancarCusto(_ context.Context, input portsin.LancarCustoInput) (*portsin.Visao, error) {
	f.custo = input
	f.chamadas++

	return f.visao(), nil
}

func (f *financeiroUseCaseFalso) RemoverParcela(_ context.Context, input portsin.ParcelaInput) (*portsin.Visao, error) {
	f.remover = input
	f.chamadas++

	return f.visao(), nil
}

func (f *financeiroUseCaseFalso) FecharCambio(_ context.Context, input portsin.FecharCambioInput) (*portsin.Visao, error) {
	f.cambio = input
	f.chamadas++

	return f.visao(), nil
}

func (f *financeiroUseCaseFalso) CustoPrevisto(_ context.Context, _, _ uuid.UUID, cambio *int64) (*domainimportacao.CalculoCusto, error) {
	f.previsto = cambio
	f.chamadas++

	return &domainimportacao.CalculoCusto{Valor: domainimportacao.ValorMoedaDe(100), ValorMercadoriaCentavos: 540}, nil
}

func executaComFilho(t *testing.T, handler http.HandlerFunc, padrao, metodo, caminho, corpo string) *httptest.ResponseRecorder {
	t.Helper()

	usuario := &domainusuarios.Usuario{ID: uuid.New(), Ativo: true}
	mux := http.NewServeMux()
	mux.Handle(padrao, middleware.Autenticar(&tokensFalso{usuarioID: usuario.ID}, &usuariosFalso{usuario: usuario}, handler))

	requisicao := httptest.NewRequest(metodo, caminho, strings.NewReader(corpo))
	requisicao.Header.Set("Authorization", "Bearer token")
	requisicao.Header.Set("Content-Type", "application/json")
	registrador := httptest.NewRecorder()
	mux.ServeHTTP(registrador, requisicao)

	return registrador
}

func TestLancarCustoCompoeOCustoPorPadrao(t *testing.T) {
	usecase := &financeiroUseCaseFalso{}
	handler := NewFinanceiroHandler(usecase)
	id := uuid.NewString()

	registrador := executaProtegido(t, handler.LancarCusto, http.MethodPost, "/api/v1/importacoes/"+id+"/custos", id,
		`{"versao": 2, "tipo": "icms", "valor_centavos": 1000, "vencimento": "2026-11-10"}`)
	if registrador.Code != http.StatusCreated || !usecase.custo.CompoeCusto || usecase.custo.Tipo != "icms" || usecase.custo.Versao != 2 {
		t.Errorf("status %d, entrada %+v", registrador.Code, usecase.custo)
	}
	if !strings.Contains(registrador.Body.String(), `"parcelas":[]`) || !strings.Contains(registrador.Body.String(), `"custos":[]`) {
		t.Errorf("detalhe do financeiro sem as listas: %s", registrador.Body.String())
	}

	executaProtegido(t, handler.LancarCusto, http.MethodPost, "/api/v1/importacoes/"+id+"/custos", id,
		`{"versao": 2, "tipo": "icms", "valor_centavos": 1000, "vencimento": "2026-11-10", "compoe_custo": false}`)
	if usecase.custo.CompoeCusto {
		t.Error("compoe_custo false foi ignorado")
	}
}

func TestRotasDaParcelaLeemOsParametros(t *testing.T) {
	usecase := &financeiroUseCaseFalso{}
	handler := NewFinanceiroHandler(usecase)
	id, parcelaID := uuid.New(), uuid.New()
	padrao := "/api/v1/importacoes/{id}/parcelas/{parcela_id}"
	caminho := "/api/v1/importacoes/" + id.String() + "/parcelas/" + parcelaID.String()

	registrador := executaComFilho(t, handler.RemoverParcela, "DELETE "+padrao, http.MethodDelete, caminho+"?versao=4", "")
	if registrador.Code != http.StatusOK || usecase.remover.ImportacaoID != id || usecase.remover.ParcelaID != parcelaID || usecase.remover.Versao != 4 {
		t.Errorf("status %d, entrada %+v", registrador.Code, usecase.remover)
	}

	registrador = executaComFilho(t, handler.FecharCambio, "POST "+padrao+"/cambio", http.MethodPost, caminho+"/cambio",
		`{"versao": 4, "cambio_micro": 5432100, "vencimento": "2026-11-01", "categoria_financeira_id": "`+uuid.NewString()+`"}`)
	if registrador.Code != http.StatusOK || usecase.cambio.CambioMicro == nil || *usecase.cambio.CambioMicro != 5432100 ||
		usecase.cambio.Vencimento.Format("2006-01-02") != "2026-11-01" {
		t.Errorf("status %d, entrada %+v", registrador.Code, usecase.cambio)
	}

	chamadas := usecase.chamadas
	apoioteste.VerificarEnvelopeDeErro(t, executaComFilho(t, handler.RemoverParcela, "DELETE "+padrao, http.MethodDelete,
		"/api/v1/importacoes/"+id.String()+"/parcelas/nao-e-uuid?versao=4", ""), http.StatusBadRequest)
	apoioteste.VerificarEnvelopeDeErro(t, executaComFilho(t, handler.RemoverParcela, "DELETE "+padrao, http.MethodDelete,
		caminho+"?versao=quatro", ""), http.StatusBadRequest)
	apoioteste.VerificarEnvelopeDeErro(t, executaComFilho(t, handler.FecharCambio, "POST "+padrao+"/cambio", http.MethodPost,
		caminho+"/cambio", `{"versao": 4, "vencimento": "01/11/2026"}`), http.StatusBadRequest)
	if usecase.chamadas != chamadas {
		t.Errorf("usecase chamado com parâmetro inválido")
	}
}

func TestCustoPrevistoLeOCambio(t *testing.T) {
	usecase := &financeiroUseCaseFalso{}
	handler := NewFinanceiroHandler(usecase)
	id := uuid.NewString()

	registrador := executaProtegido(t, handler.CustoPrevisto, http.MethodGet, "/api/v1/importacoes/"+id+"/custo-previsto?cambio_micro=5400000", id, "")
	if registrador.Code != http.StatusOK || usecase.previsto == nil || *usecase.previsto != 5400000 ||
		!strings.Contains(registrador.Body.String(), `"valor_mercadoria_centavos":540`) {
		t.Errorf("status %d, câmbio %v, corpo %s", registrador.Code, usecase.previsto, registrador.Body.String())
	}

	apoioteste.VerificarEnvelopeDeErro(t, executaProtegido(t, handler.CustoPrevisto, http.MethodGet,
		"/api/v1/importacoes/"+id+"/custo-previsto?cambio_micro=5,4", id, ""), http.StatusBadRequest)
}
