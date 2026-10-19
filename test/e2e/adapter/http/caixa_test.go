//go:build e2e

package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	caixadto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/caixa"
	solicitacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/solicitacao"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	"github.com/google/uuid"
)

func diaE2E(deslocamento int) time.Time {
	return domaincaixa.Hoje(time.Now().UTC()).AddDate(0, 0, deslocamento)
}

func extratoE2E(saldo string, transacoes ...string) []byte {
	return []byte(`OFXHEADER:100
DATA:OFXSGML
CHARSET:1252

<OFX><BANKMSGSRSV1><STMTTRNRS><STMTRS>
<BANKACCTFROM><BANKID>0748<BRANCHID>0812<ACCTID>54321-0</BANKACCTFROM>
<BANKTRANLIST>
` + strings.Join(transacoes, "\n") + `
</BANKTRANLIST>
<LEDGERBAL><BALAMT>` + saldo + `<DTASOF>` + diaE2E(-1).Format("20060102") + `</LEDGERBAL>
</STMTRS></STMTTRNRS></BANKMSGSRSV1></OFX>`)
}

func transacaoE2E(fitid string, deslocamento int, valor, memo string) string {
	return "<STMTTRN><DTPOSTED>" + diaE2E(deslocamento).Format("20060102") + "<TRNAMT>" + valor + "<FITID>" + fitid +
		"<MEMO>" + memo + "</STMTTRN>"
}

func enviarOFX(t *testing.T, servidor *httptest.Server, conteudo []byte, token string) uuid.UUID {
	t.Helper()

	resposta := enviaMultipart(t, servidor, "/api/v1/arquivos", "extrato.ofx", conteudo, token)
	conferirStatus(t, resposta, http.StatusCreated)
	arquivo := decodificarEnvelope[solicitacaodto.ArquivoResponse](t, resposta).Dados
	if arquivo.ContentType != "application/x-ofx" {
		t.Fatalf("OFX gravado como %q", arquivo.ContentType)
	}

	return arquivo.ID
}

func importarExtrato(t *testing.T, servidor *httptest.Server, corpo map[string]any, token string, status int) caixadto.RelatorioImportacaoResponse {
	t.Helper()

	resposta := enviaComToken(t, servidor, http.MethodPost, "/api/v1/extratos/importacao", corpo, token)
	conferirStatus(t, resposta, status)

	return decodificarEnvelope[caixadto.RelatorioImportacaoResponse](t, resposta).Dados
}

