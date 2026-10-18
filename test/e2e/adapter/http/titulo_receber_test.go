//go:build e2e

package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	pedidodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/pedido"
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

func TestFinanceiroGeraOTituloDeEntradaDoPedido(t *testing.T) {
	c := novoCenarioTitulosE2E(t)
	tokenFinanceiro := tokenFinanceiroDasEmpresas(t, c.servidor, "Fábio MIA", "fabio@exemplo.com", c.mia)
	tokenRed := tokenFinanceiroDasEmpresas(t, c.servidor, "Rita Red", "rita@exemplo.com", c.red)
	produtoID := produtoDoTeste(t, c.servidor)

	pedido := criarPedidoE2E(t, c.servidor, corpoDePedido(c.mia, c.clienteID, produtoID, "importacao", 7), tokenDoTeste)
	caminhoDoPedido := "/api/v1/pedidos/" + pedido.ID.String()
	vencimento := domainreceber.Hoje(time.Now().UTC()).AddDate(0, 0, 7).Format("2006-01-02")
	corpo := map[string]any{"vencimento": vencimento, "conta_bancaria_id": c.contaMia}

	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, caminhoDoPedido+"/titulo-entrada", corpo, tokenFinanceiro),
		http.StatusConflict, "o título de entrada só é gerado para pedido confirmado e não entregue")
	conferirStatus(t, envia(t, c.servidor, http.MethodPost, caminhoDoPedido+"/confirmar", map[string]any{"versao": 1}), http.StatusOK)
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, caminhoDoPedido+"/titulo-entrada", corpo, tokenRed), http.StatusForbidden)

	resposta := enviaComToken(t, c.servidor, http.MethodPost, caminhoDoPedido+"/titulo-entrada", corpo, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusCreated)
	titulo := decodificarEnvelope[receberdto.TituloResponse](t, resposta).Dados
	if titulo.Origem != "pedido" || *titulo.OrigemID != pedido.ID || titulo.Numero != "PED-1" || titulo.ValorCentavos != 3150 ||
		titulo.Status != "em_banco" || titulo.Vencimento != vencimento || titulo.CategoriaFinanceiraID != nil {
		t.Fatalf("título de entrada = %+v", titulo)
	}

	resposta = enviaComToken(t, c.servidor, http.MethodPost, caminhoDoPedido+"/titulo-entrada", map[string]any{"vencimento": "2027-01-01"}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if repetido := decodificarEnvelope[receberdto.TituloResponse](t, resposta).Dados; repetido.ID != titulo.ID || repetido.Vencimento != vencimento {
		t.Fatalf("repetir devolve o mesmo título = %+v", repetido)
	}

	resposta = enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/titulos-receber?origem=pedido&origem_id="+pedido.ID.String(), nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if lista := decodificarPagina[receberdto.TituloResponse](t, resposta).Dados; len(lista) != 1 || lista[0].ID != titulo.ID {
		t.Fatalf("títulos do pedido = %+v", lista)
	}

	resposta = enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/titulos-receber/"+titulo.ID.String()+"/historico", nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if historico := decodificarEnvelope[[]receberdto.HistoricoResponse](t, resposta).Dados; len(historico) != 1 || historico[0].Evento != "gerado" {
		t.Fatalf("histórico = %+v", historico)
	}

	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPut, "/api/v1/titulos-receber/"+titulo.ID.String(), map[string]any{
		"numero": "X", "parcela": 1, "vencimento": vencimento, "valor_centavos": 1, "versao": 1,
	}, tokenFinanceiro), http.StatusConflict, "só o título manual pode ser editado; o gerado muda pela prorrogação e pela conta de cobrança")

	prontaEntrega := criarPedidoE2E(t, c.servidor, corpoDePedido(c.mia, c.clienteID, produtoID, "pronta_entrega", 2), tokenDoTeste)
	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/pedidos/"+prontaEntrega.ID.String()+"/titulo-entrada", corpo, tokenFinanceiro),
		http.StatusBadRequest, "erro de validação: só o pedido de importação tem título de entrada")

	cancelado := criarPedidoE2E(t, c.servidor, corpoDePedido(c.mia, c.clienteID, produtoID, "importacao", 2), tokenDoTeste)
	caminhoCancelado := "/api/v1/pedidos/" + cancelado.ID.String()
	conferirStatus(t, envia(t, c.servidor, http.MethodPost, caminhoCancelado+"/cancelar", map[string]any{"versao": 1, "motivo": "desistiu"}), http.StatusOK)
	resposta = envia(t, c.servidor, http.MethodGet, caminhoCancelado, nil)
	if detalhe := decodificarEnvelope[pedidodto.PedidoDetalheResponse](t, resposta).Dados; detalhe.Status != "cancelado" {
		t.Fatalf("pedido = %+v", detalhe)
	}
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, caminhoCancelado+"/titulo-entrada", corpo, tokenFinanceiro), http.StatusConflict)
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/pedidos/"+uuid.NewString()+"/titulo-entrada", corpo, tokenFinanceiro), http.StatusNotFound)
}

