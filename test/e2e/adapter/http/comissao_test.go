//go:build e2e

package http

import (
	"net/http"
	"testing"
	"time"

	clientedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/cliente"
	comissaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/comissao"
	pedidodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/pedido"
)

func TestFluxoDeComissaoDoPercentualAApuracao(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	red := criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond").ID
	galpao := criarLocal(t, servidor, mia, "Galpão")
	produtoID := produtoDoTeste(t, servidor)
	conferirStatus(t, envia(t, servidor, http.MethodPost, "/api/v1/produtos/"+produtoID.String()+"/movimentos", map[string]any{
		"local_id": galpao.ID, "tipo": "entrada", "quantidade": 20, "custo_unitario_centavos": 900,
	}), http.StatusCreated)
	clienteID := clienteDoPedido(t, servidor)

	idAna, tokenAna := usuarioComercial(t, servidor, "Ana Souza", "ana@exemplo.com")
	idCaio, tokenCaio := usuarioComercial(t, servidor, "Caio Lima", "caio@exemplo.com")
	for _, usuarioID := range []string{idAna.String(), idCaio.String()} {
		conferirStatus(t, envia(t, servidor, http.MethodPut, "/api/v1/usuarios/"+usuarioID+"/empresas", map[string]any{
			"acesso_todas": false, "empresa_ids": []any{mia},
		}), http.StatusOK)
	}
	tokenFinanceiro := tokenFinanceiroDasEmpresas(t, servidor, "Fabi", "fabi@exemplo.com", mia)
	tokenFinanceiroRed := tokenFinanceiroDasEmpresas(t, servidor, "Rui", "rui@exemplo.com", red)

	resposta := envia(t, servidor, http.MethodPost, "/api/v1/representantes", map[string]any{
		"nome": "Ana Representações", "usuario_id": idAna, "comissao_pronta_entrega_bp": 300, "comissao_importacao_bp": 150,
	})
	conferirStatus(t, resposta, http.StatusCreated)
	ana := decodificarEnvelope[clientedto.RepresentanteResponse](t, resposta).Dados
	resposta = envia(t, servidor, http.MethodPost, "/api/v1/representantes", map[string]any{"nome": "Bia Vendas"})
	conferirStatus(t, resposta, http.StatusCreated)
	bia := decodificarEnvelope[clientedto.RepresentanteResponse](t, resposta).Dados

	corpo := corpoDePedido(mia, clienteID, produtoID, "pronta_entrega", 10)
	corpo["representante_id"] = ana.ID
	pedido := criarPedidoE2E(t, servidor, corpo, tokenDoTeste)
	caminho := "/api/v1/pedidos/" + pedido.ID.String()
	conferirStatus(t, envia(t, servidor, http.MethodPost, caminho+"/confirmar", map[string]any{"versao": 1}), http.StatusOK)

	resposta = envia(t, servidor, http.MethodGet, caminho+"/comissao", nil)
	conferirStatus(t, resposta, http.StatusOK)
	if sugerido := decodificarEnvelope[comissaodto.PercentualResponse](t, resposta).Dados; sugerido.Definido || sugerido.PercentualBP != 300 ||
		sugerido.Origem != "padrao" || sugerido.Versao != nil || sugerido.RepresentanteID != ana.ID {
		t.Errorf("percentual sugerido = %+v", sugerido)
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, caminho+"/comissao", nil, tokenAna), http.StatusOK)

	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPut, caminho+"/comissao", map[string]any{"percentual_bp": 900}, tokenAna),
		http.StatusForbidden, "representante não define a própria comissão")
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPut, caminho+"/comissao", map[string]any{"percentual_bp": 500}, tokenFinanceiro), http.StatusForbidden)
	conferirStatus(t, envia(t, servidor, http.MethodPut, caminho+"/comissao", map[string]any{}), http.StatusBadRequest)

	resposta = enviaComToken(t, servidor, http.MethodPut, caminho+"/comissao", map[string]any{"percentual_bp": 500}, tokenCaio)
	conferirStatus(t, resposta, http.StatusOK)
	if definido := decodificarEnvelope[comissaodto.PercentualResponse](t, resposta).Dados; !definido.Definido || definido.Versao == nil || *definido.Versao != 1 {
		t.Errorf("percentual definido = %+v", definido)
	}
	conferirStatus(t, envia(t, servidor, http.MethodPut, caminho+"/comissao", map[string]any{"percentual_bp": 450}), http.StatusBadRequest)
	conferirStatus(t, envia(t, servidor, http.MethodPut, caminho+"/comissao", map[string]any{"percentual_bp": 450, "versao": 9}), http.StatusConflict)
	resposta = envia(t, servidor, http.MethodPut, caminho+"/comissao", map[string]any{"percentual_bp": 450, "versao": 1})
	conferirStatus(t, resposta, http.StatusOK)
	if redefinido := decodificarEnvelope[comissaodto.PercentualResponse](t, resposta).Dados; redefinido.PercentualBP != 450 || *redefinido.Versao != 2 {
		t.Errorf("percentual redefinido = %+v", redefinido)
	}

	resposta = envia(t, servidor, http.MethodGet, caminho, nil)
	conferirStatus(t, resposta, http.StatusOK)
	item := decodificarEnvelope[pedidodto.PedidoDetalheResponse](t, resposta).Dados.Itens[0]
	resposta = envia(t, servidor, http.MethodPost, caminho+"/entregas", map[string]any{
		"frete_tipo": "cif", "frete_centavos": 5000,
		"itens": []map[string]any{{"pedido_item_id": item.ID, "local_id": galpao.ID, "quantidade": 4}},
	})
	conferirStatus(t, resposta, http.StatusCreated)
	entrega := decodificarEnvelope[pedidodto.EntregaResponse](t, resposta).Dados
	conferirStatus(t, envia(t, servidor, http.MethodPost, "/api/v1/entregas/"+entrega.ID.String()+"/saiu", map[string]any{"versao": 1}), http.StatusOK)

	conferirMensagem(t, envia(t, servidor, http.MethodPut, caminho+"/comissao", map[string]any{"percentual_bp": 100, "versao": 2}),
		http.StatusConflict, "a comissão do pedido não muda depois da primeira saída de entrega")

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/comissoes/pendencias", nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if pendentes := decodificarPagina[comissaodto.EntregaPendenteResponse](t, resposta).Dados; len(pendentes) != 1 ||
		pendentes[0].EntregaID != entrega.ID || pendentes[0].PedidoNumero != pedido.Numero {
		t.Errorf("pendências = %+v", pendentes)
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/comissoes/pendencias", nil, tokenAna), http.StatusForbidden)

	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/comissoes/apurar", map[string]any{"empresa_id": mia}, tokenFinanceiroRed), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/comissoes/apurar", map[string]any{"competencia": "10/2026"}, tokenFinanceiro), http.StatusBadRequest)

	resposta = enviaComToken(t, servidor, http.MethodPost, "/api/v1/comissoes/apurar", nil, tokenFinanceiroRed)
	conferirStatus(t, resposta, http.StatusOK)
	if resultado := decodificarEnvelope[comissaodto.ResultadoApuracaoResponse](t, resposta).Dados; len(resultado.Apuradas) != 0 {
		t.Errorf("a Red não deveria apurar entrega da MIA: %+v", resultado)
	}

	resposta = enviaComToken(t, servidor, http.MethodPost, "/api/v1/comissoes/apurar", nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if resultado := decodificarEnvelope[comissaodto.ResultadoApuracaoResponse](t, resposta).Dados; len(resultado.Apuradas) != 1 ||
		len(resultado.Falhas) != 0 || resultado.RestamEntregas {
		t.Errorf("apuração = %+v", resultado)
	}
	resposta = enviaComToken(t, servidor, http.MethodPost, "/api/v1/comissoes/apurar", map[string]any{}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if resultado := decodificarEnvelope[comissaodto.ResultadoApuracaoResponse](t, resposta).Dados; len(resultado.Apuradas) != 0 {
		t.Errorf("segunda apuração = %+v", resultado)
	}

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/comissoes?pedido="+pedido.ID.String(), nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	comissoes := decodificarPagina[comissaodto.ComissaoResponse](t, resposta).Dados
	if len(comissoes) != 1 || comissoes[0].EntregaID != entrega.ID || comissoes[0].BaseCentavos != 6000 ||
		comissoes[0].PercentualBP != 450 || comissoes[0].ValorCentavos != 270 || comissoes[0].RepresentanteID != ana.ID {
		t.Fatalf("comissões = %+v", comissoes)
	}

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/comissoes?competencia="+comissoes[0].Competencia, nil, tokenAna)
	conferirStatus(t, resposta, http.StatusOK)
	if daAna := decodificarPagina[comissaodto.ComissaoResponse](t, resposta).Dados; len(daAna) != 1 {
		t.Errorf("comissões da Ana = %+v", daAna)
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/comissoes?representante="+bia.ID.String(), nil, tokenAna), http.StatusForbidden)
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/comissoes", nil, tokenCaio), http.StatusForbidden, "perfil sem permissão para ver comissões")

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/comissoes", nil, tokenFinanceiroRed)
	conferirStatus(t, resposta, http.StatusOK)
	if daRed := decodificarPagina[comissaodto.ComissaoResponse](t, resposta).Dados; len(daRed) != 0 {
		t.Errorf("o financeiro da Red não deveria ver comissões da MIA: %+v", daRed)
	}
}

func TestAjustesDeComissaoPeloFinanceiro(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	red := criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond").ID
	produtoID := produtoDoTeste(t, servidor)
	clienteID := clienteDoPedido(t, servidor)

	idAna, tokenAna := usuarioComercial(t, servidor, "Ana Souza", "ana@exemplo.com")
	conferirStatus(t, envia(t, servidor, http.MethodPut, "/api/v1/usuarios/"+idAna.String()+"/empresas", map[string]any{
		"acesso_todas": false, "empresa_ids": []any{mia},
	}), http.StatusOK)
	_, tokenCaio := usuarioComercial(t, servidor, "Caio Lima", "caio@exemplo.com")
	tokenFinanceiro := tokenFinanceiroDasEmpresas(t, servidor, "Fabi", "fabi@exemplo.com", mia)
	tokenFinanceiroRed := tokenFinanceiroDasEmpresas(t, servidor, "Rui", "rui@exemplo.com", red)

	resposta := envia(t, servidor, http.MethodPost, "/api/v1/representantes", map[string]any{
		"nome": "Ana Representações", "usuario_id": idAna, "comissao_pronta_entrega_bp": 300,
	})
	conferirStatus(t, resposta, http.StatusCreated)
	ana := decodificarEnvelope[clientedto.RepresentanteResponse](t, resposta).Dados
	resposta = envia(t, servidor, http.MethodPost, "/api/v1/representantes", map[string]any{"nome": "Bia Vendas"})
	conferirStatus(t, resposta, http.StatusCreated)
	bia := decodificarEnvelope[clientedto.RepresentanteResponse](t, resposta).Dados

	corpo := corpoDePedido(mia, clienteID, produtoID, "pronta_entrega", 10)
	corpo["representante_id"] = ana.ID
	pedido := criarPedidoE2E(t, servidor, corpo, tokenDoTeste)

	competencia := time.Now().In(time.FixedZone("BRT", -3*60*60)).Format("2006-01")

	ajuste := map[string]any{
		"empresa_id": mia, "representante_id": ana.ID, "pedido_id": pedido.ID,
		"competencia": competencia, "valor_centavos": -375, "motivo": "devolução de uma peça",
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/comissoes/ajustes", ajuste, tokenCaio), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/comissoes/ajustes", ajuste, tokenFinanceiroRed), http.StatusForbidden)

	ajusteDaBia := map[string]any{
		"empresa_id": mia, "representante_id": bia.ID, "pedido_id": pedido.ID,
		"competencia": competencia, "valor_centavos": 100, "motivo": "teste",
	}
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/comissoes/ajustes", ajusteDaBia, tokenFinanceiro),
		http.StatusBadRequest, "erro de validação: pedido do ajuste deve ser da mesma empresa e do mesmo representante")
	futuro := map[string]any{
		"empresa_id": mia, "representante_id": ana.ID, "competencia": "2099-01", "valor_centavos": 100, "motivo": "teste",
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/comissoes/ajustes", futuro, tokenFinanceiro), http.StatusBadRequest)

	resposta = enviaComToken(t, servidor, http.MethodPost, "/api/v1/comissoes/ajustes", ajuste, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusCreated)
	criado := decodificarEnvelope[comissaodto.AjusteResponse](t, resposta).Dados
	if criado.ValorCentavos != -375 || criado.Competencia != competencia || criado.PedidoID == nil || *criado.PedidoID != pedido.ID {
		t.Fatalf("ajuste criado = %+v", criado)
	}
	premio := map[string]any{
		"empresa_id": mia, "representante_id": bia.ID, "competencia": competencia, "valor_centavos": 5000, "motivo": "prêmio de meta",
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/comissoes/ajustes", premio, tokenFinanceiro), http.StatusCreated)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/comissoes/ajustes?competencia="+competencia, nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if todos := decodificarPagina[comissaodto.AjusteResponse](t, resposta).Dados; len(todos) != 2 {
		t.Errorf("ajustes do financeiro = %+v", todos)
	}

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/comissoes/ajustes", nil, tokenAna)
	conferirStatus(t, resposta, http.StatusOK)
	if daAna := decodificarPagina[comissaodto.AjusteResponse](t, resposta).Dados; len(daAna) != 1 || daAna[0].ID != criado.ID {
		t.Errorf("ajustes da Ana = %+v", daAna)
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/comissoes/ajustes?representante="+bia.ID.String(), nil, tokenAna), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/comissoes/ajustes", nil, tokenCaio), http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/comissoes/ajustes", nil, tokenFinanceiroRed)
	conferirStatus(t, resposta, http.StatusOK)
	if daRed := decodificarPagina[comissaodto.AjusteResponse](t, resposta).Dados; len(daRed) != 0 {
		t.Errorf("o financeiro da Red não deveria ver ajustes da MIA: %+v", daRed)
	}
}
