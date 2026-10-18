//go:build e2e

package http

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	notafiscaldto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/notafiscal"
	pedidodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/pedido"
	receberdto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/receber"
	solicitacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/solicitacao"
	"github.com/CunhazadanoDale/paineladministrativo.git/test/fixtures/nfe"
	"github.com/google/uuid"
)

const (
	cnpjDaMia      = "11222333000181"
	cnpjDaRed      = "33400004000116"
	cnpjDoCliente  = "11444777000161"
	cpfDeClienteNF = "52998224725"
)

func enviarXML(t *testing.T, servidor *httptest.Server, conteudo []byte, token string) uuid.UUID {
	t.Helper()

	resposta := enviaMultipart(t, servidor, "/api/v1/arquivos", "nota.xml", conteudo, token)
	conferirStatus(t, resposta, http.StatusCreated)
	arquivo := decodificarEnvelope[solicitacaodto.ArquivoResponse](t, resposta).Dados
	if arquivo.ContentType != "application/xml" {
		t.Fatalf("XML gravado como %q", arquivo.ContentType)
	}

	return arquivo.ID
}

func importarNota(t *testing.T, servidor *httptest.Server, corpo map[string]any, token string) notafiscaldto.ImportacaoResponse {
	t.Helper()

	resposta := enviaComToken(t, servidor, http.MethodPost, "/api/v1/notas-fiscais/importacao", corpo, token)
	conferirStatus(t, resposta, http.StatusCreated)

	return decodificarEnvelope[notafiscaldto.ImportacaoResponse](t, resposta).Dados
}

func titulosDaDuplicata(t *testing.T, servidor *httptest.Server, duplicataID uuid.UUID, token string) []receberdto.TituloResponse {
	t.Helper()

	resposta := enviaComToken(t, servidor, http.MethodGet, "/api/v1/titulos-receber?origem=nota_fiscal_duplicata&origem_id="+duplicataID.String(), nil, token)
	conferirStatus(t, resposta, http.StatusOK)

	return decodificarPagina[receberdto.TituloResponse](t, resposta).Dados
}

