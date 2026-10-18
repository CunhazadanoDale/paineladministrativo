//go:build e2e

package http

import (
	"net/http"
	"testing"
	"time"

	clientedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/cliente"
	receberdto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/receber"
	solicitacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/solicitacao"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	"github.com/google/uuid"
)

type cenarioBorderoE2E struct {
	*cenarioTitulosE2E
	token     string
	tokenRed  string
	fomentoID uuid.UUID
	despesaID uuid.UUID
	hoje      string
}

func novoCenarioBorderoE2E(t *testing.T) *cenarioBorderoE2E {
	t.Helper()

	base := novoCenarioTitulosE2E(t)
	c := &cenarioBorderoE2E{
		cenarioTitulosE2E: base,
		token:             tokenFinanceiroDasEmpresas(t, base.servidor, "Fábio MIA", "fabio@exemplo.com", base.mia),
		tokenRed:          tokenFinanceiroDasEmpresas(t, base.servidor, "Rita Red", "rita@exemplo.com", base.red),
		hoje:              domainreceber.Hoje(time.Now().UTC()).Format("2006-01-02"),
	}
	c.fomentoID = criarFornecedorE2E(t, base.servidor, map[string]any{"documento": "45.723.174/0001-10", "nome": "Fomento Alfa"}, tokenDoTeste).ID
	raiz := criarCategoriaFinanceiraE2E(t, base.servidor, map[string]any{
		"nome": "Despesas financeiras", "natureza": "despesa", "linha_dre": "resultado_financeiro",
	}, tokenDoTeste)
	c.despesaID = criarCategoriaFinanceiraE2E(t, base.servidor, map[string]any{"nome": "Recompra de títulos", "categoria_pai_id": raiz.ID}, tokenDoTeste).ID

	return c
}

func (c *cenarioBorderoE2E) bordero(titulos ...receberdto.TituloResponse) map[string]any {
	ids := make([]uuid.UUID, 0, len(titulos))
	var bruto int64
	for _, titulo := range titulos {
		ids = append(ids, titulo.ID)
		bruto += titulo.SaldoCentavos
	}

	return map[string]any{
		"empresa_id": c.mia, "fornecedor_id": c.fomentoID, "numero": "B-77", "operado_em": c.hoje, "taxa_bp": 300,
		"valor_liquido_centavos": bruto - 6000, "conta_bancaria_id": c.contaMia, "titulo_ids": ids,
	}
}

func (c *cenarioBorderoE2E) antecipar(t *testing.T, titulos ...receberdto.TituloResponse) receberdto.AntecipacaoResponse {
	t.Helper()

	resposta := enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/antecipacoes", c.bordero(titulos...), c.token)
	conferirStatus(t, resposta, http.StatusCreated)

	return decodificarEnvelope[receberdto.AntecipacaoResponse](t, resposta).Dados
}

func (c *cenarioBorderoE2E) titulo(t *testing.T, id uuid.UUID) receberdto.TituloResponse {
	t.Helper()

	resposta := enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/titulos-receber/"+id.String(), nil, c.token)
	conferirStatus(t, resposta, http.StatusOK)

	return decodificarEnvelope[receberdto.TituloResponse](t, resposta).Dados
}

