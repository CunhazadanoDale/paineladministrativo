//go:build e2e

package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	financeirodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/financeiro"
	"github.com/google/uuid"
)

func tokenFinanceiroDasEmpresas(t *testing.T, servidor *httptest.Server, nome, email string, empresaIDs ...uuid.UUID) string {
	t.Helper()

	usuarioID := criarUsuario(t, servidor, nome, email, "senhaForte123", criarCargoComPerfil(t, servidor, "Financeiro "+nome, false, true))
	resposta := envia(t, servidor, http.MethodPut, "/api/v1/usuarios/"+usuarioID.String()+"/empresas", map[string]any{
		"acesso_todas": false,
		"empresa_ids":  empresaIDs,
	})
	conferirStatus(t, resposta, http.StatusOK)

	return autenticar(t, servidor, email, "senhaForte123").Token
}

func criarContaE2E(t *testing.T, servidor *httptest.Server, corpo map[string]any, token string) financeirodto.ContaBancariaResponse {
	t.Helper()

	resposta := enviaComToken(t, servidor, http.MethodPost, "/api/v1/contas-bancarias", corpo, token)
	conferirStatus(t, resposta, http.StatusCreated)

	return decodificarEnvelope[financeirodto.ContaBancariaResponse](t, resposta).Dados
}

func criarCategoriaFinanceiraE2E(t *testing.T, servidor *httptest.Server, corpo map[string]any, token string) financeirodto.CategoriaFinanceiraResponse {
	t.Helper()

	resposta := enviaComToken(t, servidor, http.MethodPost, "/api/v1/categorias-financeiras", corpo, token)
	conferirStatus(t, resposta, http.StatusCreated)

	return decodificarEnvelope[financeirodto.CategoriaFinanceiraResponse](t, resposta).Dados
}

func conferirMensagem(t *testing.T, resposta *http.Response, status int, mensagem string) {
	t.Helper()

	conferirStatus(t, resposta, status)
	if obtida := decodificarErro(t, resposta).Erro.Mensagem; obtida != mensagem {
		t.Errorf("mensagem %q, esperada %q", obtida, mensagem)
	}
}

