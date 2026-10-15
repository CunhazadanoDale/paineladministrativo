//go:build e2e

package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	solicitacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/solicitacao"
	"github.com/google/uuid"
)

func prepararPagamentoE2E(t *testing.T, servidor *httptest.Server, empresaID, solicitacaoID uuid.UUID, versao int) uuid.UUID {
	t.Helper()

	sufixo := fmt.Sprintf("%09d", time.Now().UnixNano()%1000000000)
	raiz := criarCategoriaFinanceiraE2E(t, servidor, map[string]any{
		"nome": "Despesas " + sufixo, "natureza": "despesa", "linha_dre": "despesa_fixa",
	}, tokenDoTeste)
	sub := criarCategoriaFinanceiraE2E(t, servidor, map[string]any{
		"nome": "Material " + sufixo, "categoria_pai_id": raiz.ID,
	}, tokenDoTeste)
	conta := criarContaE2E(t, servidor, map[string]any{
		"empresa_id": empresaID, "banco": "001", "tipo": "corrente", "agencia": "1234", "conta": sufixo, "apelido": "Conta " + sufixo,
	}, tokenDoTeste)

	resposta := envia(t, servidor, http.MethodPatch, "/api/v1/solicitacoes/"+solicitacaoID.String()+"/classificacao", map[string]any{
		"categoria_financeira_id": sub.ID,
		"competencia":             time.Now().UTC().Format("2006-01"),
		"versao":                  versao,
	})
	conferirStatus(t, resposta, http.StatusOK)

	return conta.ID
}

func TestFinanceiroEstornaPagamentoEPagaDeNovo(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	red := criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond").ID
	tokenFinanceiro := tokenFinanceiroDasEmpresas(t, servidor, "Fábio MIA", "fabio@exemplo.com", mia)
	tokenSolicitante, tokenAprovador := usuariosParaSolicitacao(t, servidor, "estorno")

	criada := criarSolicitacaoNaEmpresa(t, servidor, tokenSolicitante, mia)
	caminho := "/api/v1/solicitacoes/" + criada.ID.String()
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/aprovar", nil, tokenAprovador), http.StatusOK)

	contaDaRed := criarContaE2E(t, servidor, map[string]any{
		"empresa_id": red, "banco": "001", "tipo": "corrente", "agencia": "4321", "conta": "777", "apelido": "Red",
	}, tokenDoTeste)
	contaReservaDaMIA := criarContaE2E(t, servidor, map[string]any{
		"empresa_id": mia, "banco": "001", "tipo": "corrente", "agencia": "4321", "conta": "888", "apelido": "Reserva",
	}, tokenDoTeste)
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/pagamento", map[string]any{
		"valor_centavos": 50000, "conta_bancaria_id": contaReservaDaMIA.ID,
	}, tokenFinanceiro), http.StatusBadRequest, "erro de validação: solicitação sem categoria financeira; classifique antes de pagar")

	conta := prepararPagamentoE2E(t, servidor, mia, criada.ID, 2)

	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/pagamento", map[string]any{"valor_centavos": 50000}, tokenFinanceiro),
		http.StatusBadRequest, "erro de validação: conta bancária do pagamento é obrigatória")
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/pagamento", map[string]any{
		"valor_centavos": 50000, "conta_bancaria_id": contaDaRed.ID,
	}, tokenFinanceiro), http.StatusBadRequest, "erro de validação: conta bancária deve ser da empresa da solicitação")

	respostaPagamento := enviaComToken(t, servidor, http.MethodPost, caminho+"/pagamento", map[string]any{
		"valor_centavos": 50000, "conta_bancaria_id": conta,
	}, tokenFinanceiro)
	conferirStatus(t, respostaPagamento, http.StatusCreated)
	primeiro := decodificarEnvelope[solicitacaodto.PagamentoResponse](t, respostaPagamento).Dados
	if primeiro.ContaBancariaID == nil || *primeiro.ContaBancariaID != conta || primeiro.UsuarioID == nil || primeiro.Estornado {
		t.Errorf("pagamento = %+v", primeiro)
	}

	estorno := map[string]any{"motivo": "pago em duplicidade", "versao": 3}
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/pagamento/estorno", estorno, tokenSolicitante),
		http.StatusForbidden, "perfil sem permissão para estornar pagamento")
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/pagamento/estorno", estorno, tokenFinanceiro),
		http.StatusConflict, "solicitação alterada por outra operação, recarregue e tente novamente")

	estorno["versao"] = 4
	respostaEstorno := enviaComToken(t, servidor, http.MethodPost, caminho+"/pagamento/estorno", estorno, tokenFinanceiro)
	conferirStatus(t, respostaEstorno, http.StatusOK)
	reaberta := decodificarEnvelope[solicitacaodto.SolicitacaoResponse](t, respostaEstorno).Dados
	if reaberta.Status != "aprovado" || reaberta.Versao != 5 {
		t.Errorf("reaberta = status %q, versão %d", reaberta.Status, reaberta.Versao)
	}

	conferirMensagem(t, enviaComToken(t, servidor, http.MethodGet, caminho+"/pagamento", nil, tokenFinanceiro),
		http.StatusNotFound, "registro não encontrado: pagamento não encontrado")
	estorno["versao"] = 5
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/pagamento/estorno", estorno, tokenFinanceiro),
		http.StatusConflict, "só é possível estornar solicitação paga")

	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/pagamento", map[string]any{
		"valor_centavos": 49000, "conta_bancaria_id": conta,
	}, tokenFinanceiro), http.StatusCreated)

	respostaLista := enviaComToken(t, servidor, http.MethodGet, caminho+"/pagamentos", nil, tokenSolicitante)
	conferirStatus(t, respostaLista, http.StatusOK)
	pagamentos := decodificarEnvelope[[]solicitacaodto.PagamentoResponse](t, respostaLista).Dados
	if len(pagamentos) != 2 || pagamentos[0].ID != primeiro.ID || !pagamentos[0].Estornado || pagamentos[0].Estorno == nil {
		t.Fatalf("pagamentos = %+v", pagamentos)
	}
	if pagamentos[0].Estorno.Motivo != "pago em duplicidade" || pagamentos[1].Estornado || pagamentos[1].ValorCentavos != 49000 {
		t.Errorf("pagamentos = %+v / %+v", pagamentos[0], pagamentos[1])
	}

	respostaHistorico := enviaComToken(t, servidor, http.MethodGet, caminho+"/historico", nil, tokenFinanceiro)
	conferirStatus(t, respostaHistorico, http.StatusOK)
	historico := decodificarEnvelope[[]solicitacaodto.HistoricoResponse](t, respostaHistorico).Dados
	if len(historico) != 6 || historico[0].Evento != "paga" || historico[1].Evento != "estornada" ||
		historico[1].Descricao != "pagamento estornado: pago em duplicidade" {
		t.Errorf("histórico = %+v", historico)
	}
}
