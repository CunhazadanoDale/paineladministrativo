//go:build e2e

package http

import (
	"net/http"
	"strings"
	"testing"

	caixadto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/caixa"
	"github.com/google/uuid"
)

func extratoDoItauE2E(transacoes ...string) []byte {
	return []byte(`OFXHEADER:100
DATA:OFXSGML

<OFX><BANKMSGSRSV1><STMTTRNRS><STMTRS>
<BANKACCTFROM><BANKID>341<BRANCHID>1234<ACCTID>98765-4</BANKACCTFROM>
<BANKTRANLIST>
` + strings.Join(transacoes, "\n") + `
</BANKTRANLIST>
</STMTRS></STMTTRNRS></BANKMSGSRSV1></OFX>`)
}

func subcategoriaE2E(t *testing.T, c *cenarioTitulosE2E, nome, natureza, linha string) uuid.UUID {
	t.Helper()

	raiz := criarCategoriaFinanceiraE2E(t, c.servidor, map[string]any{"nome": nome, "natureza": natureza, "linha_dre": linha}, tokenDoTeste)

	return criarCategoriaFinanceiraE2E(t, c.servidor, map[string]any{"nome": nome + " - sub", "categoria_pai_id": raiz.ID}, tokenDoTeste).ID
}

func TestClassificacaoETransferenciaEntreContas(t *testing.T) {
	c := novoCenarioTitulosE2E(t)
	tokenMIA := tokenFinanceiroDasEmpresas(t, c.servidor, "Fábio MIA", "fabio@exemplo.com", c.mia)
	tokenRed := tokenFinanceiroDasEmpresas(t, c.servidor, "Rita Red", "rita@exemplo.com", c.red)
	for _, conta := range []uuid.UUID{c.contaMia, c.contaRed} {
		conferirStatus(t, envia(t, c.servidor, http.MethodPut, "/api/v1/caixa/contas/"+conta.String()+"/abertura", map[string]any{
			"data_corte": diaE2E(-30).Format("2006-01-02"), "saldo_centavos": 0,
		}), http.StatusOK)
	}
	tarifas := subcategoriaE2E(t, c, "Despesas bancárias", "despesa", "resultado_financeiro")
	rendimentos := subcategoriaE2E(t, c, "Receitas financeiras", "receita", "resultado_financeiro")
	entreContas := subcategoriaE2E(t, c, "Entre contas", "transferencia", "entre_contas")

	arquivo := enviarOFX(t, c.servidor, extratoE2E("1000.00",
		transacaoE2E("TAR", -2, "-12.90", "TARIFA PACOTE"),
		transacaoE2E("REN", -1, "3.50", "RENDIMENTO"),
		transacaoE2E("TRF", -1, "-1000.00", "TED MESMA TITULARIDADE"),
	), tokenMIA)
	importarExtrato(t, c.servidor, map[string]any{"arquivo_id": arquivo}, tokenMIA, http.StatusCreated)
	linhas := lancamentosE2E(t, c, "", tokenMIA)
	tarifa, rendimento, transferencia := lancamentoComValor(t, linhas, -1290), lancamentoComValor(t, linhas, 350), lancamentoComValor(t, linhas, -100000)
	classificar := func(linha uuid.UUID) string {
		return "/api/v1/extratos/lancamentos/" + linha.String() + "/classificacao"
	}

	resposta := enviaComToken(t, c.servidor, http.MethodPost, classificar(tarifa.ID), map[string]any{"categoria_financeira_id": tarifas}, tokenMIA)
	conferirStatus(t, resposta, http.StatusCreated)
	classificada := decodificarEnvelope[caixadto.ClassificacaoResponse](t, resposta).Dados
	if classificada.LancamentoCaixa.ValorCentavos != -1290 || classificada.LancamentoCaixa.Descricao != "TARIFA PACOTE" ||
		classificada.LancamentoCaixa.Competencia != diaE2E(-2).Format("2006-01") || classificada.Conciliacao.Modo != "classificacao" {
		t.Errorf("classificação = %+v", classificada)
	}
	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, classificar(rendimento.ID), map[string]any{"categoria_financeira_id": tarifas}, tokenMIA),
		http.StatusBadRequest, "erro de validação: linha de crédito só recebe categoria de receita; transferência entre contas tem rota própria")
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, classificar(rendimento.ID), map[string]any{
		"categoria_financeira_id": rendimentos, "descricao": "Rendimento da conta",
	}, tokenMIA), http.StatusCreated)
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, classificar(rendimento.ID), map[string]any{"categoria_financeira_id": rendimentos}, tokenMIA),
		http.StatusConflict)
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, classificar(tarifa.ID), map[string]any{"categoria_financeira_id": tarifas}, tokenRed),
		http.StatusForbidden)

	caminhoDaTransferencia := "/api/v1/extratos/lancamentos/" + transferencia.ID.String() + "/transferencia"
	corpoDaTransferencia := map[string]any{"conta_contraparte_id": c.contaRed, "categoria_financeira_id": entreContas}
	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, caminhoDaTransferencia, corpoDaTransferencia, tokenMIA),
		http.StatusForbidden, "perfil sem acesso a esta empresa")
	resposta = envia(t, c.servidor, http.MethodPost, caminhoDaTransferencia, corpoDaTransferencia)
	conferirStatus(t, resposta, http.StatusCreated)
	registrada := decodificarEnvelope[caixadto.TransferenciaRegistradaResponse](t, resposta).Dados
	if registrada.Transferencia.ContaOrigemID != c.contaMia || registrada.Transferencia.ContaDestinoID != c.contaRed ||
		!registrada.Transferencia.EntreEmpresas || registrada.Transferencia.ValorCentavos != 100000 ||
		registrada.Conciliacao.Movimentos[0].Tipo != "transferencia_saida" {
		t.Errorf("transferência = %+v", registrada)
	}

	resposta = enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/caixa/pendencias-sistema?conta="+c.contaRed.String(), nil, tokenRed)
	conferirStatus(t, resposta, http.StatusOK)
	if pendentes := decodificarPagina[caixadto.MovimentoResponse](t, resposta).Dados; len(pendentes) != 1 ||
		pendentes[0].Tipo != "transferencia_entrada" || pendentes[0].ValorCentavos != 100000 {
		t.Errorf("pendências da Red = %+v", pendentes)
	}

	arquivoDaRed := enviarOFX(t, c.servidor, extratoDoItauE2E(transacaoE2E("TEDR", -1, "1000.00", "TED RECEBIDA MIA")), tokenRed)
	importarExtrato(t, c.servidor, map[string]any{"arquivo_id": arquivoDaRed}, tokenRed, http.StatusCreated)
	resposta = enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/extratos/conciliar-automatico", map[string]any{
		"conta_bancaria_id": c.contaRed, "de": diaE2E(-5).Format("2006-01-02"), "ate": diaE2E(0).Format("2006-01-02"),
	}, tokenRed)
	conferirStatus(t, resposta, http.StatusOK)
	if resultado := decodificarEnvelope[caixadto.ResultadoAutomaticoResponse](t, resposta).Dados; resultado.Conciliadas != 1 {
		t.Errorf("conciliação automática da Red = %+v", resultado)
	}

	resposta = enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/caixa/lancamentos?conta="+c.contaMia.String(), nil, tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	if doCaixa := decodificarPagina[caixadto.LancamentoCaixaResponse](t, resposta).Dados; len(doCaixa) != 2 {
		t.Errorf("lançamentos de caixa = %+v", doCaixa)
	}
	resposta = enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/caixa/transferencias", nil, tokenRed)
	conferirStatus(t, resposta, http.StatusOK)
	if daRed := decodificarPagina[caixadto.TransferenciaResponse](t, resposta).Dados; len(daRed) != 1 || daRed[0].ID != registrada.Transferencia.ID {
		t.Errorf("transferências vistas pela Red = %+v", daRed)
	}
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/caixa/lancamentos?empresa="+c.mia.String(), nil, tokenRed), http.StatusForbidden)

	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/conciliacoes/"+classificada.Conciliacao.ID.String()+"/desfazer",
		map[string]any{"motivo": "categoria errada"}, tokenMIA), http.StatusOK)
	resposta = enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/caixa/lancamentos?conta="+c.contaMia.String(), nil, tokenMIA)
	estornados := 0
	for _, lancamento := range decodificarPagina[caixadto.LancamentoCaixaResponse](t, resposta).Dados {
		if lancamento.Estornado {
			estornados++
		}
	}
	if estornados != 1 {
		t.Errorf("lançamentos estornados = %d", estornados)
	}

	conferirStatus(t, envia(t, c.servidor, http.MethodPost, "/api/v1/conciliacoes/"+registrada.Conciliacao.ID.String()+"/desfazer",
		map[string]any{"motivo": "era aplicação"}), http.StatusOK)
	resposta = enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/extratos/lancamentos?conta="+c.contaRed.String()+"&situacao=invalidado", nil, tokenRed)
	conferirStatus(t, resposta, http.StatusOK)
	if invalidados := decodificarPagina[caixadto.LancamentoResponse](t, resposta).Dados; len(invalidados) != 1 {
		t.Errorf("ponta da Red depois de desfazer a transferência = %+v", invalidados)
	}
}