func TestFinanceiroRecebeEEstornaTitulos(t *testing.T) {
	c := novoCenarioTitulosE2E(t)
	tokenFinanceiro := tokenFinanceiroDasEmpresas(t, c.servidor, "Fábio MIA", "fabio@exemplo.com", c.mia)
	tokenRed := tokenFinanceiroDasEmpresas(t, c.servidor, "Rita Red", "rita@exemplo.com", c.red)
	hoje := domainreceber.Hoje(time.Now().UTC()).Format("2006-01-02")

	titulo := c.criar(t, c.corpo(c.mia, "ALG-10"), tokenFinanceiro)
	caminho := "/api/v1/titulos-receber/" + titulo.ID.String()
	baixa := func(versao int, valor, juros, desconto int64) map[string]any {
		return map[string]any{
			"versao": versao, "conta_bancaria_id": c.contaMia, "recebido_em": hoje,
			"valor_centavos": valor, "juros_centavos": juros, "desconto_centavos": desconto,
		}
	}

	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, caminho+"/recebimentos", baixa(1, 1000, 0, 0), tokenRed), http.StatusForbidden)
	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, caminho+"/recebimentos", baixa(1, 150000, 0, 1), tokenFinanceiro),
		http.StatusBadRequest, "erro de validação: valor mais desconto passa do saldo do título (150000 centavos)")
	semConta := baixa(1, 1000, 0, 0)
	delete(semConta, "conta_bancaria_id")
	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, caminho+"/recebimentos", semConta, tokenFinanceiro),
		http.StatusBadRequest, "erro de validação: conta bancária do recebimento é obrigatória")

	resposta := enviaComToken(t, c.servidor, http.MethodPost, caminho+"/recebimentos", baixa(1, 50000, 0, 0), tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusCreated)
	parcial := decodificarEnvelope[receberdto.RecebimentoRegistradoResponse](t, resposta).Dados
	if parcial.Titulo.Status != "carteira" || parcial.Titulo.SaldoCentavos != 100000 || parcial.Titulo.Versao != 2 || parcial.Recebimento.Tipo != "conta" {
		t.Fatalf("baixa parcial = %+v", parcial)
	}

	resposta = enviaComToken(t, c.servidor, http.MethodPost, caminho+"/recebimentos", baixa(2, 95000, 800, 5000), tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusCreated)
	total := decodificarEnvelope[receberdto.RecebimentoRegistradoResponse](t, resposta).Dados
	if total.Titulo.Status != "recebido" || total.Titulo.SaldoCentavos != 0 || total.Recebimento.EntradaNoCaixaCentavos != 95800 {
		t.Fatalf("baixa total = %+v", total)
	}
	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, caminho+"/recebimentos", baixa(3, 1, 0, 0), tokenFinanceiro),
		http.StatusConflict, "só o título em carteira, em banco ou antecipado recebe baixa")
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, caminho+"/cancelar", map[string]any{"versao": 3, "motivo": "x"}, tokenFinanceiro), http.StatusConflict)

	caminhoDoEstorno := caminho + "/recebimentos/" + total.Recebimento.ID.String() + "/estorno"
	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, caminhoDoEstorno, map[string]any{"versao": 3, "estornado_em": hoje}, tokenFinanceiro),
		http.StatusBadRequest, "erro de validação: motivo do estorno é obrigatório")
	resposta = enviaComToken(t, c.servidor, http.MethodPost, caminhoDoEstorno, map[string]any{"versao": 3, "motivo": "pix devolvido", "estornado_em": hoje}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if estornado := decodificarEnvelope[receberdto.TituloResponse](t, resposta).Dados; estornado.Status != "carteira" || estornado.SaldoCentavos != 100000 || estornado.Versao != 4 {
		t.Fatalf("título estornado = %+v", estornado)
	}
	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, caminhoDoEstorno, map[string]any{"versao": 4, "motivo": "de novo", "estornado_em": hoje}, tokenFinanceiro),
		http.StatusConflict, "o recebimento já foi estornado")
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, caminho+"/recebimentos/"+uuid.NewString()+"/estorno",
		map[string]any{"versao": 4, "motivo": "x", "estornado_em": hoje}, tokenFinanceiro), http.StatusNotFound)

	resposta = enviaComToken(t, c.servidor, http.MethodGet, caminho+"/recebimentos", nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	recebimentos := decodificarEnvelope[[]receberdto.RecebimentoResponse](t, resposta).Dados
	if len(recebimentos) != 2 || recebimentos[0].Estornado || !recebimentos[1].Estornado || recebimentos[1].Estorno == nil ||
		recebimentos[1].Estorno.Motivo != "pix devolvido" || recebimentos[1].Estorno.EstornadoEm != hoje {
		t.Fatalf("recebimentos = %+v", recebimentos)
	}
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodGet, caminho+"/recebimentos", nil, tokenRed), http.StatusForbidden)
}