func TestFluxoDeNotaFiscalComPedidoEntregaECancelamento(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	red := criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond").ID
	galpao := criarLocal(t, servidor, mia, "Galpão")
	raiz := criarCategoria(t, servidor, "Revestimentos", nil)
	sub := criarCategoria(t, servidor, "Pisos", &raiz)
	pisoID := criarProdutoComCodigo(t, servidor, sub, "Piso 60x60", "PISO-01")
	rejunteID := criarProdutoComCodigo(t, servidor, sub, "Rejunte", "REJ-INTERNO")
	conferirStatus(t, envia(t, servidor, http.MethodPost, "/api/v1/produtos/"+pisoID.String()+"/movimentos", map[string]any{
		"local_id": galpao.ID, "tipo": "entrada", "quantidade": 20, "custo_unitario_centavos": 1990,
	}), http.StatusCreated)
	clienteID := clienteDoPedido(t, servidor)

	pedido := criarPedidoE2E(t, servidor, corpoDePedido(mia, clienteID, pisoID, "pronta_entrega", 10), tokenDoTeste)
	caminhoDoPedido := "/api/v1/pedidos/" + pedido.ID.String()
	conferirStatus(t, envia(t, servidor, http.MethodPost, caminhoDoPedido+"/confirmar", map[string]any{"versao": 1}), http.StatusOK)
	pedido = decodificarEnvelope[pedidodto.PedidoResponse](t, envia(t, servidor, http.MethodGet, caminhoDoPedido, nil)).Dados
	resposta := envia(t, servidor, http.MethodPost, caminhoDoPedido+"/entregas", map[string]any{
		"frete_tipo": "fob",
		"itens":      []map[string]any{{"pedido_item_id": pedido.Itens[0].ID, "local_id": galpao.ID, "quantidade": 10}},
	})
	conferirStatus(t, resposta, http.StatusCreated)
	entrega := decodificarEnvelope[pedidodto.EntregaResponse](t, resposta).Dados
	conferirStatus(t, envia(t, servidor, http.MethodPost, "/api/v1/entregas/"+entrega.ID.String()+"/saiu", map[string]any{"versao": 1}), http.StatusOK)

	tokenFinanceiro := tokenFinanceiroDasEmpresas(t, servidor, "Fabia Fin", "fabia@mia.exemplo.com", mia)
	tokenFinanceiroRed := tokenFinanceiroDasEmpresas(t, servidor, "Rita Red", "rita@red.exemplo.com", red)
	tokenComercial := usuarioComercialDasEmpresas(t, servidor, "Caio Com", "caio@mia.exemplo.com", mia)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/entregas/sem-nota-fiscal", nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if semNota := decodificarPagina[notafiscaldto.EntregaSemNotaResponse](t, resposta).Dados; len(semNota) != 1 || semNota[0].EntregaID != entrega.ID {
		t.Fatalf("entregas sem nota = %+v", semNota)
	}

	documento := nfe.Nova(cnpjDaMia, cnpjDoCliente, 123)
	arquivoID := enviarXML(t, servidor, documento.XML(), tokenFinanceiro)
	corpo := map[string]any{"arquivo_id": arquivoID, "pedido_id": pedido.ID, "entrega_id": entrega.ID}

	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/notas-fiscais/importacao", corpo, tokenComercial), http.StatusForbidden)

	importada := importarNota(t, servidor, corpo, tokenFinanceiro)
	if importada.Chave != documento.Chave() || importada.EmpresaID != mia || importada.ClienteID == nil || *importada.ClienteID != clienteID ||
		importada.Natureza != "venda" || importada.Status != "autorizada" || importada.PedidoID == nil || *importada.PedidoID != pedido.ID ||
		importada.EntregaID == nil || *importada.EntregaID != entrega.ID || importada.ValorTotalCentavos != 109000 || importada.Versao != 1 {
		t.Fatalf("nota importada = %+v", importada.NotaFiscalResponse)
	}
	if len(importada.Itens) != 2 || importada.Itens[0].ProdutoID == nil || *importada.Itens[0].ProdutoID != pisoID ||
		importada.Itens[0].Quantidade != "10.5000" || importada.Itens[1].ProdutoID != nil || len(importada.Duplicatas) != 2 {
		t.Errorf("itens e duplicatas = %+v %+v", importada.Itens, importada.Duplicatas)
	}
	if !slices.Equal(importada.Avisos, []string{"itens_sem_produto"}) {
		t.Errorf("avisos = %v", importada.Avisos)
	}
	caminho := "/api/v1/notas-fiscais/" + importada.ID.String()

	if len(importada.TitulosGerados) != 2 {
		t.Fatalf("títulos gerados = %v", importada.TitulosGerados)
	}
	titulos := titulosDaDuplicata(t, servidor, importada.Duplicatas[0].ID, tokenFinanceiro)
	if len(titulos) != 1 || titulos[0].Numero != "NF 1-123" || titulos[0].Parcela != 1 || titulos[0].ValorCentavos != 54500 ||
		titulos[0].Status != "carteira" || titulos[0].ClienteID != clienteID || titulos[0].EmpresaID != mia {
		t.Errorf("título da primeira duplicata = %+v", titulos)
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, caminho+"/titulos", nil, tokenComercial), http.StatusForbidden)
	resposta = enviaComToken(t, servidor, http.MethodPost, caminho+"/titulos", nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if sincronizacao := decodificarEnvelope[notafiscaldto.SincronizacaoTitulosResponse](t, resposta).Dados; len(sincronizacao.Gerados) != 0 || len(sincronizacao.JaExistentes) != 2 {
		t.Errorf("sincronizar títulos já gerados = %+v", sincronizacao)
	}

	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/notas-fiscais/importacao", map[string]any{"arquivo_id": arquivoID}, tokenFinanceiro), http.StatusConflict)
	repetida := enviarXML(t, servidor, documento.XML(), tokenFinanceiro)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/notas-fiscais/importacao", map[string]any{"arquivo_id": repetida}, tokenFinanceiro), http.StatusConflict)
	alheio := enviarXML(t, servidor, nfe.Nova(cnpjDaMia, cnpjDoCliente, 124).XML(), tokenDoTeste)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/notas-fiscais/importacao", map[string]any{"arquivo_id": alheio}, tokenFinanceiro), http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/notas-fiscais?pedido="+pedido.ID.String(), nil, tokenComercial)
	conferirStatus(t, resposta, http.StatusOK)
	if notas := decodificarPagina[notafiscaldto.NotaFiscalResponse](t, resposta).Dados; len(notas) != 1 || notas[0].ID != importada.ID {
		t.Errorf("notas do pedido = %+v", notas)
	}
	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/notas-fiscais", nil, tokenFinanceiroRed)
	conferirStatus(t, resposta, http.StatusOK)
	if daRed := decodificarPagina[notafiscaldto.NotaFiscalResponse](t, resposta).Dados; len(daRed) != 0 {
		t.Errorf("Red viu %d notas da MIA", len(daRed))
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, caminho, nil, tokenFinanceiroRed), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, caminho+"/xml", nil, tokenFinanceiroRed), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/notas-fiscais?empresa="+mia.String(), nil, tokenFinanceiroRed), http.StatusForbidden)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, "/api/v1/notas-fiscais?natureza=bonificacao", nil, tokenFinanceiro), http.StatusBadRequest)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/entregas/sem-nota-fiscal", nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if semNota := decodificarPagina[notafiscaldto.EntregaSemNotaResponse](t, resposta).Dados; len(semNota) != 0 {
		t.Errorf("entrega com nota continua sem nota: %+v", semNota)
	}

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/notas-fiscais/itens-sem-produto", nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if pendentes := decodificarPagina[notafiscaldto.ItemPendenteResponse](t, resposta).Dados; len(pendentes) != 1 || pendentes[0].Item.Codigo != "REJ-02" {
		t.Errorf("itens sem produto = %+v", pendentes)
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPut, caminho+"/itens/2/produto", map[string]any{"versao": 1, "produto_id": rejunteID}, tokenComercial), http.StatusForbidden)
	resposta = enviaComToken(t, servidor, http.MethodPut, caminho+"/itens/2/produto", map[string]any{"versao": 1, "produto_id": rejunteID}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if ligada := decodificarEnvelope[notafiscaldto.NotaFiscalDetalheResponse](t, resposta).Dados; ligada.Versao != 2 || ligada.Itens[1].ProdutoID == nil {
		t.Errorf("item ligado = %+v", ligada.Itens[1])
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPut, caminho+"/itens/2/produto", map[string]any{"versao": 1, "produto_id": rejunteID}, tokenFinanceiro), http.StatusConflict)

	resposta = enviaComToken(t, servidor, http.MethodGet, caminho+"/xml", nil, tokenComercial)
	conferirStatus(t, resposta, http.StatusOK)
	if corpoXML := lerCorpo(t, resposta); string(corpoXML) != string(documento.XML()) ||
		resposta.Header.Get("X-Content-Type-Options") != "nosniff" || resposta.Header.Get("Content-Disposition") == "" {
		t.Errorf("download do XML = %d bytes, cabeçalhos %v", len(corpoXML), resposta.Header)
	}

	cancelamento := enviarXML(t, servidor, nfe.Cancelamento(documento.Chave(), cnpjDaMia).XML(), tokenFinanceiro)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/notas-fiscais/eventos", map[string]any{"arquivo_id": cancelamento}, tokenComercial), http.StatusForbidden)
	resposta = enviaComToken(t, servidor, http.MethodPost, "/api/v1/notas-fiscais/eventos", map[string]any{"arquivo_id": cancelamento}, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusCreated)
	if evento := decodificarEnvelope[notafiscaldto.ImportacaoEventoResponse](t, resposta).Dados; evento.Evento.Tipo != "cancelamento" ||
		evento.NotaFiscal.Status != "cancelada" || evento.NotaFiscal.Versao != 3 || len(evento.TitulosCancelados) != 2 || len(evento.Avisos) != 0 {
		t.Errorf("evento importado = %+v", evento)
	}
	if titulos := titulosDaDuplicata(t, servidor, importada.Duplicatas[0].ID, tokenFinanceiro); len(titulos) != 1 || titulos[0].Status != "cancelado" {
		t.Errorf("título depois do cancelamento da NF = %+v", titulos)
	}

	resposta = enviaComToken(t, servidor, http.MethodGet, caminho+"/eventos", nil, tokenComercial)
	conferirStatus(t, resposta, http.StatusOK)
	if eventos := decodificarEnvelope[[]notafiscaldto.EventoResponse](t, resposta).Dados; len(eventos) != 1 {
		t.Errorf("eventos = %+v", eventos)
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodGet, caminho+"/eventos", nil, tokenFinanceiroRed), http.StatusForbidden)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/entregas/sem-nota-fiscal", nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if semNota := decodificarPagina[notafiscaldto.EntregaSemNotaResponse](t, resposta).Dados; len(semNota) != 1 {
		t.Errorf("entrega volta a ficar sem nota depois do cancelamento: %+v", semNota)
	}
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPut, caminho+"/vinculo", map[string]any{"versao": 3}, tokenFinanceiro), http.StatusConflict)
}