func TestFinanceiroGerenciaContasBancarias(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA")
	tokenMIA := tokenFinanceiroDasEmpresas(t, servidor, "Fábio MIA", "fabio@exemplo.com", mia.ID)

	conta := criarContaE2E(t, servidor, map[string]any{
		"empresa_id": mia.ID, "banco": "748", "tipo": "corrente", "agencia": "0812", "conta": "54.321-0", "apelido": " Sicredi movimento ",
	}, tokenMIA)
	if conta.EmpresaID != mia.ID || conta.BancoNome != "Sicredi" || conta.Agencia != "0812" || conta.AgenciaDigito != nil {
		t.Errorf("conta criada = %+v", conta)
	}
	if conta.Conta != "54321" || conta.ContaDigito == nil || *conta.ContaDigito != "0" || conta.ContaFormatada != "54321-0" {
		t.Errorf("número da conta = %+v", conta)
	}
	if conta.Apelido != "Sicredi movimento" || !conta.Ativa || conta.Versao != 1 {
		t.Errorf("conta criada = %+v", conta)
	}

	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/contas-bancarias", map[string]any{
		"empresa_id": mia.ID, "banco": "748", "tipo": "pagamento", "agencia": "812", "conta": "0054321", "apelido": "Outra",
	}, tokenMIA), http.StatusConflict, "já existe uma conta com este banco, agência e número")
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/contas-bancarias", map[string]any{
		"empresa_id": mia.ID, "banco": "341", "tipo": "corrente", "agencia": "1", "conta": "2", "apelido": "SICREDI MOVIMENTO",
	}, tokenMIA), http.StatusConflict, "já existe uma conta com este apelido na empresa")

	invalidos := map[string]map[string]any{
		"banco fora do catálogo": {"empresa_id": mia.ID, "banco": "104", "tipo": "corrente", "agencia": "1", "conta": "1", "apelido": "A"},
		"tipo caixa":             {"empresa_id": mia.ID, "banco": "001", "tipo": "caixa", "agencia": "1", "conta": "1", "apelido": "A"},
		"conta zerada":           {"empresa_id": mia.ID, "banco": "001", "tipo": "corrente", "agencia": "1", "conta": "000", "apelido": "A"},
		"dígito duas vezes":      {"empresa_id": mia.ID, "banco": "001", "tipo": "corrente", "agencia": "1", "conta": "1-2", "conta_digito": "2", "apelido": "A"},
		"sem empresa":            {"banco": "001", "tipo": "corrente", "agencia": "1", "conta": "1", "apelido": "A"},
		"campo desconhecido":     {"empresa_id": mia.ID, "banco": "001", "tipo": "corrente", "agencia": "1", "conta": "1", "apelido": "A", "saldo": 10},
	}
	for nome, corpo := range invalidos {
		if resposta := enviaComToken(t, servidor, http.MethodPost, "/api/v1/contas-bancarias", corpo, tokenMIA); resposta.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status %d, esperado %d", nome, resposta.StatusCode, http.StatusBadRequest)
		}
	}

	caminho := "/api/v1/contas-bancarias/" + conta.ID.String()
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, caminho, nil, tokenMIA), http.StatusOK)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPut, caminho, map[string]any{"apelido": "Sem versão"}, tokenMIA), http.StatusBadRequest)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPut, caminho, map[string]any{"apelido": "X", "versao": 1, "conta": "9"}, tokenMIA), http.StatusBadRequest)

	resposta := enviaComToken(t, servidor, http.MethodPut, caminho, map[string]any{"apelido": "Sicredi pagamentos", "versao": 1}, tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	if alterada := decodificarEnvelope[financeirodto.ContaBancariaResponse](t, resposta).Dados; alterada.Apelido != "Sicredi pagamentos" || alterada.Versao != 2 || alterada.Conta != "54321" {
		t.Errorf("alterada = %+v", alterada)
	}
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPut, caminho, map[string]any{"apelido": "Velha", "versao": 1}, tokenMIA),
		http.StatusConflict, "a conta bancária foi alterada por outra pessoa; recarregue e tente de novo")

	conferirStatus(t, enviaComToken(t, servidor, http.MethodPatch, caminho+"/desativar", nil, tokenMIA), http.StatusBadRequest)
	resposta = enviaComToken(t, servidor, http.MethodPatch, caminho+"/desativar", map[string]any{"versao": 2}, tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	if desativada := decodificarEnvelope[financeirodto.ContaBancariaResponse](t, resposta).Dados; desativada.Ativa || desativada.Versao != 3 {
		t.Errorf("desativada = %+v", desativada)
	}
	resposta = enviaComToken(t, servidor, http.MethodPatch, caminho+"/desativar", map[string]any{"versao": 3}, tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	if denovo := decodificarEnvelope[financeirodto.ContaBancariaResponse](t, resposta).Dados; denovo.Versao != 3 {
		t.Errorf("desativar de novo mudou a versão para %d", denovo.Versao)
	}

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/contas-bancarias?ativa=false&banco=748", nil, tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	if inativas := decodificarPagina[financeirodto.ContaBancariaResponse](t, resposta).Dados; len(inativas) != 1 || inativas[0].ID != conta.ID {
		t.Errorf("inativas = %+v", inativas)
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/contas-bancarias?banco=104", nil, tokenMIA), http.StatusBadRequest)

	conferirStatus(t, envia(t, servidor, http.MethodPatch, "/api/v1/empresas/"+mia.ID.String()+"/desativar", nil), http.StatusOK)
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPatch, caminho+"/ativar", map[string]any{"versao": 3}, tokenMIA),
		http.StatusBadRequest, "erro de validação: empresa inativa não recebe conta bancária")
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/contas-bancarias", map[string]any{
		"empresa_id": mia.ID, "banco": "001", "tipo": "corrente", "agencia": "1", "conta": "1", "apelido": "BB",
	}, tokenMIA), http.StatusBadRequest, "erro de validação: empresa inativa não recebe conta bancária")

	conferirStatus(t, envia(t, servidor, http.MethodPatch, "/api/v1/empresas/"+mia.ID.String()+"/ativar", nil), http.StatusOK)
	resposta = enviaComToken(t, servidor, http.MethodPatch, caminho+"/ativar", map[string]any{"versao": 3}, tokenMIA)
	conferirStatus(t, resposta, http.StatusOK)
	if reativada := decodificarEnvelope[financeirodto.ContaBancariaResponse](t, resposta).Dados; !reativada.Ativa || reativada.Versao != 4 {
		t.Errorf("reativada = %+v", reativada)
	}
}