func TestFluxoDeExtratoBancario(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA")
	red := criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond")
	tokenMIA := tokenFinanceiroDasEmpresas(t, servidor, "Fábio MIA", "fabio@exemplo.com", mia.ID)
	tokenRed := tokenFinanceiroDasEmpresas(t, servidor, "Rita Red", "rita@exemplo.com", red.ID)

	conta := criarContaE2E(t, servidor, map[string]any{
		"empresa_id": mia.ID, "banco": "748", "tipo": "corrente", "agencia": "0812", "conta": "54321-0", "apelido": "Sicredi",
	}, tokenMIA)
	abertura := "/api/v1/caixa/contas/" + conta.ID.String() + "/abertura"

	primeiro := enviarOFX(t, servidor, extratoE2E("1949.50",
		transacaoE2E("A", -40, "-1.00", "ANTES DO CORTE"),
		transacaoE2E("B", -5, "-250.00", "BOLETO"),
		transacaoE2E("C", -3, "1200.00", "TED RECEBIDA"),
	), tokenMIA)

	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/extratos/importacao", map[string]any{"arquivo_id": primeiro}, tokenMIA),
		http.StatusBadRequest, "erro de validação: conta sem saldo inicial; informe a abertura antes de importar o extrato")
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, abertura, nil, tokenMIA), http.StatusNotFound)

	resposta := enviaComToken(t, servidor, http.MethodPut, abertura, map[string]any{
		"data_corte": diaE2E(-30).Format("2006-01-02"), "saldo_centavos": 100000,
	}, tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	if criada := decodificarEnvelope[caixadto.AberturaResponse](t, resposta).Dados; criada.Versao != 1 || criada.SaldoCentavos != 100000 {
		t.Errorf("abertura = %+v", criada)
	}
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPut, abertura, map[string]any{
		"data_corte": diaE2E(-30).Format("2006-01-02"), "saldo_centavos": 1,
	}, tokenMIA), http.StatusConflict, "a conta já tem saldo inicial; envie a versão para alterar")
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPut, abertura, map[string]any{
		"data_corte": diaE2E(0).Format("2006-01-02"), "saldo_centavos": 1, "versao": 1,
	}, tokenMIA), http.StatusBadRequest)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, abertura, nil, tokenRed), http.StatusForbidden)

	simulado := importarExtrato(t, servidor, map[string]any{"arquivo_id": primeiro, "simular": true}, tokenMIA, http.StatusOK)
	if !simulado.Simulacao || simulado.ImportacaoID != nil || simulado.Novas != 2 || simulado.Total != 3 || len(simulado.Ignoradas) != 1 {
		t.Errorf("simulação = %+v", simulado)
	}

	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/extratos/importacao", map[string]any{"arquivo_id": primeiro}, tokenRed),
		http.StatusForbidden)

	importado := importarExtrato(t, servidor, map[string]any{"arquivo_id": primeiro}, tokenMIA, http.StatusCreated)
	if importado.ImportacaoID == nil || importado.ContaBancariaID != conta.ID || importado.Novas != 2 || importado.Repetidas != 0 {
		t.Errorf("importação = %+v", importado)
	}
	if len(importado.Ignoradas) != 1 || importado.Ignoradas[0].Motivo != "anterior_ao_saldo_inicial" {
		t.Errorf("ignoradas = %+v", importado.Ignoradas)
	}
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/extratos/importacao", map[string]any{"arquivo_id": primeiro}, tokenMIA),
		http.StatusConflict, "arquivo já importado em outro extrato")

	segundo := enviarOFX(t, servidor, extratoE2E("1936.60",
		transacaoE2E("C", -3, "1200.00", "TED RECEBIDA"),
		transacaoE2E("D", -1, "-12.90", "TARIFA"),
	), tokenMIA)
	repetido := importarExtrato(t, servidor, map[string]any{"arquivo_id": segundo, "conta_bancaria_id": conta.ID}, tokenMIA, http.StatusCreated)
	if repetido.Novas != 1 || repetido.Repetidas != 1 {
		t.Errorf("segunda importação = %+v", repetido)
	}

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/extratos/lancamentos?conta="+conta.ID.String(), nil, tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	lancamentos := decodificarPagina[caixadto.LancamentoResponse](t, resposta).Dados
	if len(lancamentos) != 3 || lancamentos[0].Descricao != "BOLETO" || lancamentos[0].Tipo != "debito" || lancamentos[1].Tipo != "credito" {
		t.Errorf("lançamentos = %+v", lancamentos)
	}
	resposta = enviaComToken(t, servidor, http.MethodGet,
		"/api/v1/extratos/lancamentos?conta="+conta.ID.String()+"&de="+diaE2E(-2).Format("2006-01-02"), nil, tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	if recentes := decodificarPagina[caixadto.LancamentoResponse](t, resposta).Dados; len(recentes) != 1 || recentes[0].ValorCentavos != -1290 {
		t.Errorf("lançamentos recentes = %+v", recentes)
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/extratos/lancamentos", nil, tokenMIA), http.StatusBadRequest)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/extratos/lancamentos?conta="+conta.ID.String(), nil, tokenRed),
		http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/caixa/saldos", nil, tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	saldos := decodificarEnvelope[[]caixadto.SaldoContaResponse](t, resposta).Dados
	if len(saldos) != 1 || saldos[0].SaldoAtualCentavos == nil || *saldos[0].SaldoAtualCentavos != 193710 {
		t.Fatalf("saldos = %+v", saldos)
	}
	if saldos[0].DiferencaBancoCentavos == nil || *saldos[0].DiferencaBancoCentavos != -50 || saldos[0].BancoNome != "Sicredi" {
		t.Errorf("saldo do Sicredi = %+v", saldos[0])
	}
	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/caixa/saldos", nil, tokenRed)
	conferirStatus(t, resposta, http.StatusOK)
	if daRed := decodificarEnvelope[[]caixadto.SaldoContaResponse](t, resposta).Dados; len(daRed) != 0 {
		t.Errorf("saldos vistos pela Red = %+v", daRed)
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/caixa/saldos?empresa="+mia.ID.String(), nil, tokenRed), http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/extratos/importacoes?conta="+conta.ID.String(), nil, tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	if importacoes := decodificarPagina[caixadto.ImportacaoResponse](t, resposta).Dados; len(importacoes) != 2 || importacoes[0].ID != *repetido.ImportacaoID {
		t.Errorf("importações = %+v", importacoes)
	}

	conferirStatus(t, enviaComToken(t, servidor, http.MethodDelete, "/api/v1/extratos/importacoes/"+importado.ImportacaoID.String(), nil, tokenMIA),
		http.StatusForbidden)
	conferirMensagem(t, envia(t, servidor, http.MethodDelete, "/api/v1/extratos/importacoes/"+importado.ImportacaoID.String(), nil),
		http.StatusConflict, "só a importação mais recente da conta pode ser desfeita")
	conferirStatus(t, envia(t, servidor, http.MethodDelete, "/api/v1/extratos/importacoes/"+repetido.ImportacaoID.String(), nil), http.StatusNoContent)
	conferirStatus(t, envia(t, servidor, http.MethodDelete, "/api/v1/extratos/importacoes/"+uuid.NewString(), nil), http.StatusNotFound)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/caixa/saldos", nil, tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	if depois := decodificarEnvelope[[]caixadto.SaldoContaResponse](t, resposta).Dados; *depois[0].SaldoAtualCentavos != 195000 {
		t.Errorf("saldo depois de desfazer = %d", *depois[0].SaldoAtualCentavos)
	}

	conferirStatus(t, enviaComToken(t, servidor, http.MethodDelete, "/api/v1/arquivos/"+primeiro.String(), nil, tokenMIA), http.StatusBadRequest)
}
