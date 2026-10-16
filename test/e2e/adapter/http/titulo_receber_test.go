//go:build e2e

package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	receberdto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/receber"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	"github.com/google/uuid"
)

type cenarioTitulosE2E struct {
	servidor  *httptest.Server
	mia       uuid.UUID
	red       uuid.UUID
	clienteID uuid.UUID
	receitaID uuid.UUID
	contaMia  uuid.UUID
	contaRed  uuid.UUID
}

func novoCenarioTitulosE2E(t *testing.T) *cenarioTitulosE2E {
	t.Helper()

	servidor, _ := servidorDoTeste(t)
	c := &cenarioTitulosE2E{
		servidor:  servidor,
		mia:       criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID,
		red:       criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond").ID,
		clienteID: clienteDoPedido(t, servidor),
	}

	raiz := criarCategoriaFinanceiraE2E(t, servidor, map[string]any{
		"nome": "Outras receitas", "natureza": "receita", "linha_dre": "receita_operacional",
	}, tokenDoTeste)
	c.receitaID = criarCategoriaFinanceiraE2E(t, servidor, map[string]any{"nome": "Aluguel recebido", "categoria_pai_id": raiz.ID}, tokenDoTeste).ID
	c.contaMia = criarContaE2E(t, servidor, map[string]any{
		"empresa_id": c.mia, "banco": "748", "tipo": "corrente", "agencia": "0812", "conta": "54321-0", "apelido": "Sicredi cobrança",
	}, tokenDoTeste).ID
	c.contaRed = criarContaE2E(t, servidor, map[string]any{
		"empresa_id": c.red, "banco": "341", "tipo": "corrente", "agencia": "1234", "conta": "98765-4", "apelido": "Itaú Red",
	}, tokenDoTeste).ID

	return c
}

func (c *cenarioTitulosE2E) corpo(empresaID uuid.UUID, numero string) map[string]any {
	return map[string]any{
		"empresa_id":              empresaID,
		"cliente_id":              c.clienteID,
		"numero":                  numero,
		"parcela":                 1,
		"vencimento":              domainreceber.Hoje(time.Now().UTC()).AddDate(0, 1, 0).Format("2006-01-02"),
		"valor_centavos":          150000,
		"categoria_financeira_id": c.receitaID,
		"observacao":              "aluguel do galpão",
	}
}

func (c *cenarioTitulosE2E) criar(t *testing.T, corpo map[string]any, token string) receberdto.TituloResponse {
	t.Helper()

	resposta := enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/titulos-receber", corpo, token)
	conferirStatus(t, resposta, http.StatusCreated)

	return decodificarEnvelope[receberdto.TituloResponse](t, resposta).Dados
}