func TestContasBancariasIsoladasPorEmpresaPelaAPI(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA")
	red := criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond")
	tokenMIA := tokenFinanceiroDasEmpresas(t, servidor, "Fábio MIA", "fabio@exemplo.com", mia.ID)
	tokenRed := tokenFinanceiroDasEmpresas(t, servidor, "Rita Red", "rita@exemplo.com", red.ID)
	tokenSemEmpresa := tokenFinanceiroDasEmpresas(t, servidor, "Sérgio Sem Empresa", "sergio@exemplo.com")

	daMIA := criarContaE2E(t, servidor, map[string]any{"empresa_id": mia.ID, "banco": "403", "tipo": "corrente", "agencia": "1", "conta": "1", "apelido": "Cora MIA"}, tokenMIA)
	daRed := criarContaE2E(t, servidor, map[string]any{"empresa_id": red.ID, "banco": "403", "tipo": "corrente", "agencia": "1", "conta": "2", "apelido": "Cora Red"}, tokenRed)

	for token, propria := range map[string]uuid.UUID{tokenMIA: daMIA.ID, tokenRed: daRed.ID} {
		resposta := enviaComToken(t, servidor, http.MethodGet, "/api/v1/contas-bancarias", nil, token)
		conferirStatus(t, resposta, http.StatusOK)
		if contas := decodificarPagina[financeirodto.ContaBancariaResponse](t, resposta).Dados; len(contas) != 1 || contas[0].ID != propria {
			t.Errorf("listagem isolada = %+v", contas)
		}
	}

	resposta := enviaComToken(t, servidor, http.MethodGet, "/api/v1/contas-bancarias", nil, tokenSemEmpresa)
	conferirStatus(t, resposta, http.StatusOK)
	if contas := decodificarPagina[financeirodto.ContaBancariaResponse](t, resposta).Dados; len(contas) != 0 {
		t.Errorf("usuário sem empresa enxergou %d contas", len(contas))
	}

	resposta = envia(t, servidor, http.MethodGet, "/api/v1/contas-bancarias?empresa="+red.ID.String(), nil)
	conferirStatus(t, resposta, http.StatusOK)
	if contas := decodificarPagina[financeirodto.ContaBancariaResponse](t, resposta).Dados; len(contas) != 1 || contas[0].ID != daRed.ID {
		t.Errorf("administrador filtrando a Red = %+v", contas)
	}

	caminhoDaRed := "/api/v1/contas-bancarias/" + daRed.ID.String()
	semAcesso := "perfil sem acesso a esta empresa"
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodGet, caminhoDaRed, nil, tokenMIA), http.StatusForbidden, semAcesso)
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPut, caminhoDaRed, map[string]any{"apelido": "Minha", "versao": 1}, tokenMIA), http.StatusForbidden, semAcesso)
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPatch, caminhoDaRed+"/desativar", map[string]any{"versao": 1}, tokenMIA), http.StatusForbidden, semAcesso)
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/contas-bancarias?empresa="+red.ID.String(), nil, tokenMIA), http.StatusForbidden, semAcesso)
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/contas-bancarias", map[string]any{
		"empresa_id": red.ID, "banco": "001", "tipo": "corrente", "agencia": "1", "conta": "3", "apelido": "Invasora",
	}, tokenMIA), http.StatusForbidden, semAcesso)

	resposta = enviaComToken(t, servidor, http.MethodGet, caminhoDaRed, nil, tokenRed)
	conferirStatus(t, resposta, http.StatusOK)
	if intacta := decodificarEnvelope[financeirodto.ContaBancariaResponse](t, resposta).Dados; intacta.Apelido != "Cora Red" || !intacta.Ativa || intacta.Versao != 1 {
		t.Errorf("conta da Red foi alterada: %+v", intacta)
	}
}