func TestFinanceiroAntecipaEEstornaUmBordero(t *testing.T) {
	c := novoCenarioBorderoE2E(t)
	primeiro := c.criar(t, c.corpo(c.mia, "1"), c.token)
	segundo := c.criar(t, c.corpo(c.mia, "2"), c.token)

	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/antecipacoes", c.bordero(primeiro), c.tokenRed), http.StatusForbidden)
	excesso := c.bordero(primeiro)
	excesso["valor_liquido_centavos"] = 150001
	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/antecipacoes", excesso, c.token),
		http.StatusBadRequest, "erro de validação: valor líquido deve ser maior que zero e no máximo o bruto (150000 centavos)")

	antecipacao := c.antecipar(t, primeiro, segundo)
	if antecipacao.ValorBrutoCentavos != 300000 || antecipacao.EncargosCentavos != 6000 || antecipacao.Status != "ativa" ||
		len(antecipacao.Titulos) != 2 || antecipacao.OperadoEm != c.hoje || *antecipacao.Numero != "B-77" {
		t.Fatalf("antecipação = %+v", antecipacao)
	}
	if titulo := c.titulo(t, primeiro.ID); titulo.Status != "antecipado" {
		t.Fatalf("título antecipado = %+v", titulo)
	}
	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/antecipacoes", c.bordero(c.titulo(t, primeiro.ID)), c.token),
		http.StatusConflict, "só título em carteira ou em banco é antecipado: 1")

	caminho := "/api/v1/antecipacoes/" + antecipacao.ID.String()
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodGet, caminho, nil, c.tokenRed), http.StatusForbidden)
	resposta := enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/antecipacoes?status=ativa&fornecedor="+c.fomentoID.String(), nil, c.token)
	conferirStatus(t, resposta, http.StatusOK)
	if lista := decodificarPagina[receberdto.AntecipacaoResponse](t, resposta).Dados; len(lista) != 1 || lista[0].ID != antecipacao.ID {
		t.Fatalf("lista = %+v", lista)
	}
	resposta = enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/antecipacoes", nil, c.tokenRed)
	conferirStatus(t, resposta, http.StatusOK)
	if lista := decodificarPagina[receberdto.AntecipacaoResponse](t, resposta).Dados; len(lista) != 0 {
		t.Fatalf("a Red não vê o borderô da MIA: %+v", lista)
	}

	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, caminho+"/estorno", map[string]any{"versao": 1}, c.token),
		http.StatusBadRequest, "erro de validação: motivo do estorno da antecipação é obrigatório")
	resposta = enviaComToken(t, c.servidor, http.MethodPost, caminho+"/estorno", map[string]any{"versao": 1, "motivo": "fomento errado"}, c.token)
	conferirStatus(t, resposta, http.StatusOK)
	if estornada := decodificarEnvelope[receberdto.AntecipacaoResponse](t, resposta).Dados; estornada.Status != "estornada" || estornada.Versao != 2 ||
		*estornada.EstornoMotivo != "fomento errado" || estornada.Titulos[0].Vigente {
		t.Fatalf("antecipação estornada = %+v", estornada)
	}
	if titulo := c.titulo(t, segundo.ID); titulo.Status != "carteira" {
		t.Fatalf("título depois do estorno = %+v", titulo)
	}
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, caminho+"/estorno", map[string]any{"versao": 2, "motivo": "de novo"}, c.token), http.StatusConflict)
	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodGet, "/api/v1/antecipacoes/"+uuid.NewString(), nil, c.token), http.StatusNotFound)
}

func TestClienteQueProibeAntecipacaoBloqueiaOBordero(t *testing.T) {
	c := novoCenarioBorderoE2E(t)
	resposta := envia(t, c.servidor, http.MethodPost, "/api/v1/clientes",
		corpoDeCliente("04.252.011/0001-10", "Águia Engenharia", "construtora", map[string]any{"proibe_antecipacao": true}))
	conferirStatus(t, resposta, http.StatusCreated)
	clienteID := decodificarEnvelope[clientedto.ClienteResponse](t, resposta).Dados.ID

	corpo := c.corpo(c.mia, "9")
	corpo["cliente_id"] = clienteID
	titulo := c.criar(t, corpo, c.token)

	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/antecipacoes", c.bordero(titulo), c.token),
		http.StatusBadRequest, "erro de validação: cliente não permite antecipação: Águia Engenharia")
}

func TestRecompraGeraContaAPagarNoM7(t *testing.T) {
	c := novoCenarioBorderoE2E(t)
	titulo := c.criar(t, c.corpo(c.mia, "1"), c.token)
	antecipacao := c.antecipar(t, titulo)
	caminho := "/api/v1/titulos-receber/" + titulo.ID.String()
	recompra := map[string]any{
		"versao": 2, "recomprado_em": c.hoje, "encargos_centavos": 4500, "vencimento": c.hoje,
		"categoria_financeira_id": c.despesaID, "motivo": "cliente não pagou",
	}

	conferirStatus(t, enviaComToken(t, c.servidor, http.MethodPost, caminho+"/recompra", recompra, c.tokenRed), http.StatusForbidden)
	comReceita := map[string]any{}
	for chave, valor := range recompra {
		comReceita[chave] = valor
	}
	comReceita["categoria_financeira_id"] = c.receitaID
	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, caminho+"/recompra", comReceita, c.token),
		http.StatusBadRequest, "erro de validação: categoria financeira da conta a pagar deve ser de despesa")

	resposta := enviaComToken(t, c.servidor, http.MethodPost, caminho+"/recompra", recompra, c.token)
	conferirStatus(t, resposta, http.StatusCreated)
	registrada := decodificarEnvelope[receberdto.RecompraRegistradaResponse](t, resposta).Dados
	if registrada.Titulo.Status != "carteira" || registrada.Recompra.ValorCentavos != 154500 || registrada.Recompra.AntecipacaoID != antecipacao.ID {
		t.Fatalf("recompra = %+v", registrada)
	}

	resposta = envia(t, c.servidor, http.MethodGet, "/api/v1/solicitacoes/"+registrada.Recompra.SolicitacaoID.String(), nil)
	conferirStatus(t, resposta, http.StatusOK)
	conta := decodificarEnvelope[solicitacaodto.SolicitacaoResponse](t, resposta).Dados
	if conta.Origem != "recompra" || *conta.OrigemID != registrada.Recompra.ID || conta.Status != "pendente_aprovacao" {
		t.Fatalf("conta a pagar da recompra = %+v", conta)
	}

	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, caminho+"/recompra", recompra, c.token),
		http.StatusConflict, "o título foi alterado por outra pessoa; recarregue e tente de novo")
	recompra["versao"] = registrada.Titulo.Versao
	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, caminho+"/recompra", recompra, c.token),
		http.StatusConflict, "só o título antecipado é recomprado")

	resposta = envia(t, c.servidor, http.MethodGet, "/api/v1/solicitacoes?escopo=todas&origem=recompra", nil)
	conferirStatus(t, resposta, http.StatusOK)
	if lista := decodificarPagina[solicitacaodto.SolicitacaoResponse](t, resposta).Dados; len(lista) != 1 {
		t.Fatalf("contas de recompra = %d", len(lista))
	}
}

