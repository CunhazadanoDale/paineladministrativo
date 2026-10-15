package solicitacao

import (
	"net/http"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

func TestEditarRepassaCorpoComVersao(t *testing.T) {
	usecase := &solicitacaoUseCaseFalso{solicitacao: novaSolicitacaoDeTeste()}
	categoriaID := uuid.New()
	id := uuid.New()

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).Editar, http.MethodPut,
		"/api/v1/solicitacoes/"+id.String(), id.String(), `{
			"valor_centavos": 25000,
			"prazo_pagamento": "2030-04-15",
			"observacao": "conta de energia",
			"forma_pagamento": "debito_automatico",
			"categoria_financeira_id": "`+categoriaID.String()+`",
			"competencia": "2030-03",
			"estimado": true,
			"versao": 4
		}`)

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	entrada := usecase.inputEditar
	if entrada.SolicitacaoID != id || entrada.UsuarioID == uuid.Nil || entrada.Versao != 4 {
		t.Errorf("entrada = %+v, esperado id do caminho, usuário autenticado e versão 4", entrada)
	}
	if entrada.ValorCentavos != 25000 || !entrada.PrazoPagamento.Equal(time.Date(2030, time.April, 15, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("valor = %d, vencimento = %s", entrada.ValorCentavos, entrada.PrazoPagamento)
	}
	if entrada.FormaPagamento != "debito_automatico" || entrada.Competencia != "2030-03" || !entrada.Estimado {
		t.Errorf("forma = %q, competência = %q, estimado = %v", entrada.FormaPagamento, entrada.Competencia, entrada.Estimado)
	}
	if entrada.CategoriaFinanceiraID == nil || *entrada.CategoriaFinanceiraID != categoriaID || entrada.FornecedorID != nil {
		t.Errorf("classificação = %v/%v", entrada.FornecedorID, entrada.CategoriaFinanceiraID)
	}
}

func TestEditarComPrazoInvalidoNaoChegaAoUsecase(t *testing.T) {
	usecase := &solicitacaoUseCaseFalso{solicitacao: novaSolicitacaoDeTeste()}
	id := uuid.NewString()

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).Editar, http.MethodPut,
		"/api/v1/solicitacoes/"+id, id, `{"prazo_pagamento": "15/04/2030", "versao": 1}`)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
	if usecase.inputEditar.SolicitacaoID != uuid.Nil {
		t.Error("não deveria chegar ao usecase com prazo inválido")
	}
}

func TestEditarSemPermissaoResponde403(t *testing.T) {
	usecase := &solicitacaoUseCaseFalso{erro: domain.ErroPermissao("perfil sem permissão para editar esta solicitação")}
	id := uuid.NewString()

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).Editar, http.MethodPut,
		"/api/v1/solicitacoes/"+id, id, `{"prazo_pagamento": "2030-04-15", "versao": 1}`)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusForbidden)
	if apoioteste.MensagemDoEnvelope(t, resposta) != "perfil sem permissão para editar esta solicitação" {
		t.Errorf("mensagem = %q", apoioteste.MensagemDoEnvelope(t, resposta))
	}
}

func TestAdiarRepassaVencimentoMotivoEVersao(t *testing.T) {
	usecase := &solicitacaoUseCaseFalso{solicitacao: novaSolicitacaoDeTeste()}
	id := uuid.New()

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).Adiar, http.MethodPost,
		"/api/v1/solicitacoes/"+id.String()+"/adiar", id.String(),
		`{"vencimento": "2030-05-02", "motivo": "acordo com o fornecedor", "versao": 3}`)

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	entrada := usecase.inputAdiar
	if entrada.SolicitacaoID != id || entrada.UsuarioID == uuid.Nil || entrada.Versao != 3 || entrada.Motivo != "acordo com o fornecedor" {
		t.Errorf("entrada = %+v", entrada)
	}
	if !entrada.Vencimento.Equal(time.Date(2030, time.May, 2, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("vencimento = %s, esperado 2030-05-02", entrada.Vencimento)
	}
}

func TestAdiarComVencimentoAusenteOuInvalidoResponde400(t *testing.T) {
	casos := map[string]string{
		`{"motivo": "x", "versao": 1}`:                             "erro de validação: vencimento é obrigatório",
		`{"vencimento": "02/05/2030", "motivo": "x", "versao": 1}`: "erro de validação: vencimento deve estar no formato AAAA-MM-DD",
	}

	for corpo, mensagem := range casos {
		usecase := &solicitacaoUseCaseFalso{solicitacao: novaSolicitacaoDeTeste()}
		id := uuid.NewString()

		resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).Adiar, http.MethodPost,
			"/api/v1/solicitacoes/"+id+"/adiar", id, corpo)

		apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
		if obtida := apoioteste.MensagemDoEnvelope(t, resposta); obtida != mensagem {
			t.Errorf("mensagem = %q, esperado %q", obtida, mensagem)
		}
		if usecase.inputAdiar.SolicitacaoID != uuid.Nil {
			t.Errorf("%s chegou ao usecase", corpo)
		}
	}
}