func TestPerfisDasContasBancarias(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA")
	tokenMIA := tokenFinanceiroDasEmpresas(t, servidor, "Fábio MIA", "fabio@exemplo.com", mia.ID)
	_, tokenComercial := usuarioComercial(t, servidor, "Vera Nunes", "vera@exemplo.com")

	conta := criarContaE2E(t, servidor, map[string]any{"empresa_id": mia.ID, "banco": "001", "tipo": "corrente", "agencia": "1234-X", "conta": "1", "apelido": "BB"}, tokenMIA)
	if conta.AgenciaFormatada != "1234-X" {
		t.Errorf("agência formatada = %q", conta.AgenciaFormatada)
	}
	caminho := "/api/v1/contas-bancarias/" + conta.ID.String()

	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/contas-bancarias", nil, tokenComercial), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, caminho, nil, tokenComercial), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/contas-bancarias", map[string]any{"empresa_id": mia.ID}, tokenComercial), http.StatusForbidden)
	conferirStatus(t, enviaSemToken(t, servidor, http.MethodGet, "/api/v1/contas-bancarias", nil), http.StatusUnauthorized)

	resposta := enviaComToken(t, servidor, http.MethodGet, "/api/v1/bancos", nil, tokenComercial)
	conferirStatus(t, resposta, http.StatusOK)
	if bancos := decodificarEnvelope[[]financeirodto.BancoResponse](t, resposta).Dados; len(bancos) != 7 || bancos[0] != (financeirodto.BancoResponse{Codigo: "001", Nome: "Banco do Brasil"}) {
		t.Errorf("bancos = %+v", bancos)
	}
	conferirStatus(t, enviaSemToken(t, servidor, http.MethodGet, "/api/v1/bancos", nil), http.StatusUnauthorized)

	conferirStatus(t, enviaComToken(t, servidor, http.MethodDelete, caminho, nil, tokenMIA), http.StatusForbidden)
	conferirStatus(t, envia(t, servidor, http.MethodDelete, caminho, nil), http.StatusNoContent)
	conferirStatus(t, envia(t, servidor, http.MethodDelete, caminho, nil), http.StatusNotFound)
	conferirStatus(t, envia(t, servidor, http.MethodGet, caminho, nil), http.StatusNotFound)

	criarContaE2E(t, servidor, map[string]any{"empresa_id": mia.ID, "banco": "001", "tipo": "corrente", "agencia": "1", "conta": "2", "apelido": "Presa"}, tokenMIA)
	conferirMensagem(t, envia(t, servidor, http.MethodDelete, "/api/v1/empresas/"+mia.ID.String(), nil),
		http.StatusBadRequest, "erro de validação: registro em uso por outros dados e não pode ser excluído")
}

