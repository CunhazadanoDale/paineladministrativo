//go:build e2e

package http

import (
	"net/http"
	"testing"
	"time"

	importacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/importacao"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	"github.com/google/uuid"
)

func corpoDeImportacao(empresaID, fornecedorID, produtoID uuid.UUID) map[string]any {
	return map[string]any{
		"empresa_id":    empresaID,
		"fornecedor_id": fornecedorID,
		"referencia":    "PI-2026-001",
		"itens":         []map[string]any{{"produto_id": produtoID, "quantidade": 1000, "valor_moeda_centesimos": 450000}},
	}
}

func TestFluxoDaImportacaoAteODesembaraco(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	red := criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond").ID
	tokenFinanceiro := tokenFinanceiroDasEmpresas(t, servidor, "Fábio MIA", "fabio@exemplo.com", mia)
	tokenFinanceiroRed := tokenFinanceiroDasEmpresas(t, servidor, "Rita Red", "rita@exemplo.com", red)
	tokenComercial := usuarioComercialDasEmpresas(t, servidor, "Carla MIA", "carla@exemplo.com", mia)
	fornecedor := criarFornecedorE2E(t, servidor, map[string]any{
		"pais": "CN", "identificacao_estrangeira": "91440600MA4UX", "nome": "Foshan Ceramics", "moeda_padrao": "USD",
	}, tokenDoTeste)
	produtoID := produtoDoTeste(t, servidor)

	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/importacoes", corpoDeImportacao(mia, fornecedor.ID, produtoID), tokenComercial), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/importacoes", corpoDeImportacao(red, fornecedor.ID, produtoID), tokenFinanceiro), http.StatusForbidden)
	semItens := corpoDeImportacao(mia, fornecedor.ID, produtoID)
	delete(semItens, "itens")
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/importacoes", semItens, tokenFinanceiro),
		http.StatusBadRequest, "erro de validação: a importação precisa de pelo menos um item")

	resposta := enviaComToken(t, servidor, http.MethodPost, "/api/v1/importacoes", corpoDeImportacao(mia, fornecedor.ID, produtoID), tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusCreated)
	criada := decodificarEnvelope[importacaodto.ImportacaoDetalheResponse](t, resposta).Dados
	if criada.Numero != 1 || criada.Status != "aberta" || criada.Moeda != "USD" || criada.Versao != 1 ||
		criada.ValorTotalMoedaCentesimos == nil || *criada.ValorTotalMoedaCentesimos != 450000 || len(criada.Itens) != 1 {
		t.Fatalf("importação criada = %+v", criada)
	}
	caminho := "/api/v1/importacoes/" + criada.ID.String()

	resposta = enviaComToken(t, servidor, http.MethodGet, caminho, nil, tokenComercial)
	conferirStatus(t, resposta, http.StatusOK)
	if vista := decodificarEnvelope[importacaodto.ImportacaoDetalheResponse](t, resposta).Dados; vista.ValorTotalMoedaCentesimos != nil ||
		vista.Itens[0].ValorMoedaCentesimos != nil || vista.Itens[0].Quantidade != 1000 {
		t.Errorf("visão do comercial = %+v", vista)
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, caminho, nil, tokenFinanceiroRed), http.StatusForbidden)

	alteracao := corpoDeImportacao(mia, fornecedor.ID, produtoID)
	delete(alteracao, "empresa_id")
	alteracao["versao"] = 1
	alteracao["moeda"] = "cny"
	alteracao["itens"] = []map[string]any{{"produto_id": produtoID, "quantidade": 1200, "valor_moeda_centesimos": 540000}}
	resposta = enviaComToken(t, servidor, http.MethodPut, caminho, alteracao, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	alterada := decodificarEnvelope[importacaodto.ImportacaoDetalheResponse](t, resposta).Dados
	if alterada.Versao != 2 || alterada.Moeda != "CNY" || alterada.Itens[0].ID != criada.Itens[0].ID || alterada.Itens[0].Quantidade != 1200 {
		t.Errorf("importação alterada = %+v", alterada)
	}
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPut, caminho, alteracao, tokenFinanceiro),
		http.StatusConflict, "a importação foi alterada por outra pessoa; recarregue e tente de novo")

	hoje := domainimportacao.Hoje(time.Now())
	data := func(dias int) string { return hoje.AddDate(0, 0, dias).Format("2006-01-02") }
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/embarcar", map[string]any{
		"versao": 2, "embarcada_em": data(1), "chegada_prevista": data(40),
	}, tokenFinanceiro), http.StatusBadRequest)
	resposta = enviaComToken(t, servidor, http.MethodPost, caminho+"/embarcar", map[string]any{
		"versao": 2, "embarcada_em": data(-10), "chegada_prevista": data(30), "documento_embarque": "BL-778",
	}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if embarcada := decodificarEnvelope[importacaodto.ImportacaoDetalheResponse](t, resposta).Dados; embarcada.Status != "embarcada" ||
		embarcada.ChegadaPrevista == nil || *embarcada.ChegadaPrevista != data(30) || embarcada.Versao != 3 {
		t.Errorf("importação embarcada = %+v", embarcada)
	}

	trocaDeFornecedor := criarFornecedorE2E(t, servidor, map[string]any{
		"pais": "IT", "identificacao_estrangeira": "IT0123456789", "nome": "Sassuolo Tiles", "moeda_padrao": "EUR",
	}, tokenDoTeste)
	alteracao["versao"] = 3
	alteracao["fornecedor_id"] = trocaDeFornecedor.ID
	alteracao["moeda"] = ""
	alteracao["chegada_prevista"] = data(35)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPut, caminho, alteracao, tokenFinanceiro), http.StatusConflict)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/importacoes?status=embarcada&chegada_ate="+data(30), nil, tokenComercial)
	conferirStatus(t, resposta, http.StatusOK)
	if pagina := decodificarPagina[importacaodto.ImportacaoResponse](t, resposta); len(pagina.Dados) != 1 || pagina.Dados[0].ID != criada.ID ||
		pagina.Dados[0].ValorTotalMoedaCentesimos != nil {
		t.Errorf("lista do comercial = %+v", pagina)
	}
	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/importacoes", nil, tokenFinanceiroRed)
	conferirStatus(t, resposta, http.StatusOK)
	if pagina := decodificarPagina[importacaodto.ImportacaoResponse](t, resposta); len(pagina.Dados) != 0 {
		t.Errorf("lista da Red = %+v", pagina)
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/importacoes?empresa="+mia.String(), nil, tokenFinanceiroRed), http.StatusForbidden)

	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/desembaracar", map[string]any{
		"versao": 3, "chegada_em": data(-2), "desembaracada_em": data(0),
	}, tokenFinanceiro), http.StatusBadRequest, "erro de validação: número da declaração de importação é obrigatório")
	resposta = enviaComToken(t, servidor, http.MethodPost, caminho+"/desembaracar", map[string]any{
		"versao": 3, "chegada_em": data(-2), "desembaracada_em": data(0), "declaracao": "26/0001234-5",
	}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if desembaracada := decodificarEnvelope[importacaodto.ImportacaoDetalheResponse](t, resposta).Dados; desembaracada.Status != "desembaracada" ||
		desembaracada.Declaracao == nil || *desembaracada.Declaracao != "26/0001234-5" {
		t.Errorf("importação desembaraçada = %+v", desembaracada)
	}
	alteracao["fornecedor_id"] = fornecedor.ID
	alteracao["versao"] = 4
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPut, caminho, alteracao, tokenFinanceiro), http.StatusConflict)

	resposta = enviaComToken(t, servidor, http.MethodGet, caminho+"/historico", nil, tokenComercial)
	conferirStatus(t, resposta, http.StatusOK)
	historico := decodificarEnvelope[[]importacaodto.HistoricoResponse](t, resposta).Dados
	acoes := make([]string, 0, len(historico))
	for _, linha := range historico {
		acoes = append(acoes, linha.Acao)
	}
	if len(acoes) != 4 || acoes[0] != "criada" || acoes[1] != "alterada" || acoes[2] != "embarcada" || acoes[3] != "desembaracada" {
		t.Errorf("histórico = %v", acoes)
	}
}

