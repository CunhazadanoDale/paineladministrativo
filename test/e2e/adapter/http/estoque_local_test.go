//go:build e2e

package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	estoquedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/estoque"
	"github.com/google/uuid"
)

func criarLocal(t *testing.T, servidor *httptest.Server, empresaID uuid.UUID, nome string) estoquedto.LocalResponse {
	t.Helper()

	resposta := envia(t, servidor, http.MethodPost, "/api/v1/estoque/locais", map[string]any{
		"empresa_id": empresaID,
		"nome":       nome,
	})
	conferirStatus(t, resposta, http.StatusCreated)

	return decodificarEnvelope[estoquedto.LocalResponse](t, resposta).Dados
}

func produtoDoTeste(t *testing.T, servidor *httptest.Server) uuid.UUID {
	t.Helper()

	raiz := criarCategoria(t, servidor, "Materiais", nil)
	sub := criarCategoria(t, servidor, "Cimento", &raiz)

	return criarProduto(t, servidor, sub, "Cimento CP II")
}

func TestAdministradorGerenciaLocaisDeEstoque(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID

	galpao := criarLocal(t, servidor, mia, "Galpão Cabedelo")
	if galpao.EmpresaID != mia || galpao.Nome != "Galpão Cabedelo" || !galpao.Ativo {
		t.Errorf("local criado = %+v", galpao)
	}
	caminho := "/api/v1/estoque/locais/" + galpao.ID.String()

	resposta := envia(t, servidor, http.MethodPost, "/api/v1/estoque/locais", map[string]any{"empresa_id": mia, "nome": "galpão cabedelo"})
	conferirStatus(t, resposta, http.StatusConflict)
	if mensagem := decodificarErro(t, resposta).Erro.Mensagem; mensagem != "já existe um local com este nome na empresa" {
		t.Errorf("mensagem do nome repetido = %q", mensagem)
	}

	resposta = envia(t, servidor, http.MethodPost, "/api/v1/estoque/locais", map[string]any{"empresa_id": uuid.New(), "nome": "Galpão"})
	conferirStatus(t, resposta, http.StatusBadRequest)

	resposta = envia(t, servidor, http.MethodPut, caminho, map[string]any{"nome": "Galpão Norte"})
	conferirStatus(t, resposta, http.StatusOK)
	if renomeado := decodificarEnvelope[estoquedto.LocalResponse](t, resposta).Dados; renomeado.Nome != "Galpão Norte" {
		t.Errorf("nome depois de renomear = %q", renomeado.Nome)
	}

	resposta = envia(t, servidor, http.MethodPut, caminho, map[string]any{"nome": "Galpão", "empresa_id": uuid.New()})
	conferirStatus(t, resposta, http.StatusBadRequest)

	resposta = envia(t, servidor, http.MethodPatch, caminho+"/desativar", nil)
	conferirStatus(t, resposta, http.StatusOK)
	if desativado := decodificarEnvelope[estoquedto.LocalResponse](t, resposta).Dados; desativado.Ativo {
		t.Error("local deveria estar inativo")
	}

	resposta = envia(t, servidor, http.MethodGet, "/api/v1/estoque/locais?ativos=true", nil)
	conferirStatus(t, resposta, http.StatusOK)
	if ativos := decodificarPagina[estoquedto.LocalResponse](t, resposta).Dados; len(ativos) != 0 {
		t.Errorf("locais ativos = %+v, esperado nenhum", ativos)
	}

	resposta = envia(t, servidor, http.MethodPatch, caminho+"/ativar", nil)
	conferirStatus(t, resposta, http.StatusOK)

	resposta = envia(t, servidor, http.MethodGet, caminho+"/saldos", nil)
	conferirStatus(t, resposta, http.StatusOK)
	if saldos := decodificarPagina[estoquedto.SaldoLocalResponse](t, resposta).Dados; len(saldos) != 0 {
		t.Errorf("saldos de local novo = %+v, esperado nenhum", saldos)
	}

	resposta = envia(t, servidor, http.MethodDelete, caminho, nil)
	conferirStatus(t, resposta, http.StatusNoContent)

	resposta = envia(t, servidor, http.MethodGet, caminho, nil)
	conferirStatus(t, resposta, http.StatusNotFound)
}