func TestEdicoesSimultaneasDeContaBancariaPelaAPI(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA")
	conta := criarContaE2E(t, servidor, map[string]any{"empresa_id": mia.ID, "banco": "077", "tipo": "pagamento", "agencia": "1", "conta": "1", "apelido": "Inter"}, tokenDoTeste)
	caminho := "/api/v1/contas-bancarias/" + conta.ID.String()

	const concorrentes = 6
	var espera sync.WaitGroup
	status := make(chan int, concorrentes)
	falhas := make(chan error, concorrentes)
	for indice := range concorrentes {
		espera.Add(1)
		go func() {
			defer espera.Done()
			codigo, err := putConcorrente(servidor, caminho, map[string]any{"versao": 1, "apelido": fmt.Sprintf("Edição %d", indice)})
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
}

func TestFinanceiroGerenciaPlanoDeContas(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	tokenFinanceiro := tokenComPerfil(t, servidor, "Tesouraria", "Tânia Caixa", "tania@exemplo.com", false, true)
	_, tokenComercial := usuarioComercial(t, servidor, "Vera Nunes", "vera@exemplo.com")

	despesas := criarCategoriaFinanceiraE2E(t, servidor, map[string]any{
		"nome": "Despesas administrativas", "natureza": "despesa", "linha_dre": "despesa_fixa", "codigo": "4.2", "ordem": 2,
	}, tokenFinanceiro)
	receitas := criarCategoriaFinanceiraE2E(t, servidor, map[string]any{
		"nome": "Receitas financeiras", "natureza": "receita", "linha_dre": "resultado_financeiro", "ordem": 1,
	}, tokenFinanceiro)
	if despesas.CategoriaPaiID != nil || despesas.Lancavel || despesas.Natureza != "despesa" || despesas.LinhaDRE != "despesa_fixa" || despesas.Versao != 1 {
		t.Errorf("raiz criada = %+v", despesas)
	}

	aluguel := criarCategoriaFinanceiraE2E(t, servidor, map[string]any{"nome": "Aluguel", "categoria_pai_id": despesas.ID, "codigo": "4.2.01"}, tokenFinanceiro)
	if aluguel.CategoriaPaiID == nil || *aluguel.CategoriaPaiID != despesas.ID || !aluguel.Lancavel || aluguel.Natureza != "despesa" || aluguel.LinhaDRE != "despesa_fixa" {
		t.Errorf("subcategoria criada = %+v", aluguel)
	}
	energia := criarCategoriaFinanceiraE2E(t, servidor, map[string]any{"nome": "Energia", "categoria_pai_id": despesas.ID}, tokenFinanceiro)

	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/categorias-financeiras", map[string]any{
		"nome": "Água", "categoria_pai_id": despesas.ID, "natureza": "despesa",
	}, tokenFinanceiro), http.StatusBadRequest, "erro de validação: subcategoria herda natureza e linha da DRE da categoria pai")
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/categorias-financeiras", map[string]any{"nome": "Sem classificação"}, tokenFinanceiro),
		http.StatusBadRequest, "erro de validação: natureza e linha da DRE são obrigatórias na categoria raiz")
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/categorias-financeiras", map[string]any{
		"nome": "Errada", "natureza": "receita", "linha_dre": "compra_mercadoria",
	}, tokenFinanceiro), http.StatusBadRequest, "erro de validação: linha da DRE compra_mercadoria não aceita natureza receita")
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/categorias-financeiras", map[string]any{
		"nome": "Neta", "categoria_pai_id": aluguel.ID,
	}, tokenFinanceiro), http.StatusBadRequest, "erro de validação: categoria pai deve ser uma categoria raiz ativa")
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/categorias-financeiras", map[string]any{
		"nome": "ALUGUEL", "categoria_pai_id": despesas.ID,
	}, tokenFinanceiro), http.StatusConflict, "já existe uma categoria com este nome neste nível")
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/categorias-financeiras", map[string]any{
		"nome": "Outra", "natureza": "despesa", "linha_dre": "fora_dre", "codigo": "4.2",
	}, tokenFinanceiro), http.StatusConflict, "já existe uma categoria com este código")
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/categorias-financeiras", map[string]any{
		"nome": "Comercial", "natureza": "despesa", "linha_dre": "despesa_fixa",
	}, tokenComercial), http.StatusForbidden)

	resposta := enviaComToken(t, servidor, http.MethodGet, "/api/v1/categorias-financeiras?raiz=true", nil, tokenComercial)
	conferirStatus(t, resposta, http.StatusOK)
	if raizes := decodificarPagina[financeirodto.CategoriaFinanceiraResponse](t, resposta).Dados; len(raizes) != 2 || raizes[0].ID != receitas.ID || raizes[1].ID != despesas.ID {
		t.Errorf("raízes = %+v", raizes)
	}
	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/categorias-financeiras?busca=4.2.0&natureza=despesa", nil, tokenComercial)
	conferirStatus(t, resposta, http.StatusOK)
	if encontradas := decodificarPagina[financeirodto.CategoriaFinanceiraResponse](t, resposta).Dados; len(encontradas) != 1 || encontradas[0].ID != aluguel.ID {
		t.Errorf("busca pelo código = %+v", encontradas)
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/categorias-financeiras?linha_dre=cmv", nil, tokenComercial), http.StatusBadRequest)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/categorias-financeiras/"+aluguel.ID.String(), nil, tokenComercial), http.StatusOK)

	caminhoRaiz := "/api/v1/categorias-financeiras/" + despesas.ID.String()
	caminhoEnergia := "/api/v1/categorias-financeiras/" + energia.ID.String()
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPatch, caminhoRaiz+"/desativar", map[string]any{"versao": 1}, tokenFinanceiro),
		http.StatusConflict, "categoria possui subcategorias ativas e não pode ser desativada")

	conferirStatus(t, enviaComToken(t, servidor, http.MethodPatch, caminhoEnergia+"/desativar", map[string]any{"versao": 1}, tokenFinanceiro), http.StatusOK)
	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/categorias-financeiras/arvore?ativas=true", nil, tokenComercial)
	conferirStatus(t, resposta, http.StatusOK)
	arvore := decodificarEnvelope[[]financeirodto.RamoDoPlanoResponse](t, resposta).Dados
	if len(arvore) != 2 || arvore[0].ID != receitas.ID || len(arvore[0].Subcategorias) != 0 || len(arvore[1].Subcategorias) != 1 || arvore[1].Subcategorias[0].ID != aluguel.ID {
		t.Errorf("árvore ativa = %+v", arvore)
	}

	conferirStatus(t, enviaComToken(t, servidor, http.MethodPatch, "/api/v1/categorias-financeiras/"+aluguel.ID.String()+"/desativar", map[string]any{"versao": 1}, tokenFinanceiro), http.StatusOK)
	resposta = enviaComToken(t, servidor, http.MethodPatch, caminhoRaiz+"/desativar", map[string]any{"versao": 1}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if desativada := decodificarEnvelope[financeirodto.CategoriaFinanceiraResponse](t, resposta).Dados; desativada.Ativa || desativada.Versao != 2 {
		t.Errorf("raiz desativada = %+v", desativada)
	}
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPatch, caminhoEnergia+"/ativar", map[string]any{"versao": 2}, tokenFinanceiro),
		http.StatusBadRequest, "erro de validação: categoria pai está inativa")
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/categorias-financeiras", map[string]any{"nome": "Água", "categoria_pai_id": despesas.ID}, tokenFinanceiro),
		http.StatusBadRequest, "erro de validação: categoria pai deve ser uma categoria raiz ativa")

	conferirStatus(t, enviaComToken(t, servidor, http.MethodPatch, caminhoRaiz+"/ativar", map[string]any{"versao": 2}, tokenFinanceiro), http.StatusOK)
	resposta = enviaComToken(t, servidor, http.MethodPatch, caminhoEnergia+"/ativar", map[string]any{"versao": 2}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if reativada := decodificarEnvelope[financeirodto.CategoriaFinanceiraResponse](t, resposta).Dados; !reativada.Lancavel || reativada.Versao != 3 {
		t.Errorf("subcategoria reativada = %+v", reativada)
	}

	conferirStatus(t, enviaComToken(t, servidor, http.MethodPut, caminhoRaiz, map[string]any{"nome": "X", "versao": 3, "natureza": "receita"}, tokenFinanceiro), http.StatusBadRequest)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPut, caminhoRaiz, map[string]any{"nome": "X", "versao": 3}, tokenComercial), http.StatusForbidden)
	resposta = enviaComToken(t, servidor, http.MethodPut, caminhoRaiz, map[string]any{"nome": "Despesas fixas", "codigo": "4.1", "ordem": 0, "versao": 3}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if alterada := decodificarEnvelope[financeirodto.CategoriaFinanceiraResponse](t, resposta).Dados; alterada.Nome != "Despesas fixas" || *alterada.Codigo != "4.1" || alterada.Versao != 4 || alterada.Natureza != "despesa" {
		t.Errorf("raiz alterada = %+v", alterada)
	}
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPut, caminhoRaiz, map[string]any{"nome": "Velha", "versao": 3}, tokenFinanceiro),
		http.StatusConflict, "a categoria financeira foi alterada por outra pessoa; recarregue e tente de novo")
}

