//go:build e2e

package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	dividadto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/divida"
	solicitacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/solicitacao"
	domaindivida "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/divida"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type cenarioDividasE2E struct {
	servidor       *httptest.Server
	banco          *sqlx.DB
	mia            uuid.UUID
	red            uuid.UUID
	tokenMIA       string
	tokenRed       string
	tokenComercial string
	credor         uuid.UUID
	amortizacao    uuid.UUID
	juros          uuid.UUID
	despesaFixa    uuid.UUID
	contaMIA       uuid.UUID
	hoje           time.Time
}

func novoCenarioDividasE2E(t *testing.T) *cenarioDividasE2E {
	t.Helper()

	servidor, banco := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	red := criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond").ID
	raizEmprestimos := criarCategoriaFinanceiraE2E(t, servidor, map[string]any{
		"nome": "Empréstimos", "natureza": "despesa", "linha_dre": "fora_dre",
	}, tokenDoTeste)
	raizFinanceiras := criarCategoriaFinanceiraE2E(t, servidor, map[string]any{
		"nome": "Despesas financeiras", "natureza": "despesa", "linha_dre": "resultado_financeiro",
	}, tokenDoTeste)
	raizFixas := criarCategoriaFinanceiraE2E(t, servidor, map[string]any{
		"nome": "Fixas", "natureza": "despesa", "linha_dre": "despesa_fixa",
	}, tokenDoTeste)

	return &cenarioDividasE2E{
		servidor:       servidor,
		banco:          banco,
		mia:            mia,
		red:            red,
		tokenMIA:       tokenFinanceiroDasEmpresas(t, servidor, "Fábio MIA", "fabio@exemplo.com", mia),
		tokenRed:       tokenFinanceiroDasEmpresas(t, servidor, "Rita Red", "rita@exemplo.com", red),
		tokenComercial: usuarioComercialDasEmpresas(t, servidor, "Carla MIA", "carla@exemplo.com", mia),
		credor:         criarFornecedorE2E(t, servidor, map[string]any{"documento": "01.181.521/0001-55", "nome": "Banco Sicredi"}, tokenDoTeste).ID,
		amortizacao:    criarCategoriaFinanceiraE2E(t, servidor, map[string]any{"nome": "Amortização", "categoria_pai_id": raizEmprestimos.ID}, tokenDoTeste).ID,
		juros:          criarCategoriaFinanceiraE2E(t, servidor, map[string]any{"nome": "Juros", "categoria_pai_id": raizFinanceiras.ID}, tokenDoTeste).ID,
		despesaFixa:    criarCategoriaFinanceiraE2E(t, servidor, map[string]any{"nome": "Aluguel", "categoria_pai_id": raizFixas.ID}, tokenDoTeste).ID,
		contaMIA: criarContaE2E(t, servidor, map[string]any{
			"empresa_id": mia, "banco": "748", "tipo": "corrente", "agencia": "0101", "conta": "55001", "apelido": "Sicredi MIA",
		}, tokenDoTeste).ID,
		hoje: domaindivida.Hoje(time.Now()),
	}
}

func (c *cenarioDividasE2E) data(dias, meses int) string {
	return c.hoje.AddDate(0, meses, dias).Format("2006-01-02")
}

func (c *cenarioDividasE2E) corpo() map[string]any {
	return map[string]any{
		"empresa_id": c.mia, "fornecedor_id": c.credor, "credor_tipo": "banco", "descricao": "Capital de giro Sicredi",
		"contrato": "CG-2026-77", "valor_original_centavos": 500000, "taxa_bp": 185, "taxa_periodo": "mensal", "indexador": "cdi",
		"conta_liberacao_id": c.contaMIA, "liberada_em": c.data(-1, 0),
		"saldo_inicial_centavos": 300000, "data_saldo": c.data(-1, 0),
		"categoria_amortizacao_id": c.amortizacao, "categoria_juros_id": c.juros,
		"parcelas": []map[string]any{
			{"vencimento": c.data(0, 0), "amortizacao_centavos": 100000, "juros_centavos": 5550},
			{"vencimento": c.data(0, 2), "amortizacao_centavos": 100000, "juros_centavos": 3700},
			{"vencimento": c.data(0, 4), "amortizacao_centavos": 100000, "juros_centavos": 1850},
		},
	}
}

func (c *cenarioDividasE2E) criar(t *testing.T) dividadto.DividaDetalheResponse {
	t.Helper()

	resposta := enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/dividas", c.corpo(), c.tokenMIA)
	conferirStatus(t, resposta, http.StatusCreated)

	return decodificarEnvelope[dividadto.DividaDetalheResponse](t, resposta).Dados
}

