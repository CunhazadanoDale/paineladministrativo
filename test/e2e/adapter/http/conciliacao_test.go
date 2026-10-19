//go:build e2e

package http

import (
	"net/http"
	"testing"
	"time"

	caixadto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/caixa"
	solicitacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/solicitacao"
	"github.com/google/uuid"
)

func lancamentosE2E(t *testing.T, c *cenarioTitulosE2E, situacao, token string) []caixadto.LancamentoResponse {
	t.Helper()

	caminho := "/api/v1/extratos/lancamentos?conta=" + c.contaMia.String()
	if situacao != "" {
		caminho += "&situacao=" + situacao
	}
	resposta := enviaComToken(t, c.servidor, http.MethodGet, caminho, nil, token)
	conferirStatus(t, resposta, http.StatusOK)

	return decodificarPagina[caixadto.LancamentoResponse](t, resposta).Dados
}

func lancamentoComValor(t *testing.T, lancamentos []caixadto.LancamentoResponse, valor int64) caixadto.LancamentoResponse {
	t.Helper()

	for _, lancamento := range lancamentos {
		if lancamento.ValorCentavos == valor {
			return lancamento
		}
	}
	t.Fatalf("lançamento de %d não encontrado em %+v", valor, lancamentos)

	return caixadto.LancamentoResponse{}
}

