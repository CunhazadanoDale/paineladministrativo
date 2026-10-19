//go:build e2e

package http

import (
	"net/http"
	"strings"
	"testing"
	"time"

	caixadto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/caixa"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
)

func extratoComSaldoE2E(saldo string, saldoEm time.Time, transacoes ...string) []byte {
	return []byte(`OFXHEADER:100
DATA:OFXSGML

<OFX><BANKMSGSRSV1><STMTTRNRS><STMTRS>
<BANKACCTFROM><BANKID>0748<BRANCHID>0812<ACCTID>54321-0</BANKACCTFROM>
<BANKTRANLIST>
` + strings.Join(transacoes, "\n") + `
</BANKTRANLIST>
<LEDGERBAL><BALAMT>` + saldo + `<DTASOF>` + saldoEm.Format("20060102") + `</LEDGERBAL>
</STMTRS></STMTTRNRS></BANKMSGSRSV1></OFX>`)
}

func transacaoNaDataE2E(fitid string, data time.Time, valor, memo string) string {
	return "<STMTTRN><DTPOSTED>" + data.Format("20060102") + "<TRNAMT>" + valor + "<FITID>" + fitid + "<MEMO>" + memo + "</STMTTRN>"
}

func TestFechamentoMensalDoCaixa(t *testing.T) {
	c := novoCenarioTitulosE2E(t)
	tokenMIA := tokenFinanceiroDasEmpresas(t, c.servidor, "Fábio MIA", "fabio@exemplo.com", c.mia)
	mesAnterior := domaincaixa.CompetenciaDe(diaE2E(0)).AddDate(0, -1, 0)
	fimDoMesAnterior := domaincaixa.FimDaCompetencia(mesAnterior)
	competencia := mesAnterior.Format("2006-01")
	tarifas := subcategoriaE2E(t, c, "Despesas bancárias", "despesa", "resultado_financeiro")
	rendimentos := subcategoriaE2E(t, c, "Receitas financeiras", "receita", "resultado_financeiro")

	caminhoDaAbertura := "/api/v1/caixa/contas/" + c.contaMia.String() + "/abertura"
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPut, caminhoDaAbertura, map[string]any{
		"data_corte": mesAnterior.AddDate(0, 0, -1).Format("2006-01-02"), "saldo_centavos": 100000,
	}, tokenMIA), http.StatusOK)

	arquivo := enviarOFX(t, c.servidor, extratoComSaldoE2E("1087.10", fimDoMesAnterior,
		transacaoNaDataE2E("TAR", mesAnterior.AddDate(0, 0, 4), "-12.90", "TARIFA"),
		transacaoNaDataE2E("REN", mesAnterior.AddDate(0, 0, 9), "100.00", "RENDIMENTO"),
	), tokenMIA)
	importarExtrato(t, c.servidor, map[string]any{"arquivo_id": arquivo}, tokenMIA, http.StatusCreated)

	caminhoDaPrevia := "/api/v1/caixa/fechamentos/previa?conta=" + c.contaMia.String() + "&competencia=" + competencia
	resposta := enviaComToken(t, c.servidor, http.MethodGet, caminhoDaPrevia, nil, tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	if previa := decodificarEnvelope[caixadto.PreviaResponse](t, resposta).Dados; previa.PodeFechar || previa.LancamentosPendentes != 2 ||
		previa.SaldoInicialCentavos != 100000 || previa.SaldoFinalCentavos != 108710 {
		t.Errorf("prévia com pendências = %+v", previa)
	}

	corpoDoFechamento := map[string]any{"conta_bancaria_id": c.contaMia, "competencia": competencia}
	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/caixa/fechamentos", corpoDoFechamento, tokenMIA), http.StatusConflict,
		"o mês tem 2 lançamento(s) pendente(s) e 0 conciliação(ões) invalidada(s); concilie, classifique ou desfaça antes de fechar")

	linhas := lancamentosE2E(t, c, "", tokenMIA)
	resposta = enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/extratos/lancamentos/"+lancamentoComValor(t, linhas, -1290).ID.String()+"/classificacao",
		map[string]any{"categoria_financeira_id": tarifas}, tokenMIA)
	conferirStatus(t, resposta, http.StatusCreated)
	tarifa := decodificarEnvelope[caixadto.ClassificacaoResponse](t, resposta).Dados
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost,
		"/api/v1/extratos/lancamentos/"+lancamentoComValor(t, linhas, 10000).ID.String()+"/classificacao",
		map[string]any{"categoria_financeira_id": rendimentos}, tokenMIA), http.StatusCreated)

	resposta = enviaComToken(t, c.servidor, http.MethodGet, caminhoDaPrevia, nil, tokenMIA)
	if previa := decodificarEnvelope[caixadto.PreviaResponse](t, resposta).Dados; !previa.PodeFechar || previa.SaldoBancoCentavos == nil ||
		*previa.SaldoBancoCentavos != 108710 {
		t.Errorf("prévia liberada = %+v", previa)
	}

	resposta = enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/caixa/fechamentos", corpoDoFechamento, tokenMIA)
	conferirStatus(t, resposta, http.StatusCreated)
	fechamento := decodificarEnvelope[caixadto.DetalheFechamentoResponse](t, resposta).Dados
	if fechamento.SaldoFinalCentavos != 108710 || fechamento.EntradasCentavos != 10000 || fechamento.SaidasCentavos != 1290 ||
		fechamento.Competencia != competencia || !fechamento.Vigente {
		t.Errorf("fechamento = %+v", fechamento)
	}

	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/caixa/fechamentos", corpoDoFechamento, tokenMIA),
		http.StatusConflict, "mês já fechado nesta conta")
	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/caixa/fechamentos", map[string]any{
		"conta_bancaria_id": c.contaMia, "competencia": diaE2E(0).Format("2006-01"),
	}, tokenMIA), http.StatusBadRequest, "erro de validação: só é possível fechar mês encerrado")

	resposta = enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/caixa/fechamentos?conta="+c.contaMia.String(), nil, tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	if lista := decodificarPagina[caixadto.FechamentoResponse](t, resposta).Dados; len(lista) != 1 || lista[0].ID != fechamento.ID {
		t.Errorf("fechamentos = %+v", lista)
	}
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/caixa/fechamentos/"+fechamento.ID.String(), nil, tokenMIA), http.StatusOK)

	caminhoDoDesfazer := "/api/v1/conciliacoes/" + tarifa.Conciliacao.ID.String() + "/desfazer"
	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, caminhoDoDesfazer, map[string]any{"motivo": "categoria errada"}, tokenMIA),
		http.StatusConflict, "mês fechado nesta conta; reabra o fechamento para alterar")
	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPut, caminhoDaAbertura, map[string]any{
		"data_corte": mesAnterior.AddDate(0, 0, -1).Format("2006-01-02"), "saldo_centavos": 1, "versao": 1,
	}, tokenMIA), http.StatusConflict, "a conta já tem mês fechado; reabra os fechamentos para mudar o saldo inicial")

	caminhoDaReabertura := "/api/v1/caixa/fechamentos/" + fechamento.ID.String() + "/reabrir"
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, caminhoDaReabertura, map[string]any{"motivo": "tarifa errada"}, tokenMIA),
		http.StatusForbidden)
	resposta = envia(t, c.servidor, http.MethodPost, caminhoDaReabertura, map[string]any{"motivo": "tarifa errada"})
	conferirStatus(t, resposta, http.StatusOK)
	if reaberto := decodificarEnvelope[caixadto.FechamentoResponse](t, resposta).Dados; reaberto.Vigente || reaberto.MotivoReabertura == nil {
		t.Errorf("reaberto = %+v", reaberto)
	}
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, caminhoDoDesfazer, map[string]any{"motivo": "categoria errada"}, tokenMIA),
		http.StatusOK)
}