func (c *cenarioDividasE2E) obter(t *testing.T, id uuid.UUID) dividadto.DividaDetalheResponse {
	t.Helper()

	resposta := enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/dividas/"+id.String(), nil, c.tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)

	return decodificarEnvelope[dividadto.DividaDetalheResponse](t, resposta).Dados
}

func (c *cenarioDividasE2E) conta(t *testing.T, id uuid.UUID) solicitacaodto.SolicitacaoResponse {
	t.Helper()

	resposta := enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/solicitacoes/"+id.String(), nil, c.tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)

	return decodificarEnvelope[solicitacaodto.SolicitacaoResponse](t, resposta).Dados
}

func TestDividaCriadaGeraAsDuasContasDoMes(t *testing.T) {
	c := novoCenarioDividasE2E(t)

	corpo := c.corpo()
	corpo["saldo_inicial_centavos"] = 300001
	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/dividas", corpo, c.tokenMIA), http.StatusBadRequest,
		"erro de validação: a soma das amortizações (R$ 3000,00) deve ser igual ao saldo devedor na data do saldo (R$ 3000,01)")
	corpo = c.corpo()
	corpo["categoria_juros_id"] = c.despesaFixa
	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/dividas", corpo, c.tokenMIA), http.StatusBadRequest,
		"erro de validação: categoria financeira de juros deve ser de despesa na linha resultado_financeiro")
	corpo = c.corpo()
	corpo["parcelas"] = []map[string]any{{"vencimento": c.data(-1, 0), "amortizacao_centavos": 300000}}
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/dividas", corpo, c.tokenMIA), http.StatusBadRequest)
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/dividas", c.corpo(), c.tokenComercial), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/dividas", c.corpo(), c.tokenRed), http.StatusForbidden)

	divida := c.criar(t)
	if divida.Versao != 1 || divida.Status != "ativa" || divida.SaldoDevedorCentavos != 300000 || divida.Quitada ||
		divida.TaxaBP == nil || *divida.TaxaBP != 185 || divida.Indexador != "cdi" || len(divida.Parcelas) != 3 {
		t.Fatalf("dívida criada = %+v", divida)
	}
	primeira := divida.Parcelas[0]
	if primeira.Status != "gerada" || primeira.Situacao != "a_vencer" || primeira.ContaAmortizacao == nil || primeira.ContaJuros == nil {
		t.Fatalf("parcela do mês = %+v", primeira)
	}
	if divida.Parcelas[1].Status != "prevista" || divida.Parcelas[1].ContaJuros != nil {
		t.Fatalf("parcela futura = %+v", divida.Parcelas[1])
	}

	juros := c.conta(t, primeira.ContaJuros.SolicitacaoID)
	if juros.Origem != "divida_juros" || juros.OrigemID == nil || *juros.OrigemID != primeira.ID || juros.ValorCentavos != 5550 ||
		juros.CategoriaFinanceiraID == nil || *juros.CategoriaFinanceiraID != c.juros || juros.FornecedorID == nil ||
		*juros.FornecedorID != c.credor || juros.Competencia != c.hoje.Format("2006-01") || juros.Status != "pendente_aprovacao" {
		t.Errorf("conta de juros = %+v", juros)
	}
	amortizacao := c.conta(t, primeira.ContaAmortizacao.SolicitacaoID)
	if amortizacao.Origem != "divida_amortizacao" || amortizacao.ValorCentavos != 100000 || *amortizacao.CategoriaFinanceiraID != c.amortizacao {
		t.Errorf("conta de amortização = %+v", amortizacao)
	}

	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/dividas/"+divida.ID.String(), nil, c.tokenRed), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/dividas/"+divida.ID.String(), nil, c.tokenComercial), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/dividas/"+uuid.NewString(), nil, c.tokenMIA), http.StatusNotFound)

	resposta := enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/dividas?credor_tipo=banco", nil, c.tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	if pagina := decodificarPagina[dividadto.DividaResponse](t, resposta); len(pagina.Dados) != 1 || pagina.Dados[0].ID != divida.ID {
		t.Errorf("lista = %+v", pagina.Dados)
	}
	resposta = enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/dividas", nil, c.tokenRed)
	conferirStatus(t, resposta, http.StatusOK)
	if pagina := decodificarPagina[dividadto.DividaResponse](t, resposta); len(pagina.Dados) != 0 {
		t.Errorf("lista da Red = %+v", pagina.Dados)
	}
}