func TestBaixaDoTituloDeEntradaRegistraAEntradaNoPedido(t *testing.T) {
	c := novoCenarioTitulosE2E(t)
	tokenFinanceiro := tokenFinanceiroDasEmpresas(t, c.servidor, "Fábio MIA", "fabio@exemplo.com", c.mia)
	produtoID := produtoDoTeste(t, c.servidor)
	hoje := domainreceber.Hoje(time.Now().UTC()).Format("2006-01-02")

	pedido := criarPedidoE2E(t, c.servidor, corpoDePedido(c.mia, c.clienteID, produtoID, "importacao", 10), tokenDoTeste)
	caminhoDoPedido := "/api/v1/pedidos/" + pedido.ID.String()
	conferirStatus(t, envia(t, c.servidor, http.MethodPost, caminhoDoPedido+"/confirmar", map[string]any{"versao": 1}), http.StatusOK)
	resposta := enviaComToken(t, c.servidor, http.MethodPost, caminhoDoPedido+"/titulo-entrada", map[string]any{"vencimento": hoje}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusCreated)
	titulo := decodificarEnvelope[receberdto.TituloResponse](t, resposta).Dados
	caminho := "/api/v1/titulos-receber/" + titulo.ID.String()

	resposta = enviaComToken(t, c.servidor, http.MethodPost, caminho+"/recebimentos", map[string]any{
		"versao": 1, "conta_bancaria_id": c.contaMia, "recebido_em": hoje, "valor_centavos": titulo.ValorCentavos,
	}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusCreated)
	baixa := decodificarEnvelope[receberdto.RecebimentoRegistradoResponse](t, resposta).Dados
	if !baixa.Recebimento.RegistrouEntradaPedido || baixa.Titulo.Status != "recebido" {
		t.Fatalf("baixa = %+v", baixa)
	}

	resposta = envia(t, c.servidor, http.MethodGet, caminhoDoPedido, nil)
	conferirStatus(t, resposta, http.StatusOK)
	detalhe := decodificarEnvelope[pedidodto.PedidoDetalheResponse](t, resposta).Dados
	if detalhe.EntradaPagaEm == nil || detalhe.EntradaPagaEm.Format("2006-01-02") != hoje || detalhe.EntradaPagaCentavos == nil ||
		*detalhe.EntradaPagaCentavos != 4500 || detalhe.PrazoLimite == nil || detalhe.Versao != 3 {
		t.Fatalf("pedido depois da baixa = %+v", detalhe)
	}

	resposta = envia(t, c.servidor, http.MethodGet, caminhoDoPedido+"/historico", nil)
	conferirStatus(t, resposta, http.StatusOK)
	historico := decodificarEnvelope[[]pedidodto.HistoricoResponse](t, resposta).Dados
	if ultimo := historico[len(historico)-1]; ultimo.Acao != "entrada_registrada" || ultimo.Detalhe != "pela baixa do título PED-1" {
		t.Fatalf("histórico do pedido = %+v", historico)
	}

	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, caminho+"/recebimentos/"+baixa.Recebimento.ID.String()+"/estorno",
		map[string]any{"versao": 2, "motivo": "x", "estornado_em": hoje}, tokenFinanceiro),
		http.StatusConflict, "o recebimento registrou a entrada paga do pedido e não pode ser estornado")
}
