package solicitacao

import (
	"net/http"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/google/uuid"
)

func TestAprovarComVersaoNoCorpoUsaAVersao(t *testing.T) {
	usecase := &solicitacaoUseCaseFalso{solicitacao: novaSolicitacaoDeTeste()}
	id := uuid.New()

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).Aprovar, http.MethodPost,
		"/api/v1/solicitacoes/"+id.String()+"/aprovar", id.String(), `{"versao": 2}`)

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d: %s", resposta.Code, resposta.Body.String())
	}
	if usecase.aprovadoID != id || usecase.versaoAprovada != 2 {
		t.Errorf("aprovado = %s na versão %d, esperado %s na versão 2", usecase.aprovadoID, usecase.versaoAprovada, id)
	}
}

func TestAprovarSemCorpoContinuaFuncionando(t *testing.T) {
	usecase := &solicitacaoUseCaseFalso{solicitacao: novaSolicitacaoDeTeste()}
	id := uuid.New()

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).Aprovar, http.MethodPost,
		"/api/v1/solicitacoes/"+id.String()+"/aprovar", id.String(), "")

	if resposta.Code != http.StatusOK || usecase.aprovadoID != id || usecase.versaoAprovada != 0 {
		t.Errorf("status %d, aprovado %s, versão %d", resposta.Code, usecase.aprovadoID, usecase.versaoAprovada)
	}
}

func TestAprovarComCorpoInvalidoResponde400(t *testing.T) {
	usecase := &solicitacaoUseCaseFalso{solicitacao: novaSolicitacaoDeTeste()}
	id := uuid.NewString()

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).Aprovar, http.MethodPost,
		"/api/v1/solicitacoes/"+id+"/aprovar", id, `{"versao": "2"}`)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
	if usecase.aprovadoID != uuid.Nil {
		t.Error("não deveria chegar ao usecase com corpo inválido")
	}
}

func TestListarComVencidasInvalidoResponde400(t *testing.T) {
	usecase := &solicitacaoUseCaseFalso{solicitacao: novaSolicitacaoDeTeste()}

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).Listar, http.MethodGet, "/api/v1/solicitacoes?vencidas=sim", "", "")

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
	if usecase.inputListar.UsuarioID != uuid.Nil {
		t.Error("não deveria chegar ao usecase com consulta inválida")
	}
}