func TestNotaFiscalEntreEmpresasEClienteCriado(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	mia := criarEmpresa(t, servidor, "11.222.333/0001-81", "MIA").ID
	red := criarEmpresa(t, servidor, "33.400.004/0001-16", "Red Diamond").ID
	tokenFinanceiro := tokenFinanceiroDasEmpresas(t, servidor, "Fabia Fin", "fabia@mia.exemplo.com", mia)

	transferencia := importarNota(t, servidor, map[string]any{
		"arquivo_id": enviarXML(t, servidor, nfe.Nova(cnpjDaMia, cnpjDaRed, 10).XML(), tokenFinanceiro),
	}, tokenFinanceiro)
	if !transferencia.EntreEmpresas || transferencia.Natureza != "transferencia" || transferencia.ClienteID != nil ||
		transferencia.EmpresaDestinatariaID == nil || *transferencia.EmpresaDestinatariaID != red || len(transferencia.Duplicatas) != 2 ||
		len(transferencia.TitulosGerados) != 0 {
		t.Errorf("transferência = %+v", transferencia.NotaFiscalResponse)
	}

	paraPessoa := nfe.Nova(cnpjDaMia, cpfDeClienteNF, 11)
	paraPessoa.NomeDestinatario = "Maria da Silva"
	venda := importarNota(t, servidor, map[string]any{"arquivo_id": enviarXML(t, servidor, paraPessoa.XML(), tokenFinanceiro)}, tokenFinanceiro)
	if venda.ClienteID == nil || !slices.Contains(venda.Avisos, "cliente_criado") || venda.DestinatarioDocumento != cpfDeClienteNF {
		t.Fatalf("venda para cliente novo = %+v avisos %v", venda.NotaFiscalResponse, venda.Avisos)
	}
	resposta := envia(t, servidor, http.MethodGet, "/api/v1/clientes/"+venda.ClienteID.String(), nil)
	conferirStatus(t, resposta, http.StatusOK)

	resposta = enviaComToken(t, servidor, http.MethodGet, "/api/v1/notas-fiscais?entre_empresas=true", nil, tokenFinanceiro)
	conferirStatus(t, resposta, http.StatusOK)
	if notas := decodificarPagina[notafiscaldto.NotaFiscalResponse](t, resposta).Dados; len(notas) != 1 || notas[0].ID != transferencia.ID {
		t.Errorf("notas entre empresas = %+v", notas)
	}

	semProtocolo := []byte(`<?xml version="1.0"?><NFe xmlns="http://www.portalfiscal.inf.br/nfe"></NFe>`)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/notas-fiscais/"+transferencia.ID.String()+"/titulos", nil, tokenFinanceiro), http.StatusBadRequest)
	if titulos := titulosDaDuplicata(t, servidor, transferencia.Duplicatas[0].ID, tokenFinanceiro); len(titulos) != 0 {
		t.Errorf("transferência gerou títulos: %+v", titulos)
	}

	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/notas-fiscais/importacao", map[string]any{
		"arquivo_id": enviarXML(t, servidor, semProtocolo, tokenFinanceiro),
	}, tokenFinanceiro), http.StatusBadRequest)
	conferirStatus(t, enviaComToken(t, servidor, http.MethodPost, "/api/v1/notas-fiscais/importacao", map[string]any{
		"arquivo_id": enviarXML(t, servidor, nfe.Nova(cnpjDaRed, cnpjDoCliente, 12).XML(), tokenFinanceiro),
	}, tokenFinanceiro), http.StatusForbidden)
}
