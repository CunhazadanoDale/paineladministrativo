//go:build e2e

package http

import (
	"net/http"
	"testing"

	dividadto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/divida"
)

func TestDividaResumoEFiltrosDaLista(t *testing.T) {
	c := novoCenarioDividasE2E(t)
	banco := c.criar(t)

	fisco := c.corpo()
	fisco["credor_tipo"] = "fisco"
	fisco["descricao"] = "Parcelamento PERT"
	fisco["saldo_inicial_centavos"] = 90000
	fisco["data_saldo"] = c.data(-40, 0)
	fisco["parcelas"] = []map[string]any{
		{"vencimento": c.data(-10, 0), "amortizacao_centavos": 30000, "juros_centavos": 600},
		{"vencimento": c.data(20, 0), "amortizacao_centavos": 30000, "juros_centavos": 400},
		{"vencimento": c.data(0, 6), "amortizacao_centavos": 30000, "juros_centavos": 200},
	}
	delete(fisco, "conta_liberacao_id")
	delete(fisco, "liberada_em")
	resposta := enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/dividas", fisco, c.tokenMIA)
	conferirStatus(t, resposta, http.StatusCreated)
	parcelamento := decodificarEnvelope[dividadto.DividaDetalheResponse](t, resposta).Dados
	if parcelamento.Parcelas[0].Situacao != "vencida" {
		t.Fatalf("parcela vencida da carga = %+v", parcelamento.Parcelas[0])
	}
	if _, err := c.banco.Exec(`UPDATE solicitacao SET status = 'pago' WHERE id = $1`, banco.Parcelas[0].ContaAmortizacao.SolicitacaoID); err != nil {
		t.Fatalf("pagamento de teste falhou: %v", err)
	}

	resposta = enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/dividas/resumo", nil, c.tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	resumo := decodificarEnvelope[dividadto.ResumoResponse](t, resposta).Dados
	if resumo.Hoje != c.hoje.Format("2006-01-02") || resumo.SaldoDevedorCentavos != 290000 || resumo.Dividas != 2 {
		t.Fatalf("resumo = %+v", resumo)
	}
	if len(resumo.PorCredor) != 4 || resumo.PorCredor[0].CredorTipo != "banco" || resumo.PorCredor[0].SaldoDevedorCentavos != 200000 ||
		resumo.PorCredor[2].CredorTipo != "fisco" || resumo.PorCredor[2].SaldoDevedorCentavos != 90000 {
		t.Errorf("por credor = %+v", resumo.PorCredor)
	}
	if len(resumo.PorEmpresa) != 1 || resumo.PorEmpresa[0].EmpresaID != c.mia || resumo.PorEmpresa[0].Dividas != 2 {
		t.Errorf("por empresa = %+v", resumo.PorEmpresa)
	}
	if resumo.Vencido != (dividadto.TotaisResponse{AmortizacaoCentavos: 30000, JurosCentavos: 600, Parcelas: 1}) {
		t.Errorf("vencido = %+v", resumo.Vencido)
	}
	if resumo.AVencer.Parcelas != 5 || resumo.AVencer30.AmortizacaoCentavos != 30000 || resumo.AVencer30.JurosCentavos != 5950 {
		t.Errorf("a vencer = %+v, 30 dias = %+v", resumo.AVencer, resumo.AVencer30)
	}

	resposta = enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/dividas/resumo", nil, c.tokenRed)
	conferirStatus(t, resposta, http.StatusOK)
	if daRed := decodificarEnvelope[dividadto.ResumoResponse](t, resposta).Dados; daRed.Dividas != 0 || len(daRed.PorEmpresa) != 0 {
		t.Errorf("resumo da Red = %+v", daRed)
	}
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/dividas/resumo?empresa="+c.mia.String(), nil, c.tokenRed), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/dividas/resumo", nil, c.tokenComercial), http.StatusForbidden)

	resposta = enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/dividas?vencidas=true", nil, c.tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	vencidas := decodificarPagina[dividadto.DividaDaListaResponse](t, resposta).Dados
	if len(vencidas) != 1 || vencidas[0].ID != parcelamento.ID || vencidas[0].VencidoCentavos != 30600 || vencidas[0].SaldoDevedorCentavos != 90000 {
		t.Errorf("vencidas = %+v", vencidas)
	}
	resposta = enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/dividas?quitada=false&credor_tipo=banco", nil, c.tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	if abertas := decodificarPagina[dividadto.DividaDaListaResponse](t, resposta).Dados; len(abertas) != 1 || abertas[0].ID != banco.ID ||
		abertas[0].SaldoDevedorCentavos != 200000 || abertas[0].Quitada {
		t.Errorf("não quitadas do banco = %+v", abertas)
	}
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/dividas?vencidas=talvez", nil, c.tokenMIA), http.StatusBadRequest)
}
