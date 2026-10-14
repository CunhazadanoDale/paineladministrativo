//go:build e2e

package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	fornecedordto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/fornecedor"
)

func tokenComPerfil(t *testing.T, servidor *httptest.Server, cargo, nome, email string, administrador, financeiro bool) string {
	t.Helper()

	criarUsuario(t, servidor, nome, email, "senhaForte123", criarCargoComPerfil(t, servidor, cargo, administrador, financeiro))

	return autenticar(t, servidor, email, "senhaForte123").Token
}

func criarFornecedorE2E(t *testing.T, servidor *httptest.Server, corpo map[string]any, token string) fornecedordto.FornecedorResponse {
	t.Helper()

	resposta := enviaComToken(t, servidor, http.MethodPost, "/api/v1/fornecedores", corpo, token)
	conferirStatus(t, resposta, http.StatusCreated)

	return decodificarEnvelope[fornecedordto.FornecedorResponse](t, resposta).Dados
}

func TestFinanceiroGerenciaFornecedores(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	tokenFinanceiro := tokenComPerfil(t, servidor, "Tesouraria", "Tânia Caixa", "tania@exemplo.com", false, true)
	_, tokenComercial := usuarioComercial(t, servidor, "Vera Nunes", "vera@exemplo.com")

	nacional := criarFornecedorE2E(t, servidor, map[string]any{
		"documento": "11.222.333/0001-81", "nome": "Cerâmica Paraíba LTDA", "uf": "pb", "cidade": "João Pessoa",
		"email": "vendas@ceramicapb.com.br",
	}, tokenFinanceiro)
	if nacional.Pais != "BR" || nacional.MoedaPadrao != "BRL" || nacional.DocumentoFormatado == nil || *nacional.DocumentoFormatado != "11.222.333/0001-81" {
		t.Errorf("nacional criado = %+v", nacional)
	}
	if nacional.TipoPessoa == nil || *nacional.TipoPessoa != "pj" || nacional.UF == nil || *nacional.UF != "PB" || nacional.Versao != 1 {
		t.Errorf("nacional criado = %+v", nacional)
	}

	estrangeiro := criarFornecedorE2E(t, servidor, map[string]any{
		"pais": "cn", "identificacao_estrangeira": "91330200ma2", "nome": "Foshan Ceramics", "moeda_padrao": "usd",
	}, tokenFinanceiro)
	if estrangeiro.Documento != nil || estrangeiro.IdentificacaoEstrangeira == nil || *estrangeiro.IdentificacaoEstrangeira != "91330200MA2" || estrangeiro.MoedaPadrao != "USD" {
		t.Errorf("estrangeiro criado = %+v", estrangeiro)
	}

	resposta := enviaComToken(t, servidor, http.MethodPost, "/api/v1/fornecedores", map[string]any{"documento": "11222333000181", "nome": "Outra"}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusConflict)
	if mensagem := decodificarErro(t, resposta).Erro.Mensagem; mensagem != "já existe um fornecedor com este documento" {
		t.Errorf("mensagem do documento repetido = %q", mensagem)
	}
	resposta = enviaComToken(t, servidor, http.MethodPost, "/api/v1/fornecedores", map[string]any{"pais": "CN", "identificacao_estrangeira": "91330200MA2", "nome": "Outra"}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusConflict)
	if mensagem := decodificarErro(t, resposta).Erro.Mensagem; mensagem != "já existe um fornecedor com esta identificação neste país" {
		t.Errorf("mensagem da identificação repetida = %q", mensagem)
	}

	invalidos := map[string]map[string]any{
		"brasil sem documento":      {"nome": "Sem documento"},
		"estrangeiro com documento": {"pais": "US", "documento": "11.222.333/0001-81", "nome": "Ohio"},
		"estrangeiro com uf":        {"pais": "US", "uf": "SP", "nome": "Ohio"},
		"moeda fora da lista":       {"documento": "529.982.247-25", "nome": "José", "moeda_padrao": "GBP"},
		"país inválido":             {"pais": "BRA", "nome": "X"},
	}
	for nome, corpo := range invalidos {
		if resposta := enviaComToken(t, servidor, http.MethodPost, "/api/v1/fornecedores", corpo, tokenFinanceiro); resposta.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status %d, esperado %d", nome, resposta.StatusCode, http.StatusBadRequest)
		}
	}

	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/fornecedores", map[string]any{"documento": "529.982.247-25", "nome": "José"}, tokenComercial), http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/fornecedores?busca=CERAMICA&pais=br", nil, tokenComercial)
	conferirStatus(t, resposta, http.StatusOK)
	if encontrados := decodificarPagina[fornecedordto.FornecedorResponse](t, resposta).Dados; len(encontrados) != 1 || encontrados[0].ID != nacional.ID {
		t.Errorf("busca por cerâmica no Brasil = %+v", encontrados)
	}
	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/fornecedores?moeda=USD", nil, tokenComercial)
	conferirStatus(t, resposta, http.StatusOK)
	if encontrados := decodificarPagina[fornecedordto.FornecedorResponse](t, resposta).Dados; len(encontrados) != 1 || encontrados[0].ID != estrangeiro.ID {
		t.Errorf("filtro por dólar = %+v", encontrados)
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/fornecedores?pais=BRA", nil, tokenComercial), http.StatusBadRequest)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/fornecedores/"+estrangeiro.ID.String(), nil, tokenComercial), http.StatusOK)

	caminho := "/api/v1/fornecedores/" + nacional.ID.String()
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPut, caminho, map[string]any{"versao": 1, "nome": "X", "documento": "529.982.247-25"}, tokenFinanceiro), http.StatusBadRequest)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPut, caminho, map[string]any{"nome": "Sem versão"}, tokenFinanceiro), http.StatusBadRequest)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPut, caminho, map[string]any{"versao": 1, "nome": "Comercial"}, tokenComercial), http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodPut, caminho, map[string]any{"versao": 1, "nome": "Cerâmica Nordeste", "moeda_padrao": "EUR", "uf": "PE"}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	alterado := decodificarEnvelope[fornecedordto.FornecedorResponse](t, resposta).Dados
	if alterado.Versao != 2 || alterado.Nome != "Cerâmica Nordeste" || alterado.MoedaPadrao != "EUR" || *alterado.Documento != "11222333000181" || alterado.Email != nil {
		t.Errorf("alterado = %+v", alterado)
	}

	resposta = enviaComToken(t, servidor, http.MethodPut, caminho, map[string]any{"versao": 1, "nome": "Versão velha"}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusConflict)
	if mensagem := decodificarErro(t, resposta).Erro.Mensagem; mensagem != "o fornecedor foi alterado por outra pessoa; recarregue e tente de novo" {
		t.Errorf("mensagem da versão velha = %q", mensagem)
	}

	conferirStatus(t, enviaComToken(t, servidor, http.MethodPatch, caminho+"/desativar", nil, tokenFinanceiro), http.StatusBadRequest)
	resposta = enviaComToken(t, servidor, http.MethodPatch, caminho+"/desativar", map[string]any{"versao": 2}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if desativado := decodificarEnvelope[fornecedordto.FornecedorResponse](t, resposta).Dados; desativado.Ativo || desativado.Versao != 3 {
		t.Errorf("desativado = %+v", desativado)
	}
	resposta = enviaComToken(t, servidor, http.MethodPatch, caminho+"/desativar", map[string]any{"versao": 3}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if denovo := decodificarEnvelope[fornecedordto.FornecedorResponse](t, resposta).Dados; denovo.Versao != 3 {
		t.Errorf("desativar de novo mudou a versão para %d", denovo.Versao)
	}

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/fornecedores?ativo=false", nil, tokenComercial)
	conferirStatus(t, resposta, http.StatusOK)
	if inativos := decodificarPagina[fornecedordto.FornecedorResponse](t, resposta).Dados; len(inativos) != 1 || inativos[0].ID != nacional.ID {
		t.Errorf("inativos = %+v", inativos)
	}

	resposta = enviaComToken(t, servidor, http.MethodPatch, caminho+"/ativar", map[string]any{"versao": 3}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if ativado := decodificarEnvelope[fornecedordto.FornecedorResponse](t, resposta).Dados; !ativado.Ativo || ativado.Versao != 4 {
		t.Errorf("ativado = %+v", ativado)
	}
}

func TestSoAdministradorExcluiFornecedor(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	tokenFinanceiro := tokenComPerfil(t, servidor, "Tesouraria", "Tânia Caixa", "tania@exemplo.com", false, true)

	fornecedor := criarFornecedorE2E(t, servidor, map[string]any{"pais": "IT", "nome": "Ceramiche Italia", "moeda_padrao": "EUR"}, tokenFinanceiro)
	caminho := "/api/v1/fornecedores/" + fornecedor.ID.String()

	conferirStatus(t, enviaComToken(t, servidor, http.MethodDelete, caminho, nil, tokenFinanceiro), http.StatusForbidden)
	conferirStatus(t, envia(t, servidor, http.MethodDelete, caminho, nil), http.StatusNoContent)
	conferirStatus(t, envia(t, servidor, http.MethodDelete, caminho, nil), http.StatusNotFound)
	conferirStatus(t, envia(t, servidor, http.MethodGet, caminho, nil), http.StatusNotFound)
}

func TestFornecedorExigeAutenticacao(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	conferirStatus(t, enviaSemToken(t, servidor, http.MethodGet, "/api/v1/fornecedores", nil), http.StatusUnauthorized)
	conferirStatus(t, enviaSemToken(t, servidor, http.MethodPost, "/api/v1/fornecedores", map[string]any{"nome": "X"}), http.StatusUnauthorized)
}

func putConcorrente(servidor *httptest.Server, caminho string, corpo map[string]any) (int, error) {
	bruto, err := json.Marshal(corpo)
	if err != nil {
		return 0, err
	}

	requisicao, err := http.NewRequest(http.MethodPut, servidor.URL+caminho, bytes.NewReader(bruto))
	if err != nil {
		return 0, err
	}
	requisicao.Header.Set("Content-Type", "application/json")
	requisicao.Header.Set("Authorization", "Bearer "+tokenDoTeste)

	resposta, err := servidor.Client().Do(requisicao)
	if err != nil {
		return 0, err
	}
	defer resposta.Body.Close()

	return resposta.StatusCode, nil
}

func TestEdicoesSimultaneasDeFornecedorPelaAPI(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	fornecedor := criarFornecedorE2E(t, servidor, map[string]any{"documento": "11.222.333/0001-81", "nome": "Cerâmica"}, tokenDoTeste)
	caminho := "/api/v1/fornecedores/" + fornecedor.ID.String()

	const concorrentes = 6
	var espera sync.WaitGroup
	status := make(chan int, concorrentes)
	falhas := make(chan error, concorrentes)
	for indice := range concorrentes {
		espera.Add(1)
		go func() {
			defer espera.Done()
			codigo, err := putConcorrente(servidor, caminho, map[string]any{"versao": 1, "nome": fmt.Sprintf("Edição %d", indice)})
			if err != nil {
				falhas <- err
				return
			}
			status <- codigo
		}()
	}
	espera.Wait()
	close(status)
	close(falhas)

	for err := range falhas {
		t.Errorf("requisição concorrente falhou: %v", err)
	}

	contagem := map[int]int{}
	for codigo := range status {
		contagem[codigo]++
	}
	if contagem[http.StatusOK] != 1 || contagem[http.StatusConflict] != concorrentes-1 {
		t.Errorf("respostas = %v; esperado uma 200 e %d 409", contagem, concorrentes-1)
	}

	resposta := envia(t, servidor, http.MethodGet, caminho, nil)
	conferirStatus(t, resposta, http.StatusOK)
	if final := decodificarEnvelope[fornecedordto.FornecedorResponse](t, resposta).Dados; final.Versao != 2 {
		t.Errorf("versão final = %d, esperada 2", final.Versao)
	}
}
