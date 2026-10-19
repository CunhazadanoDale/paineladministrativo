//go:build e2e

package http

import (
	"net/http"
	"testing"
	"time"

	clientedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/cliente"
	comissaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/comissao"
	pedidodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/pedido"
	solicitacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/solicitacao"
)

func TestFechamentoDeComissaoGeraContaAPagar(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	galpao := criarLocal(t, servidor, mia, "Galpão")
	produtoID := produtoDoTeste(t, servidor)
	conferirStatus(t, envia(t, servidor, http.MethodPost, "/api/v1/produtos/"+produtoID.String()+"/movimentos", map[string]any{
		"local_id": galpao.ID, "tipo": "entrada", "quantidade": 20, "custo_unitario_centavos": 900,
	}), http.StatusCreated)
	clienteID := clienteDoPedido(t, servidor)

	idAna, tokenAna := usuarioComercial(t, servidor, "Ana Souza", "ana@exemplo.com")
	conferirStatus(t, envia(t, servidor, http.MethodPut, "/api/v1/usuarios/"+idAna.String()+"/empresas", map[string]any{
		"acesso_todas": false, "empresa_ids": []any{mia},
	}), http.StatusOK)
	_, tokenCaio := usuarioComercial(t, servidor, "Caio Lima", "caio@exemplo.com")
	tokenFinanceiro := tokenFinanceiroDasEmpresas(t, servidor, "Fabi", "fabi@exemplo.com", mia)

	resposta := envia(t, servidor, http.MethodPost, "/api/v1/representantes", map[string]any{
		"nome": "Ana Representações", "usuario_id": idAna, "comissao_pronta_entrega_bp": 300,
	})
	conferirStatus(t, resposta, http.StatusCreated)
	ana := decodificarEnvelope[clientedto.RepresentanteResponse](t, resposta).Dados
	fornecedor := criarFornecedorE2E(t, servidor, map[string]any{"documento": "45.723.174/0001-10", "nome": "Ana Representações Ltda"}, tokenDoTeste)
	raiz := criarCategoriaFinanceiraE2E(t, servidor, map[string]any{
		"nome": "Despesas de venda", "natureza": "despesa", "linha_dre": "despesa_variavel",
	}, tokenDoTeste)
	comissoes := criarCategoriaFinanceiraE2E(t, servidor, map[string]any{"nome": "Comissões", "categoria_pai_id": raiz.ID}, tokenDoTeste)

	brasilia := time.FixedZone("BRT", -3*60*60)
	inicioDoMes := time.Now().In(brasilia)
	inicioDoMes = time.Date(inicioDoMes.Year(), inicioDoMes.Month(), 1, 0, 0, 0, 0, time.UTC)
	ultimoDiaAnterior := inicioDoMes.AddDate(0, 0, -1)
	competencia := ultimoDiaAnterior.Format("2006-01")
	mesCorrente := inicioDoMes.Format("2006-01")

	corpo := corpoDePedido(mia, clienteID, produtoID, "pronta_entrega", 10)
	corpo["representante_id"] = ana.ID
	pedido := criarPedidoE2E(t, servidor, corpo, tokenDoTeste)
	caminho := "/api/v1/pedidos/" + pedido.ID.String()
	conferirStatus(t, envia(t, servidor, http.MethodPost, caminho+"/confirmar", map[string]any{"versao": 1}), http.StatusOK)
	resposta = envia(t, servidor, http.MethodGet, caminho, nil)
	conferirStatus(t, resposta, http.StatusOK)
	item := decodificarEnvelope[pedidodto.PedidoDetalheResponse](t, resposta).Dados.Itens[0]
	resposta = envia(t, servidor, http.MethodPost, caminho+"/entregas", map[string]any{
		"frete_tipo": "fob", "itens": []map[string]any{{"pedido_item_id": item.ID, "local_id": galpao.ID, "quantidade": 4}},
	})
	conferirStatus(t, resposta, http.StatusCreated)
	entrega := decodificarEnvelope[pedidodto.EntregaResponse](t, resposta).Dados
	conferirStatus(t, envia(t, servidor, http.MethodPost, "/api/v1/entregas/"+entrega.ID.String()+"/saiu", map[string]any{
		"versao": 1, "saiu_em": ultimoDiaAnterior.Format("2006-01-02"),
	}), http.StatusOK)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/comissoes/ajustes", map[string]any{
		"empresa_id": mia, "representante_id": ana.ID, "competencia": competencia, "valor_centavos": -100, "motivo": "devolução",
	}, tokenFinanceiro), http.StatusCreated)

	fechar := map[string]any{"empresa_id": mia, "competencia": competencia, "categoria_financeira_id": comissoes.ID}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/comissoes/fechamentos", fechar, tokenCaio), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/comissoes/fechamentos", map[string]any{
		"empresa_id": mia, "competencia": mesCorrente, "categoria_financeira_id": comissoes.ID,
	}, tokenFinanceiro), http.StatusBadRequest)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/comissoes/fechamentos", map[string]any{
		"empresa_id": mia, "competencia": competencia, "categoria_financeira_id": raiz.ID,
	}, tokenFinanceiro), http.StatusBadRequest)

	resposta = enviaComToken(t, servidor, http.MethodPost, "/api/v1/comissoes/fechamentos", fechar, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	semFornecedor := decodificarEnvelope[comissaodto.ResultadoFechamentoResponse](t, resposta).Dados
	if len(semFornecedor.Fechados) != 0 || len(semFornecedor.Falhas) != 1 || semFornecedor.Falhas[0].RepresentanteID != ana.ID ||
		len(semFornecedor.Apuracao.Apuradas) != 1 {
		t.Fatalf("fechamento sem fornecedor = %+v", semFornecedor)
	}

	caminhoDoFornecedor := "/api/v1/representantes/" + ana.ID.String() + "/fornecedor"
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPut, caminhoDoFornecedor, map[string]any{"fornecedor_id": fornecedor.ID}, tokenCaio), http.StatusForbidden)
	conferirStatus(t, envia(t, servidor, http.MethodPut, caminhoDoFornecedor, map[string]any{}), http.StatusBadRequest)
	resposta = envia(t, servidor, http.MethodPut, caminhoDoFornecedor, map[string]any{"fornecedor_id": fornecedor.ID})
	conferirStatus(t, resposta, http.StatusOK)
	if comFornecedor := decodificarEnvelope[clientedto.RepresentanteResponse](t, resposta).Dados; comFornecedor.FornecedorID == nil || *comFornecedor.FornecedorID != fornecedor.ID {
		t.Fatalf("representante com fornecedor = %+v", comFornecedor)
	}

	resposta = enviaComToken(t, servidor, http.MethodPost, "/api/v1/comissoes/fechamentos", fechar, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	resultado := decodificarEnvelope[comissaodto.ResultadoFechamentoResponse](t, resposta).Dados
	if len(resultado.Fechados) != 1 || len(resultado.Falhas) != 0 || resultado.Competencia != competencia {
		t.Fatalf("fechamento = %+v", resultado)
	}
	fechado := resultado.Fechados[0]
	vencimento := inicioDoMes.AddDate(0, 0, 14).Format("2006-01-02")
	if fechado.ComissoesCentavos != 180 || fechado.AjustesCentavos != -100 || fechado.TotalCentavos != 80 ||
		fechado.Vencimento != vencimento || fechado.StatusContaAPagar != "pendente_aprovacao" {
		t.Fatalf("fechado = %+v", fechado)
	}

	resposta = envia(t, servidor, http.MethodGet, "/api/v1/solicitacoes/"+fechado.SolicitacaoID.String(), nil)
	conferirStatus(t, resposta, http.StatusOK)
	if conta := decodificarEnvelope[solicitacaodto.SolicitacaoResponse](t, resposta).Dados; conta.Origem != "comissao" ||
		*conta.OrigemID != fechado.ID || conta.ValorCentavos != 80 || *conta.FornecedorID != fornecedor.ID ||
		*conta.CategoriaFinanceiraID != comissoes.ID || conta.Competencia != competencia || conta.Status != "pendente_aprovacao" {
		t.Fatalf("conta a pagar = %+v", conta)
	}

	resposta = enviaComToken(t, servidor, http.MethodPost, "/api/v1/comissoes/fechamentos", fechar, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if deNovo := decodificarEnvelope[comissaodto.ResultadoFechamentoResponse](t, resposta).Dados; len(deNovo.Fechados) != 0 || len(deNovo.Falhas) != 0 {
		t.Fatalf("fechar de novo sem saldo = %+v", deNovo)
	}
	fechar["representante_id"] = ana.ID
	conferirMensagem(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/comissoes/fechamentos", fechar, tokenFinanceiro),
		http.StatusBadRequest, "erro de validação: nada a pagar ao representante na competência")

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/comissoes/fechamentos", nil, tokenAna)
	conferirStatus(t, resposta, http.StatusOK)
	if daAna := decodificarPagina[comissaodto.FechamentoResponse](t, resposta).Dados; len(daAna) != 1 || daAna[0].ID != fechado.ID {
		t.Fatalf("fechamentos da Ana = %+v", daAna)
	}
	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/comissoes/fechamentos/"+fechado.ID.String(), nil, tokenAna)
	conferirStatus(t, resposta, http.StatusOK)
	if detalhe := decodificarEnvelope[comissaodto.FechamentoDetalheResponse](t, resposta).Dados; len(detalhe.Comissoes) != 1 ||
		len(detalhe.Ajustes) != 1 || detalhe.Comissoes[0].FechamentoID == nil || *detalhe.Comissoes[0].FechamentoID != fechado.ID {
		t.Fatalf("detalhe = %+v", detalhe)
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/comissoes/fechamentos/"+fechado.ID.String(), nil, tokenCaio), http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/comissoes?aberta=true", nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if abertas := decodificarPagina[comissaodto.ComissaoResponse](t, resposta).Dados; len(abertas) != 0 {
		t.Fatalf("comissões abertas = %+v", abertas)
	}
	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/comissoes/ajustes?aberto=false", nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if fechados := decodificarPagina[comissaodto.AjusteResponse](t, resposta).Dados; len(fechados) != 1 {
		t.Fatalf("ajustes fechados = %+v", fechados)
	}

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/comissoes/resumo?competencia="+competencia, nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	resumo := decodificarEnvelope[comissaodto.ResumoResponse](t, resposta).Dados
	if resumo.Competencia != competencia || resumo.TotalCentavos != 80 || len(resumo.Linhas) != 1 {
		t.Fatalf("resumo = %+v", resumo)
	}
	if linha := resumo.Linhas[0]; linha.RepresentanteID != ana.ID || linha.ComissoesCentavos != 180 || linha.AjustesCentavos != -100 ||
		linha.FechadoCentavos != 80 || linha.EmAbertoCentavos != 0 || linha.PagoCentavos != 0 || linha.RepresentanteNome != "Ana Representações" {
		t.Fatalf("linha do resumo = %+v", linha)
	}
	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/comissoes/resumo?competencia="+competencia, nil, tokenAna)
	conferirStatus(t, resposta, http.StatusOK)
	if daAna := decodificarEnvelope[comissaodto.ResumoResponse](t, resposta).Dados; len(daAna.Linhas) != 1 {
		t.Fatalf("resumo da Ana = %+v", daAna)
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/comissoes/resumo?competencia="+competencia, nil, tokenCaio), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/comissoes/resumo", nil, tokenFinanceiro), http.StatusBadRequest)
}