func TestFluxoDeConciliacaoBancaria(t *testing.T) {
	c := novoCenarioTitulosE2E(t)
	tokenMIA := tokenFinanceiroDasEmpresas(t, c.servidor, "Fábio MIA", "fabio@exemplo.com", c.mia)
	tokenRed := tokenFinanceiroDasEmpresas(t, c.servidor, "Rita Red", "rita@exemplo.com", c.red)
	hoje := diaE2E(0).Format("2006-01-02")

	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPut, "/api/v1/caixa/contas/"+c.contaMia.String()+"/abertura", map[string]any{
		"data_corte": diaE2E(-30).Format("2006-01-02"), "saldo_centavos": 0,
	}, tokenMIA), http.StatusOK)

	titulo := c.criar(t, c.corpo(c.mia, "ALG-1"), tokenMIA)
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/titulos-receber/"+titulo.ID.String()+"/recebimentos", map[string]any{
		"versao": 1, "conta_bancaria_id": c.contaMia, "recebido_em": hoje, "valor_centavos": 150000,
	}, tokenMIA), http.StatusCreated)

	tokenSolicitante, tokenAprovador := usuariosParaSolicitacao(t, c.servidor, "conciliacao")
	solicitacao := criarSolicitacaoNaEmpresa(t, c.servidor, tokenSolicitante, c.mia)
	caminhoSolicitacao := "/api/v1/solicitacoes/" + solicitacao.ID.String()
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, caminhoSolicitacao+"/aprovar", nil, tokenAprovador), http.StatusOK)
	raiz := criarCategoriaFinanceiraE2E(t, c.servidor, map[string]any{
		"nome": "Despesas gerais", "natureza": "despesa", "linha_dre": "despesa_fixa",
	}, tokenDoTeste)
	material := criarCategoriaFinanceiraE2E(t, c.servidor, map[string]any{"nome": "Material", "categoria_pai_id": raiz.ID}, tokenDoTeste)
	conferirStatus(t, envia(t, c.servidor, http.MethodPatch, caminhoSolicitacao+"/classificacao", map[string]any{
		"categoria_financeira_id": material.ID, "competencia": time.Now().UTC().Format("2006-01"), "versao": 2,
	}), http.StatusOK)
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, caminhoSolicitacao+"/pagamento", map[string]any{
		"valor_centavos": 50000, "conta_bancaria_id": c.contaMia,
	}, tokenMIA), http.StatusCreated)

	arquivo := enviarOFX(t, c.servidor, extratoE2E("1000.00",
		transacaoE2E("REC", 0, "1500.00", "TED CONSTRUTORA"),
		transacaoE2E("PAG", 0, "-500.00", "PIX FORNECEDOR"),
		transacaoE2E("TAR", -1, "-12.90", "TARIFA"),
	), tokenMIA)
	importarExtrato(t, c.servidor, map[string]any{"arquivo_id": arquivo}, tokenMIA, http.StatusCreated)

	pendentes := lancamentosE2E(t, c, "pendente", tokenMIA)
	if len(pendentes) != 3 {
		t.Fatalf("lançamentos pendentes = %+v", pendentes)
	}
	debito := lancamentoComValor(t, pendentes, -50000)
	credito := lancamentoComValor(t, pendentes, 150000)

	resposta := enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/extratos/lancamentos/"+debito.ID.String()+"/sugestoes", nil, tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	if sugestoes := decodificarEnvelope[[]caixadto.MovimentoResponse](t, resposta).Dados; len(sugestoes) != 1 ||
		sugestoes[0].Tipo != "pagamento" || sugestoes[0].ValorCentavos != -50000 || sugestoes[0].OrigemID != solicitacao.ID {
		t.Errorf("sugestões = %+v", sugestoes)
	}
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/extratos/lancamentos/"+debito.ID.String()+"/sugestoes", nil, tokenRed),
		http.StatusForbidden)

	resposta = enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/caixa/pendencias-sistema?conta="+c.contaMia.String(), nil, tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	if pendencias := decodificarPagina[caixadto.MovimentoResponse](t, resposta).Dados; len(pendencias) != 2 {
		t.Errorf("pendências do sistema = %+v", pendencias)
	}

	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/extratos/conciliar-automatico", map[string]any{
		"conta_bancaria_id": c.contaMia, "de": diaE2E(-5).Format("2006-01-02"),
	}, tokenMIA), http.StatusBadRequest)
	resposta = enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/extratos/conciliar-automatico", map[string]any{
		"conta_bancaria_id": c.contaMia, "de": diaE2E(-5).Format("2006-01-02"), "ate": hoje,
	}, tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	if resultado := decodificarEnvelope[caixadto.ResultadoAutomaticoResponse](t, resposta).Dados; resultado.Conciliadas != 2 ||
		resultado.SemPar != 1 || resultado.Ambiguos != 0 || resultado.RestamLinhas {
		t.Errorf("conciliação automática = %+v", resultado)
	}

	conciliados := lancamentosE2E(t, c, "conciliado", tokenMIA)
	if len(conciliados) != 2 {
		t.Fatalf("lançamentos conciliados = %+v", conciliados)
	}
	debito = lancamentoComValor(t, conciliados, -50000)
	credito = lancamentoComValor(t, conciliados, 150000)
	caminhoDoDebito := "/api/v1/conciliacoes/" + debito.ConciliacaoID.String()
	resposta = enviaComToken(t, c.servidor, http.MethodGet, caminhoDoDebito, nil, tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	if conciliacao := decodificarEnvelope[caixadto.ConciliacaoResponse](t, resposta).Dados; conciliacao.Modo != "automatica" ||
		!conciliacao.Vigente || len(conciliacao.Movimentos) != 1 || conciliacao.ExtratoLancamentoID != debito.ID {
		t.Errorf("conciliação = %+v", conciliacao)
	}
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodGet, caminhoDoDebito, nil, tokenRed), http.StatusForbidden)

	resposta = enviaComToken(t, c.servidor, http.MethodGet, caminhoSolicitacao, nil, tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	versao := decodificarEnvelope[solicitacaodto.SolicitacaoResponse](t, resposta).Dados.Versao
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, caminhoSolicitacao+"/pagamento/estorno", map[string]any{
		"motivo": "pago em duplicidade", "versao": versao,
	}, tokenMIA), http.StatusOK)

	if invalidados := lancamentosE2E(t, c, "invalidado", tokenMIA); len(invalidados) != 1 || invalidados[0].ID != debito.ID {
		t.Errorf("lançamentos invalidados = %+v", invalidados)
	}
	resposta = enviaComToken(t, c.servidor, http.MethodGet, caminhoDoDebito, nil, tokenMIA)
	if conciliacao := decodificarEnvelope[caixadto.ConciliacaoResponse](t, resposta).Dados; !conciliacao.Invalidada || !conciliacao.Movimentos[0].Estornado {
		t.Errorf("conciliação invalidada = %+v", conciliacao)
	}

	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, caminhoDoDebito+"/desfazer", map[string]any{}, tokenMIA), http.StatusBadRequest)
	resposta = enviaComToken(t, c.servidor, http.MethodPost, caminhoDoDebito+"/desfazer", map[string]any{"motivo": "pagamento estornado"}, tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	if desfeita := decodificarEnvelope[caixadto.ConciliacaoResponse](t, resposta).Dados; desfeita.Vigente || desfeita.MotivoDesfazer == nil {
		t.Errorf("conciliação desfeita = %+v", desfeita)
	}
	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, caminhoDoDebito+"/desfazer", map[string]any{"motivo": "de novo"}, tokenMIA),
		http.StatusConflict, "conciliação já desfeita")

	caminhoDoCredito := "/api/v1/conciliacoes/" + credito.ConciliacaoID.String()
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, caminhoDoCredito+"/desfazer", map[string]any{"motivo": "conferir à mão"}, tokenMIA),
		http.StatusOK)
	resposta = enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/caixa/pendencias-sistema?conta="+c.contaMia.String(), nil, tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	pendencias := decodificarPagina[caixadto.MovimentoResponse](t, resposta).Dados
	if len(pendencias) != 1 || pendencias[0].Tipo != "recebimento" {
		t.Fatalf("pendências depois de desfazer = %+v", pendencias)
	}

	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/extratos/lancamentos/"+credito.ID.String()+"/conciliacao", map[string]any{
		"movimentos": []map[string]any{{"tipo": "recebimento", "id": uuid.New()}},
	}, tokenMIA), http.StatusBadRequest, "erro de validação: movimento não encontrado nesta conta ou anterior ao saldo inicial")
	resposta = enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/extratos/lancamentos/"+credito.ID.String()+"/conciliacao", map[string]any{
		"movimentos": []map[string]any{{"tipo": pendencias[0].Tipo, "id": pendencias[0].ID}},
	}, tokenMIA)
	conferirStatus(t, resposta, http.StatusCreated)
	if manual := decodificarEnvelope[caixadto.ConciliacaoResponse](t, resposta).Dados; manual.Modo != "manual" || !manual.Vigente {
		t.Errorf("conciliação manual = %+v", manual)
	}

	resposta = enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/caixa/saldos", nil, tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	if saldos := decodificarEnvelope[[]caixadto.SaldoContaResponse](t, resposta).Dados; len(saldos) != 1 || saldos[0].LancamentosPendentes != 2 {
		t.Errorf("saldos = %+v", saldos)
	}

	resposta = enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/extratos/importacoes?conta="+c.contaMia.String(), nil, tokenMIA)
	importacoes := decodificarPagina[caixadto.ImportacaoResponse](t, resposta).Dados
	conferirMensagem(t, envia(t, c.servidor, http.MethodDelete, "/api/v1/extratos/importacoes/"+importacoes[0].ID.String(), nil),
		http.StatusConflict, "importação com lançamento que já passou por conciliação não pode ser desfeita")

	resposta = enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/caixa/pagamentos-sem-conta", nil, tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	if semConta := decodificarPagina[caixadto.MovimentoResponse](t, resposta).Dados; len(semConta) != 0 {
		t.Errorf("pagamentos sem conta = %+v", semConta)
	}
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/caixa/pagamentos-sem-conta?empresa="+c.mia.String(), nil, tokenRed),
		http.StatusForbidden)
}