func TestFinanceiroOperaTitulosAReceber(t *testing.T) {
	c := novoCenarioTitulosE2E(t)
	tokenFinanceiro := tokenFinanceiroDasEmpresas(t, c.servidor, "Fábio MIA", "fabio@exemplo.com", c.mia)

	titulo := c.criar(t, c.corpo(c.mia, "ALG-10"), tokenFinanceiro)
	if titulo.Status != "carteira" || titulo.Origem != "manual" || titulo.SaldoCentavos != 150000 || titulo.Vencido || titulo.Versao != 1 {
		t.Fatalf("título criado = %+v", titulo)
	}
	caminho := "/api/v1/titulos-receber/" + titulo.ID.String()

	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/titulos-receber", c.corpo(c.mia, "alg-10"), tokenFinanceiro),
		http.StatusConflict, "já existe um título com este número e parcela na empresa")
	semCategoria := c.corpo(c.mia, "ALG-11")
	delete(semCategoria, "categoria_financeira_id")
	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/titulos-receber", semCategoria, tokenFinanceiro),
		http.StatusBadRequest, "erro de validação: categoria financeira do título manual é obrigatória")

	edicao := c.corpo(c.mia, "ALG-10")
	delete(edicao, "empresa_id")
	delete(edicao, "cliente_id")
	edicao["valor_centavos"] = 180000
	edicao["versao"] = 1
	resposta := enviaComToken(t, c.servidor, http.MethodPut, caminho, edicao, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if editado := decodificarEnvelope[receberdto.TituloResponse](t, resposta).Dados; editado.ValorCentavos != 180000 || editado.Versao != 2 {
		t.Fatalf("título editado = %+v", editado)
	}
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPut, caminho, edicao, tokenFinanceiro), http.StatusConflict)

	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, caminho+"/enviar-banco", map[string]any{
		"versao": 2, "conta_bancaria_id": c.contaRed,
	}, tokenFinanceiro), http.StatusBadRequest, "erro de validação: conta bancária deve ser da mesma empresa do título")
	resposta = enviaComToken(t, c.servidor, http.MethodPost, caminho+"/enviar-banco", map[string]any{
		"versao": 2, "conta_bancaria_id": c.contaMia,
	}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if emBanco := decodificarEnvelope[receberdto.TituloResponse](t, resposta).Dados; emBanco.Status != "em_banco" || *emBanco.ContaBancariaID != c.contaMia {
		t.Fatalf("título em banco = %+v", emBanco)
	}

	enviadoEm := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	resposta = enviaComToken(t, c.servidor, http.MethodPost, caminho+"/boleto-enviado", map[string]any{
		"versao": 3, "canal": "whatsapp", "enviado_em": enviadoEm,
	}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if boleto := decodificarEnvelope[receberdto.TituloResponse](t, resposta).Dados; boleto.BoletoCanal == nil || *boleto.BoletoCanal != "whatsapp" || boleto.BoletoEnviadoEm == nil {
		t.Fatalf("boleto = %+v", boleto)
	}

	novoVencimento := domainreceber.Hoje(time.Now().UTC()).AddDate(0, 2, 0).Format("2006-01-02")
	resposta = enviaComToken(t, c.servidor, http.MethodPost, caminho+"/prorrogar", map[string]any{
		"versao": 4, "vencimento": novoVencimento, "motivo": "cliente pediu",
	}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if prorrogado := decodificarEnvelope[receberdto.TituloResponse](t, resposta).Dados; prorrogado.Vencimento != novoVencimento || prorrogado.Status != "em_banco" {
		t.Fatalf("título prorrogado = %+v", prorrogado)
	}

	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, caminho+"/retirar-banco", map[string]any{"versao": 5}, tokenFinanceiro), http.StatusOK)
	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, caminho+"/cancelar", map[string]any{"versao": 6}, tokenFinanceiro),
		http.StatusBadRequest, "erro de validação: motivo do cancelamento é obrigatório")
	resposta = enviaComToken(t, c.servidor, http.MethodPost, caminho+"/cancelar", map[string]any{"versao": 6, "motivo": "lançado errado"}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if cancelado := decodificarEnvelope[receberdto.TituloResponse](t, resposta).Dados; cancelado.Status != "cancelado" || *cancelado.CanceladoMotivo != "lançado errado" {
		t.Fatalf("título cancelado = %+v", cancelado)
	}

	resposta = enviaComToken(t, c.servidor, http.MethodGet, caminho+"/historico", nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	historico := decodificarEnvelope[[]receberdto.HistoricoResponse](t, resposta).Dados
	eventos := make([]string, 0, len(historico))
	for _, item := range historico {
		eventos = append(eventos, item.Evento)
	}
	esperados := []string{"criado", "editado", "enviado_banco", "boleto_enviado", "prorrogado", "retirado_banco", "cancelado"}
	if len(eventos) != len(esperados) {
		t.Fatalf("eventos = %v", eventos)
	}
	for i := range esperados {
		if eventos[i] != esperados[i] {
			t.Fatalf("eventos = %v; esperados %v", eventos, esperados)
		}
	}

	c.criar(t, c.corpo(c.mia, "ALG-10"), tokenFinanceiro)
}

func TestTitulosAReceberIsoladosPorEmpresaEPerfil(t *testing.T) {
	c := novoCenarioTitulosE2E(t)
	tokenMia := tokenFinanceiroDasEmpresas(t, c.servidor, "Fábio MIA", "fabio@exemplo.com", c.mia)
	tokenRed := tokenFinanceiroDasEmpresas(t, c.servidor, "Rita Red", "rita@exemplo.com", c.red)
	tokenComercial := usuarioComercialDasEmpresas(t, c.servidor, "Carla", "carla@exemplo.com", c.mia)

	daMia := c.criar(t, c.corpo(c.mia, "1"), tokenMia)
	c.criar(t, c.corpo(c.red, "1"), tokenRed)
	caminho := "/api/v1/titulos-receber/" + daMia.ID.String()

	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/titulos-receber", c.corpo(c.red, "2"), tokenMia),
		http.StatusForbidden, "perfil sem acesso a esta empresa")
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodGet, caminho, nil, tokenRed), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, caminho+"/cancelar", map[string]any{"versao": 1, "motivo": "x"}, tokenRed), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/titulos-receber?empresa="+c.mia.String(), nil, tokenRed), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/titulos-receber", nil, tokenComercial), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodGet, caminho, nil, tokenComercial), http.StatusForbidden)
	conferirStatus(t, enviaSemToken(t, c.servidor, http.MethodGet, "/api/v1/titulos-receber", nil), http.StatusUnauthorized)
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/titulos-receber/"+uuid.NewString(), nil, tokenMia), http.StatusNotFound)

	resposta := enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/titulos-receber", nil, tokenRed)
	conferirStatus(t, resposta, http.StatusOK)
	if lista := decodificarPagina[receberdto.TituloResponse](t, resposta).Dados; len(lista) != 1 || lista[0].EmpresaID != c.red {
		t.Fatalf("lista da Red = %+v", lista)
	}

	resposta = envia(t, c.servidor, http.MethodGet, "/api/v1/titulos-receber?status=carteira&origem=manual&cliente="+c.clienteID.String(), nil)
	conferirStatus(t, resposta, http.StatusOK)
	if lista := decodificarPagina[receberdto.TituloResponse](t, resposta).Dados; len(lista) != 2 {
		t.Fatalf("lista do administrador = %d títulos", len(lista))
	}

	conferirStatus(t, envia(t, c.servidor, http.MethodGet, "/api/v1/titulos-receber?status=vencido", nil), http.StatusBadRequest)
	resposta = envia(t, c.servidor, http.MethodGet, "/api/v1/titulos-receber?vencidos=true", nil)
	conferirStatus(t, resposta, http.StatusOK)
	if lista := decodificarPagina[receberdto.TituloResponse](t, resposta).Dados; len(lista) != 0 {
		t.Fatalf("nenhum título está vencido: %+v", lista)
	}
}
