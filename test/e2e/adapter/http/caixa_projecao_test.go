//go:build e2e

package http

import (
	"net/http"
	"testing"

	caixadto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/caixa"
)

func TestProjecaoEMovimentoDiarioDoCaixa(t *testing.T) {
	c := novoCenarioTitulosE2E(t)
	tokenMIA := tokenFinanceiroDasEmpresas(t, c.servidor, "Fábio MIA", "fabio@exemplo.com", c.mia)
	tokenRed := tokenFinanceiroDasEmpresas(t, c.servidor, "Rita Red", "rita@exemplo.com", c.red)

	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPut, "/api/v1/caixa/contas/"+c.contaMia.String()+"/abertura", map[string]any{
		"data_corte": diaE2E(-30).Format("2006-01-02"), "saldo_centavos": 100000,
	}, tokenMIA), http.StatusOK)
	arquivo := enviarOFX(t, c.servidor, extratoE2E("1015.00",
		transacaoE2E("REC", -5, "20.00", "PIX RECEBIDO"),
		transacaoE2E("TAR", -3, "-5.00", "TARIFA"),
	), tokenMIA)
	importarExtrato(t, c.servidor, map[string]any{"arquivo_id": arquivo}, tokenMIA, http.StatusCreated)

	c.criar(t, c.corpo(c.mia, "ALG-1"), tokenMIA)
	tokenSolicitante, _ := usuariosParaSolicitacao(t, c.servidor, "projecao")
	criarSolicitacaoNaEmpresa(t, c.servidor, tokenSolicitante, c.mia)

	ate := diaE2E(40).Format("2006-01-02")
	resposta := enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/caixa/projecao?ate="+ate, nil, tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	projecao := decodificarEnvelope[caixadto.ProjecaoResponse](t, resposta).Dados
	if projecao.SaldoDePartida.ExtratoCentavos != 101500 || projecao.SaldoDePartida.TotalCentavos != 101500 ||
		projecao.SaldoDePartida.ContasConsideradas != 1 || len(projecao.Dias) != 41 {
		t.Errorf("saldo de partida = %+v, dias %d", projecao.SaldoDePartida, len(projecao.Dias))
	}
	if projecao.SaldoFinalCentavos != 101500+150000-50000 || projecao.De != diaE2E(0).Format("2006-01-02") || projecao.Ate != ate {
		t.Errorf("projeção = de %s a %s, saldo final %d", projecao.De, projecao.Ate, projecao.SaldoFinalCentavos)
	}

	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/caixa/projecao?ate="+diaE2E(91).Format("2006-01-02"), nil, tokenMIA),
		http.StatusBadRequest)
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/caixa/projecao?empresa="+c.mia.String(), nil, tokenRed), http.StatusForbidden)
	resposta = enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/caixa/projecao", nil, tokenRed)
	conferirStatus(t, resposta, http.StatusOK)
	if daRed := decodificarEnvelope[caixadto.ProjecaoResponse](t, resposta).Dados; daRed.SaldoDePartida.ContasConsideradas != 0 ||
		daRed.SaldoDePartida.ContasSemAbertura != 1 || daRed.SaldoFinalCentavos != 0 {
		t.Errorf("projeção da Red = %+v", daRed.SaldoDePartida)
	}

	resposta = enviaComToken(t, c.servidor, http.MethodGet,
		"/api/v1/caixa/movimento-diario?conta="+c.contaMia.String()+"&de="+diaE2E(-10).Format("2006-01-02"), nil, tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	diario := decodificarEnvelope[[]caixadto.MovimentoDiarioResponse](t, resposta).Dados
	if len(diario) != 2 || diario[0].EntradasCentavos != 2000 || diario[1].SaidasCentavos != 500 || diario[1].SaldoCentavos != 101500 {
		t.Errorf("movimento diário = %+v", diario)
	}
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/caixa/movimento-diario?conta="+c.contaMia.String(), nil, tokenRed),
		http.StatusForbidden)
}