func TestDividaAlteraParcelaEGeraACompetencia(t *testing.T) {
	c := novoCenarioDividasE2E(t)
	divida := c.criar(t)
	caminho := "/api/v1/dividas/" + divida.ID.String()
	segunda := divida.Parcelas[1]

	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPut, caminho+"/parcelas/"+divida.Parcelas[0].ID.String(), map[string]any{
		"versao": 1, "vencimento": c.data(1, 0), "juros_centavos": 1,
	}, c.tokenMIA), http.StatusConflict)
	resposta := enviaComToken(t, c.servidor, http.MethodPut, caminho+"/parcelas/"+segunda.ID.String(), map[string]any{
		"versao": 1, "vencimento": c.data(3, 2), "juros_centavos": 4100,
	}, c.tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	alterada := decodificarEnvelope[dividadto.DividaDetalheResponse](t, resposta).Dados
	if alterada.Versao != 2 || alterada.Parcelas[1].JurosCentavos != 4100 || alterada.Parcelas[1].Vencimento != c.data(3, 2) {
		t.Fatalf("parcela alterada = %+v", alterada.Parcelas[1])
	}

	resposta = enviaComToken(t, c.servidor, http.MethodPut, caminho, map[string]any{
		"versao": 2, "credor_tipo": "banco", "descricao": "Capital de giro renovado", "taxa_bp": 170, "taxa_periodo": "mensal",
		"categoria_amortizacao_id": c.amortizacao, "categoria_juros_id": c.juros,
	}, c.tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	if dados := decodificarEnvelope[dividadto.DividaDetalheResponse](t, resposta).Dados; dados.Versao != 3 ||
		dados.Descricao != "Capital de giro renovado" || dados.ContaLiberacaoID != nil || dados.SaldoInicialCentavos != 300000 {
		t.Fatalf("dívida alterada = %+v", dados)
	}
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPut, caminho, map[string]any{
		"versao": 2, "credor_tipo": "banco", "descricao": "x", "categoria_amortizacao_id": c.amortizacao, "categoria_juros_id": c.juros,
	}, c.tokenMIA), http.StatusConflict)

	competencia := c.hoje.AddDate(0, 2, 3).Format("2006-01")
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/dividas/gerar", nil, c.tokenMIA), http.StatusBadRequest)
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/dividas/gerar?competencia="+competencia+"&empresa="+c.red.String(), nil, c.tokenMIA), http.StatusForbidden)
	resposta = enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/dividas/gerar?competencia="+competencia, nil, c.tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	resultado := decodificarEnvelope[dividadto.ResultadoGeracaoResponse](t, resposta).Dados
	if resultado.Competencia != competencia || len(resultado.Geradas) != 1 || resultado.Geradas[0].Contas != 2 || len(resultado.Recusadas) != 0 {
		t.Fatalf("geração = %+v", resultado)
	}

	resposta = enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/dividas/gerar?competencia="+competencia, nil, c.tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	if deNovo := decodificarEnvelope[dividadto.ResultadoGeracaoResponse](t, resposta).Dados; len(deNovo.Geradas) != 0 {
		t.Fatalf("segunda geração = %+v", deNovo)
	}

	depois := c.obter(t, divida.ID)
	if depois.Versao != 4 || depois.Parcelas[1].Status != "gerada" || depois.Parcelas[1].ContaJuros == nil {
		t.Fatalf("dívida depois da geração = %+v", depois)
	}
	if juros := c.conta(t, depois.Parcelas[1].ContaJuros.SolicitacaoID); juros.ValorCentavos != 4100 || juros.Competencia != competencia {
		t.Errorf("juros da segunda parcela = %+v", juros)
	}
}