func TestPagamentoNaContaErradaGeraRepasseAoFomento(t *testing.T) {
	c := novoCenarioBorderoE2E(t)
	titulo := c.criar(t, c.corpo(c.mia, "1"), c.token)
	c.antecipar(t, titulo)
	caminho := "/api/v1/titulos-receber/" + titulo.ID.String()
	baixa := map[string]any{"versao": 2, "conta_bancaria_id": c.contaMia, "recebido_em": c.hoje, "valor_centavos": 150000}

	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, caminho+"/recebimentos", baixa, c.token), http.StatusBadRequest,
		"erro de validação: título antecipado recebido em conta própria exige o repasse ao fomento (vencimento e categoria)")

	baixa["repasse"] = map[string]any{"vencimento": c.hoje, "categoria_financeira_id": c.despesaID}
	resposta := enviaComToken(t, c.servidor, http.MethodPost, caminho+"/recebimentos", baixa, c.token)
	conferirStatus(t, resposta, http.StatusCreated)
	registrada := decodificarEnvelope[receberdto.RecebimentoRegistradoResponse](t, resposta).Dados
	repasse := registrada.Recebimento.Repasse
	if registrada.Titulo.Status != "recebido" || repasse == nil || repasse.ValorCentavos != 150000 || repasse.SolicitacaoID == uuid.Nil {
		t.Fatalf("baixa com repasse = %+v", registrada)
	}

	resposta = envia(t, c.servidor, http.MethodGet, "/api/v1/solicitacoes/"+repasse.SolicitacaoID.String(), nil)
	conferirStatus(t, resposta, http.StatusOK)
	if conta := decodificarEnvelope[solicitacaodto.SolicitacaoResponse](t, resposta).Dados; conta.Origem != "repasse_fomento" || *conta.OrigemID != repasse.ID {
		t.Fatalf("conta do repasse = %+v", conta)
	}

	resposta = enviaComToken(t, c.servidor, http.MethodGet, caminho+"/recebimentos", nil, c.token)
	conferirStatus(t, resposta, http.StatusOK)
	recebimentos := decodificarEnvelope[[]receberdto.RecebimentoResponse](t, resposta).Dados
	if len(recebimentos) != 1 || recebimentos[0].Repasse == nil || recebimentos[0].Repasse.SolicitacaoID != repasse.SolicitacaoID {
		t.Fatalf("recebimentos = %+v", recebimentos)
	}
	conferirMensagem(t, enviaComToken(t, c.servidor, http.MethodPost, caminho+"/recebimentos/"+recebimentos[0].ID.String()+"/estorno",
		map[string]any{"versao": 3, "motivo": "x", "estornado_em": c.hoje}, c.token),
		http.StatusConflict, "o recebimento gerou repasse ao fomento e não pode ser estornado; trate a conta a pagar do repasse")
}

func TestLiquidacaoNoFomentoBaixaSemConta(t *testing.T) {
	c := novoCenarioBorderoE2E(t)
	titulo := c.criar(t, c.corpo(c.mia, "1"), c.token)
	c.antecipar(t, titulo)

	resposta := enviaComToken(t, c.servidor, http.MethodPost, "/api/v1/titulos-receber/"+titulo.ID.String()+"/recebimentos", map[string]any{
		"versao": 2, "tipo": "fomento", "recebido_em": c.hoje, "valor_centavos": 150000,
	}, c.token)
	conferirStatus(t, resposta, http.StatusCreated)
	registrada := decodificarEnvelope[receberdto.RecebimentoRegistradoResponse](t, resposta).Dados
	if registrada.Titulo.Status != "recebido" || registrada.Recebimento.Tipo != "fomento" || registrada.Recebimento.EntradaNoCaixaCentavos != 0 ||
		registrada.Recebimento.ContaBancariaID != nil || registrada.Recebimento.Repasse != nil {
		t.Fatalf("liquidação no fomento = %+v", registrada)
	}
}