func TestSoAdministradorExcluiCategoriaFinanceira(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	tokenFinanceiro := tokenComPerfil(t, servidor, "Tesouraria", "Tânia Caixa", "tania@exemplo.com", false, true)

	raiz := criarCategoriaFinanceiraE2E(t, servidor, map[string]any{"nome": "Impostos", "natureza": "despesa", "linha_dre": "deducao_venda"}, tokenFinanceiro)
	sub := criarCategoriaFinanceiraE2E(t, servidor, map[string]any{"nome": "ICMS", "categoria_pai_id": raiz.ID}, tokenFinanceiro)
	caminhoRaiz := "/api/v1/categorias-financeiras/" + raiz.ID.String()
	caminhoSub := "/api/v1/categorias-financeiras/" + sub.ID.String()

	conferirStatus(t, enviaComToken(t, servidor, http.MethodDelete, caminhoSub, nil, tokenFinanceiro), http.StatusForbidden)
	conferirMensagem(t, envia(t, servidor, http.MethodDelete, caminhoRaiz, nil), http.StatusBadRequest,
		"erro de validação: registro em uso por outros dados e não pode ser excluído")
	conferirStatus(t, envia(t, servidor, http.MethodDelete, caminhoSub, nil), http.StatusNoContent)
	conferirStatus(t, envia(t, servidor, http.MethodDelete, caminhoRaiz, nil), http.StatusNoContent)
	conferirStatus(t, envia(t, servidor, http.MethodGet, caminhoRaiz, nil), http.StatusNotFound)
	conferirStatus(t, enviaSemToken(t, servidor, http.MethodGet, "/api/v1/categorias-financeiras/arvore", nil), http.StatusUnauthorized)
}