func TestCancelamentoDaImportacao(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	fornecedor := criarFornecedorE2E(t, servidor, map[string]any{
		"pais": "CN", "identificacao_estrangeira": "91440600MA4UX", "nome": "Foshan Ceramics", "moeda_padrao": "CNY",
	}, tokenDoTeste)
	produtoID := produtoDoTeste(t, servidor)

	resposta := envia(t, servidor, http.MethodPost, "/api/v1/importacoes", corpoDeImportacao(mia, fornecedor.ID, produtoID))
	conferirStatus(t, resposta, http.StatusCreated)
	criada := decodificarEnvelope[importacaodto.ImportacaoDetalheResponse](t, resposta).Dados
	if criada.Moeda != "CNY" {
		t.Errorf("moeda padrão do fornecedor = %q", criada.Moeda)
	}
	caminho := "/api/v1/importacoes/" + criada.ID.String()

	conferirMensagem(t, envia(t, servidor, http.MethodPost, caminho+"/cancelar", map[string]any{"versao": 1, "motivo": " "}),
		http.StatusBadRequest, "erro de validação: motivo do cancelamento é obrigatório")
	resposta = envia(t, servidor, http.MethodPost, caminho+"/cancelar", map[string]any{"versao": 1, "motivo": "fornecedor desistiu"})
	conferirStatus(t, resposta, http.StatusOK)
	if cancelada := decodificarEnvelope[importacaodto.ImportacaoDetalheResponse](t, resposta).Dados; cancelada.Status != "cancelada" ||
		cancelada.CanceladaMotivo == nil || *cancelada.CanceladaMotivo != "fornecedor desistiu" {
		t.Errorf("importação cancelada = %+v", cancelada)
	}
	conferirStatus(t, envia(t, servidor, http.MethodPost, caminho+"/cancelar", map[string]any{"versao": 2, "motivo": "de novo"}), http.StatusConflict)

	conferirStatus(t, envia(t, servidor, http.MethodGet, "/api/v1/importacoes/"+uuid.NewString(), nil), http.StatusNotFound)
	conferirStatus(t, envia(t, servidor, http.MethodGet, "/api/v1/importacoes?status=em_transito", nil), http.StatusBadRequest)
	resposta = envia(t, servidor, http.MethodGet, "/api/v1/importacoes?status=cancelada", nil)
	conferirStatus(t, resposta, http.StatusOK)
	if pagina := decodificarPagina[importacaodto.ImportacaoResponse](t, resposta); len(pagina.Dados) != 1 {
		t.Errorf("canceladas = %+v", pagina)
	}

	conferirMensagem(t, envia(t, servidor, http.MethodDelete, "/api/v1/fornecedores/"+fornecedor.ID.String(), nil),
		http.StatusBadRequest, "erro de validação: registro em uso por outros dados e não pode ser excluído")
}