func TestLocaisDeEstoqueFicamIsoladosPorEmpresa(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	red := criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond").ID
	galpaoMia := criarLocal(t, servidor, mia, "Galpão MIA")
	depositoRed := criarLocal(t, servidor, red, "Depósito Red")
	produtoID := produtoDoTeste(t, servidor)

	cargoFinanceiro := criarCargoComPerfil(t, servidor, "Financeiro", false, true)
	cargoOperador := criarCargo(t, servidor, "Operador", false)
	idAna := criarUsuario(t, servidor, "Ana Souza", "ana@exemplo.com", "senhaForte123", cargoFinanceiro)
	idBia := criarUsuario(t, servidor, "Bia Lima", "bia@exemplo.com", "senhaForte123", cargoOperador)
	conferirStatus(t, envia(t, servidor, http.MethodPut, "/api/v1/usuarios/"+idAna.String()+"/empresas", map[string]any{
		"empresa_ids": []uuid.UUID{mia},
	}), http.StatusOK)
	conferirStatus(t, envia(t, servidor, http.MethodPut, "/api/v1/usuarios/"+idBia.String()+"/empresas", map[string]any{
		"empresa_ids": []uuid.UUID{red},
	}), http.StatusOK)
	tokenAna := autenticar(t, servidor, "ana@exemplo.com", "senhaForte123").Token
	tokenBia := autenticar(t, servidor, "bia@exemplo.com", "senhaForte123").Token

	resposta := enviaComToken(t, servidor, http.MethodGet, "/api/v1/estoque/locais", nil, tokenAna)
	conferirStatus(t, resposta, http.StatusOK)
	if locais := decodificarPagina[estoquedto.LocalResponse](t, resposta).Dados; len(locais) != 1 || locais[0].ID != galpaoMia.ID {
		t.Errorf("locais da Ana = %+v, esperado só o galpão da MIA", locais)
	}

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/estoque/locais/"+depositoRed.ID.String(), nil, tokenAna)
	conferirStatus(t, resposta, http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/estoque/locais?empresa="+red.String(), nil, tokenAna)
	conferirStatus(t, resposta, http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/estoque/locais/"+depositoRed.ID.String()+"/saldos", nil, tokenAna)
	conferirStatus(t, resposta, http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/produtos/"+produtoID.String()+"/movimentos?local="+depositoRed.ID.String(), nil, tokenAna)
	conferirStatus(t, resposta, http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/estoque/resumo?empresa="+red.String(), nil, tokenAna)
	conferirStatus(t, resposta, http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/produtos/"+produtoID.String()+"/lotes", nil, tokenAna)
	conferirStatus(t, resposta, http.StatusOK)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/produtos/"+produtoID.String()+"/lotes", nil, tokenBia)
	conferirStatus(t, resposta, http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodPost, "/api/v1/estoque/locais", map[string]any{"empresa_id": red, "nome": "Novo"}, tokenBia)
	conferirStatus(t, resposta, http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodPatch, "/api/v1/estoque/locais/"+depositoRed.ID.String()+"/desativar", nil, tokenBia)
	conferirStatus(t, resposta, http.StatusForbidden)
}

func TestSaldoMostraNaoAlocadoETransferenciaValidaLocais(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	red := criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond").ID
	galpao := criarLocal(t, servidor, mia, "Galpão")
	loja := criarLocal(t, servidor, mia, "Loja")
	depositoRed := criarLocal(t, servidor, red, "Depósito")
	produtoID := produtoDoTeste(t, servidor)
	caminho := "/api/v1/produtos/" + produtoID.String()

	conferirStatus(t, envia(t, servidor, http.MethodPost, caminho+"/movimentos", map[string]any{"tipo": "entrada", "quantidade": 9}), http.StatusCreated)

	resposta := envia(t, servidor, http.MethodGet, caminho+"/saldo", nil)
	conferirStatus(t, resposta, http.StatusOK)
	saldo := decodificarEnvelope[estoquedto.SaldoResponse](t, resposta).Dados
	if saldo.Saldo != 9 || saldo.NaoAlocado != 9 || saldo.Disponivel != 9 || saldo.Reservado != 0 || len(saldo.Locais) != 0 {
		t.Errorf("saldo antes dos locais = %+v, esperado tudo não alocado", saldo)
	}

	transferencia := func(origem, destino uuid.UUID, quantidade int) *http.Response {
		return envia(t, servidor, http.MethodPost, caminho+"/transferencias", map[string]any{
			"local_origem_id":  origem,
			"local_destino_id": destino,
			"quantidade":       quantidade,
		})
	}

	conferirStatus(t, transferencia(galpao.ID, loja.ID, 1), http.StatusBadRequest)
	conferirStatus(t, transferencia(galpao.ID, depositoRed.ID, 1), http.StatusBadRequest)
	conferirStatus(t, transferencia(galpao.ID, galpao.ID, 1), http.StatusBadRequest)
	conferirStatus(t, transferencia(uuid.New(), loja.ID, 1), http.StatusBadRequest)
	conferirStatus(t, envia(t, servidor, http.MethodPost, "/api/v1/produtos/"+uuid.NewString()+"/transferencias", map[string]any{
		"local_origem_id":  galpao.ID,
		"local_destino_id": loja.ID,
		"quantidade":       1,
	}), http.StatusNotFound)

	resposta = envia(t, servidor, http.MethodGet, caminho+"/lotes", nil)
	conferirStatus(t, resposta, http.StatusOK)
	if lotes := decodificarPagina[estoquedto.LoteResponse](t, resposta).Dados; len(lotes) != 0 {
		t.Errorf("lotes antes da implantação = %+v, esperado nenhum", lotes)
	}
}