func TestDividaCanceladaCancelaAsContasNoContasAPagar(t *testing.T) {
	c := novoCenarioDividasE2E(t)
	paga := c.criar(t)
	if _, err := c.banco.Exec(`UPDATE solicitacao SET status = 'pago' WHERE id = $1`, paga.Parcelas[0].ContaAmortizacao.SolicitacaoID); err != nil {
		t.Fatalf("pagamento de teste falhou: %v", err)
	}
	comPagamento := c.obter(t, paga.ID)
	if comPagamento.SaldoDevedorCentavos != 200000 || comPagamento.Parcelas[0].Situacao != "a_vencer" {
		t.Fatalf("dívida com amortização paga = %+v", comPagamento)
	}
	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/dividas/"+paga.ID.String()+"/cancelar", map[string]any{
		"versao": 1, "motivo": "erro",
	}, c.tokenMIA), http.StatusConflict, "dívida com parcela paga não pode ser cancelada; renegocie o saldo ou estorne o pagamento antes")

	corpo := c.corpo()
	corpo["descricao"] = "Lançada em duplicidade"
	resposta := enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/dividas", corpo, c.tokenMIA)
	conferirStatus(t, resposta, http.StatusCreated)
	divida := decodificarEnvelope[dividadto.DividaDetalheResponse](t, resposta).Dados
	caminho := "/api/v1/dividas/" + divida.ID.String()

	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, caminho+"/cancelar", map[string]any{"versao": 1, "motivo": " "}, c.tokenMIA), http.StatusBadRequest)
	resposta = enviaComToken(t, c.servidor, http.MethodPost, caminho+"/cancelar", map[string]any{"versao": 1, "motivo": "lançada em duplicidade"}, c.tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	cancelada := decodificarEnvelope[dividadto.DividaDetalheResponse](t, resposta).Dados
	if cancelada.Status != "cancelada" || cancelada.Motivo == nil || cancelada.SaldoDevedorCentavos != 0 {
		t.Fatalf("dívida cancelada = %+v", cancelada)
	}
	for _, parcela := range cancelada.Parcelas {
		if parcela.Status != "cancelada" || parcela.Situacao != "cancelada" {
			t.Errorf("parcela %d = %+v", parcela.Numero, parcela)
		}
	}
	for _, id := range []uuid.UUID{divida.Parcelas[0].ContaAmortizacao.SolicitacaoID, divida.Parcelas[0].ContaJuros.SolicitacaoID} {
		if conta := c.conta(t, id); conta.Status != "cancelado" {
			t.Errorf("conta %s = %s", id, conta.Status)
		}
	}
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, caminho+"/cancelar", map[string]any{"versao": 2, "motivo": "de novo"}, c.tokenMIA), http.StatusConflict)

	resposta = enviaComToken(t, c.servidor, http.MethodGet, caminho+"/historico", nil, c.tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	acoes := make([]string, 0)
	for _, linha := range decodificarEnvelope[[]dividadto.HistoricoResponse](t, resposta).Dados {
		acoes = append(acoes, linha.Acao)
	}
	if len(acoes) != 3 || acoes[0] != "criada" || acoes[1] != "contas_geradas" || acoes[2] != "cancelada" {
		t.Errorf("histórico = %v", acoes)
	}
}

func TestDividaRenegociadaCancelaOAbertoECriaANova(t *testing.T) {
	c := novoCenarioDividasE2E(t)
	original := c.criar(t)
	caminho := "/api/v1/dividas/" + original.ID.String()
	primeira := original.Parcelas[0]
	if _, err := c.banco.Exec(`UPDATE solicitacao SET status = 'pago' WHERE id = $1`, primeira.ContaAmortizacao.SolicitacaoID); err != nil {
		t.Fatalf("pagamento de teste falhou: %v", err)
	}

	fiscal := criarFornecedorE2E(t, c.servidor, map[string]any{"documento": "00.394.460/0001-41", "nome": "Receita Federal"}, tokenDoTeste).ID
	nova := map[string]any{
		"fornecedor_id": fiscal, "credor_tipo": "banco", "descricao": "Renegociação Sicredi",
		"saldo_inicial_centavos": 210000, "data_saldo": c.data(-1, 0),
		"categoria_amortizacao_id": c.amortizacao, "categoria_juros_id": c.juros,
		"parcelas": []map[string]any{
			{"vencimento": c.data(0, 0), "amortizacao_centavos": 105000, "juros_centavos": 900},
			{"vencimento": c.data(0, 3), "amortizacao_centavos": 105000, "juros_centavos": 450},
		},
	}

	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, caminho+"/renegociar", map[string]any{
		"versao": 1, "motivo": "acordo", "nova": nova,
	}, c.tokenRed), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, caminho+"/renegociar", map[string]any{
		"versao": 1, "motivo": " ", "nova": nova,
	}, c.tokenMIA), http.StatusBadRequest)
	antes := map[string]any{}
	for chave, valor := range nova {
		antes[chave] = valor
	}
	antes["data_saldo"] = c.data(-5, 0)
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, caminho+"/renegociar", map[string]any{
		"versao": 1, "motivo": "acordo", "nova": antes,
	}, c.tokenMIA), http.StatusBadRequest)

	resposta := enviaComToken(t, c.servidor, http.MethodPost, caminho+"/renegociar", map[string]any{
		"versao": 1, "motivo": "acordo com o banco", "nova": nova,
	}, c.tokenMIA)
	conferirStatus(t, resposta, http.StatusCreated)
	substituta := decodificarEnvelope[dividadto.DividaDetalheResponse](t, resposta).Dados
	if substituta.ID == original.ID || substituta.DividaOrigemID == nil || *substituta.DividaOrigemID != original.ID ||
		substituta.EmpresaID != c.mia || substituta.FornecedorID != fiscal || substituta.SaldoDevedorCentavos != 210000 ||
		substituta.Parcelas[0].Status != "gerada" || substituta.Parcelas[1].Status != "prevista" {
		t.Fatalf("substituta = %+v", substituta)
	}

	depois := c.obter(t, original.ID)
	if depois.Status != "renegociada" || depois.Motivo == nil || *depois.Motivo != "acordo com o banco" || depois.SaldoDevedorCentavos != 0 {
		t.Fatalf("original = %+v", depois)
	}
	if depois.Parcelas[0].Status != "gerada" || depois.Parcelas[0].Situacao != "conta_cancelada" ||
		depois.Parcelas[1].Status != "cancelada" || depois.Parcelas[2].Status != "cancelada" {
		t.Errorf("parcelas da original = %+v", depois.Parcelas)
	}
	if conta := c.conta(t, primeira.ContaJuros.SolicitacaoID); conta.Status != "cancelado" {
		t.Errorf("juros em aberto da original = %s", conta.Status)
	}
	if conta := c.conta(t, primeira.ContaAmortizacao.SolicitacaoID); conta.Status != "pago" {
		t.Errorf("amortização paga da original = %s", conta.Status)
	}

	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, caminho+"/renegociar", map[string]any{
		"versao": 2, "motivo": "de novo", "nova": nova,
	}, c.tokenMIA), http.StatusConflict)
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPut, caminho, map[string]any{
		"versao": 2, "credor_tipo": "banco", "descricao": "x", "categoria_amortizacao_id": c.amortizacao, "categoria_juros_id": c.juros,
	}, c.tokenMIA), http.StatusConflict)

	resposta = enviaComToken(t, c.servidor, http.MethodGet, caminho+"/historico", nil, c.tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	historico := decodificarEnvelope[[]dividadto.HistoricoResponse](t, resposta).Dados
	ultimo := historico[len(historico)-1]
	if ultimo.Acao != "renegociada" || ultimo.StatusAnterior == nil || *ultimo.StatusAnterior != "ativa" || ultimo.StatusNovo != "renegociada" {
		t.Errorf("histórico da original = %+v", ultimo)
	}
}

