//go:build e2e

package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	estoquedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/estoque"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
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

func localDoTeste(t *testing.T, servidor *httptest.Server) uuid.UUID {
	t.Helper()

	return criarLocal(t, servidor, empresaDoTeste(t, servidor), "Local "+uuid.NewString()[:8]).ID
}

func saldoAnteriorAosLocais(t *testing.T, banco *sqlx.DB, produtoID uuid.UUID, quantidade int) {
	t.Helper()

	if _, err := banco.Exec(
		`UPDATE produto SET quantidade_atual = quantidade_atual + $1, versao_estoque = versao_estoque + 1 WHERE id = $2`,
		quantidade, produtoID,
	); err != nil {
		t.Fatalf("simular saldo anterior aos locais falhou: %v", err)
	}
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
	servidor, banco := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	red := criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond").ID
	galpao := criarLocal(t, servidor, mia, "Galpão")
	loja := criarLocal(t, servidor, mia, "Loja")
	depositoRed := criarLocal(t, servidor, red, "Depósito")
	produtoID := produtoDoTeste(t, servidor)
	caminho := "/api/v1/produtos/" + produtoID.String()

	saldoAnteriorAosLocais(t, banco, produtoID, 9)

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

func TestFluxoDeEstoqueComLocaisLotesETransferencia(t *testing.T) {
	servidor, banco := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	galpao := criarLocal(t, servidor, mia, "Galpão")
	loja := criarLocal(t, servidor, mia, "Loja")
	produtoID := produtoDoTeste(t, servidor)
	caminho := "/api/v1/produtos/" + produtoID.String()
	saldoAnteriorAosLocais(t, banco, produtoID, 4)

	movimentar := func(corpo map[string]any) *http.Response {
		return envia(t, servidor, http.MethodPost, caminho+"/movimentos", corpo)
	}

	conferirStatus(t, movimentar(map[string]any{"tipo": "entrada", "quantidade": 1, "custo_unitario_centavos": 100}), http.StatusBadRequest)
	conferirStatus(t, movimentar(map[string]any{"local_id": galpao.ID, "tipo": "entrada", "quantidade": 1}), http.StatusBadRequest)
	conferirStatus(t, movimentar(map[string]any{"local_id": galpao.ID, "tipo": "implantacao", "quantidade": 5}), http.StatusBadRequest)
	conferirStatus(t, movimentar(map[string]any{"local_id": galpao.ID, "tipo": "transferencia", "quantidade": 1}), http.StatusBadRequest)

	resposta := movimentar(map[string]any{"local_id": galpao.ID, "tipo": "implantacao", "quantidade": 4})
	conferirStatus(t, resposta, http.StatusCreated)
	implantacao := decodificarEnvelope[estoquedto.MovimentoResponse](t, resposta).Dados
	if implantacao.SaldoApos != 4 || implantacao.LocalID == nil || *implantacao.LocalID != galpao.ID || *implantacao.SaldoLocalApos != 4 {
		t.Errorf("implantação = %+v, esperado total 4 e galpão 4", implantacao)
	}

	conferirStatus(t, movimentar(map[string]any{"local_id": galpao.ID, "tipo": "entrada", "quantidade": 6, "custo_unitario_centavos": 2590}), http.StatusCreated)

	resposta = envia(t, servidor, http.MethodPost, caminho+"/transferencias", map[string]any{
		"local_origem_id":  galpao.ID,
		"local_destino_id": loja.ID,
		"quantidade":       5,
	})
	conferirStatus(t, resposta, http.StatusCreated)
	if transferencia := decodificarEnvelope[estoquedto.MovimentoResponse](t, resposta).Dados; transferencia.Tipo != "transferencia" || transferencia.SaldoApos != 10 || *transferencia.SaldoLocalApos != 5 {
		t.Errorf("transferência = %+v, esperado total 10 e galpão 5", transferencia)
	}

	conferirStatus(t, movimentar(map[string]any{"local_id": loja.ID, "tipo": "saida", "quantidade": 6}), http.StatusBadRequest)
	conferirStatus(t, movimentar(map[string]any{"local_id": loja.ID, "tipo": "saida", "quantidade": 5}), http.StatusCreated)

	resposta = envia(t, servidor, http.MethodGet, caminho+"/saldo", nil)
	conferirStatus(t, resposta, http.StatusOK)
	saldo := decodificarEnvelope[estoquedto.SaldoResponse](t, resposta).Dados
	porLocal := map[uuid.UUID]int{}
	for _, local := range saldo.Locais {
		porLocal[local.LocalID] = local.Quantidade
	}
	if saldo.Saldo != 5 || saldo.NaoAlocado != 0 || porLocal[galpao.ID] != 5 || porLocal[loja.ID] != 0 {
		t.Errorf("saldo final = %+v, esperado 5 no galpão e loja zerada", saldo)
	}

	resposta = envia(t, servidor, http.MethodGet, caminho+"/lotes?com_saldo=true", nil)
	conferirStatus(t, resposta, http.StatusOK)
	lotes := decodificarPagina[estoquedto.LoteResponse](t, resposta).Dados
	if len(lotes) != 1 || lotes[0].LocalID != galpao.ID || lotes[0].QuantidadeAtual != 5 || lotes[0].CustoUnitarioCentavos == nil || *lotes[0].CustoUnitarioCentavos != 2590 {
		t.Errorf("lotes com saldo = %+v, esperado só o da entrada no galpão com 5", lotes)
	}

	resposta = envia(t, servidor, http.MethodGet, caminho+"/lotes?local="+loja.ID.String(), nil)
	conferirStatus(t, resposta, http.StatusOK)
	lotesDaLoja := decodificarPagina[estoquedto.LoteResponse](t, resposta).Dados
	if len(lotesDaLoja) != 2 || lotesDaLoja[0].LoteOrigemID == nil || lotesDaLoja[0].CustoUnitarioCentavos != nil || lotesDaLoja[1].QuantidadeAtual != 0 {
		t.Errorf("lotes da loja = %+v, esperado o da implantação sem custo e o da entrada, ambos consumidos", lotesDaLoja)
	}

	resposta = envia(t, servidor, http.MethodGet, caminho+"/movimentos?local="+loja.ID.String(), nil)
	conferirStatus(t, resposta, http.StatusOK)
	if movimentos := decodificarPagina[estoquedto.MovimentoResponse](t, resposta).Dados; len(movimentos) != 2 {
		t.Errorf("movimentos da loja = %d, esperado a transferência e a saída", len(movimentos))
	}

	conferirStatus(t, envia(t, servidor, http.MethodPatch, "/api/v1/estoque/locais/"+galpao.ID.String()+"/desativar", nil), http.StatusBadRequest)
	conferirStatus(t, envia(t, servidor, http.MethodPatch, "/api/v1/estoque/locais/"+loja.ID.String()+"/desativar", nil), http.StatusOK)
	conferirStatus(t, movimentar(map[string]any{"local_id": loja.ID, "tipo": "devolucao", "quantidade": 1}), http.StatusBadRequest)
}