func TestDividaReabreParcelaComContasCanceladasNoContasAPagar(t *testing.T) {
	c := novoCenarioDividasE2E(t)
	divida := c.criar(t)
	primeira := divida.Parcelas[0]
	caminho := "/api/v1/dividas/" + divida.ID.String() + "/parcelas/" + primeira.ID.String() + "/reabrir"

	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, caminho, map[string]any{"versao": 1}, c.tokenMIA), http.StatusConflict,
		"cancele ou rejeite antes as contas a pagar da parcela no contas a pagar")
	for _, id := range []uuid.UUID{primeira.ContaAmortizacao.SolicitacaoID, primeira.ContaJuros.SolicitacaoID} {
		conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/solicitacoes/"+id.String()+"/cancelar", nil, c.tokenMIA), http.StatusOK)
	}
	if situacao := c.obter(t, divida.ID).Parcelas[0].Situacao; situacao != "conta_cancelada" {
		t.Errorf("situação com as contas canceladas = %s", situacao)
	}

	resposta := enviaComToken(t, c.servidor, http.MethodPost, caminho, map[string]any{"versao": 1}, c.tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	reaberta := decodificarEnvelope[dividadto.DividaDetalheResponse](t, resposta).Dados
	if reaberta.Versao != 2 || len(reaberta.Parcelas) != 4 || reaberta.Parcelas[0].Status != "cancelada" ||
		reaberta.Parcelas[0].SubstitutaID == nil || reaberta.Parcelas[1].Status != "prevista" || reaberta.Parcelas[1].Numero != 1 ||
		reaberta.SaldoDevedorCentavos != 300000 {
		t.Fatalf("dívida reaberta = %+v", reaberta)
	}

	resposta = enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/dividas/gerar?competencia="+c.hoje.Format("2006-01"), nil, c.tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	if resultado := decodificarEnvelope[dividadto.ResultadoGeracaoResponse](t, resposta).Dados; len(resultado.Geradas) != 1 || resultado.Geradas[0].Contas != 2 {
		t.Fatalf("geração da reaberta = %+v", resultado)
	}
	if gerada := c.obter(t, divida.ID).Parcelas[1]; gerada.Status != "gerada" || gerada.ContaJuros == nil || gerada.ContaJuros.Status != "pendente_aprovacao" {
		t.Errorf("parcela reaberta depois da geração = %+v", gerada)
	}
}
