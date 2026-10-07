//go:build e2e

package http

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	solicitacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/solicitacao"
	"github.com/google/uuid"
)

func enviaMultipart(t *testing.T, servidor *httptest.Server, caminho, nomeArquivo string, conteudo []byte, token string) *http.Response {
	t.Helper()

	corpo := &bytes.Buffer{}
	escritor := multipart.NewWriter(corpo)

	parte, err := escritor.CreateFormFile("arquivo", nomeArquivo)
	if err != nil {
		t.Fatalf("não montei a parte do arquivo: %v", err)
	}
	if _, err := parte.Write(conteudo); err != nil {
		t.Fatalf("não escrevi o arquivo: %v", err)
	}
	if err := escritor.Close(); err != nil {
		t.Fatalf("não fechei o multipart: %v", err)
	}

	requisicao, err := http.NewRequest(http.MethodPost, servidor.URL+caminho, corpo)
	if err != nil {
		t.Fatalf("não montei a requisição multipart: %v", err)
	}
	requisicao.Header.Set("Content-Type", escritor.FormDataContentType())
	if token != "" {
		requisicao.Header.Set("Authorization", "Bearer "+token)
	}

	resposta, err := servidor.Client().Do(requisicao)
	if err != nil {
		t.Fatalf("requisição multipart %s falhou: %v", caminho, err)
	}
	t.Cleanup(func() {
		_ = resposta.Body.Close()
	})

	return resposta
}

func TestFluxoCompletoDeSolicitacaoDePagamento(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	cargoOperador := criarCargo(t, servidor, "Operador da obra", false)
	cargoFinanceiro := criarCargoComPerfil(t, servidor, "Setor financeiro", false, true)

	criarUsuario(t, servidor, "Ana Souza", "ana.souza@exemplo.com", "senhaForte123", cargoOperador)
	criarUsuario(t, servidor, "Bruno Costa", "bruno.costa@exemplo.com", "senhaForte123", cargoFinanceiro)
	idAprovador := criarUsuario(t, servidor, "Carla Dias", "carla.dias@exemplo.com", "senhaForte123", cargoOperador)

	tokenSolicitante := autenticar(t, servidor, "ana.souza@exemplo.com", "senhaForte123").Token
	tokenFinanceiro := autenticar(t, servidor, "bruno.costa@exemplo.com", "senhaForte123").Token
	tokenAprovador := autenticar(t, servidor, "carla.dias@exemplo.com", "senhaForte123").Token

	conteudoOrcamento := []byte("%PDF-1.4 conteudo do orcamento")
	respostaUpload := enviaMultipart(t, servidor, "/api/v1/arquivos", "orcamento.pdf", conteudoOrcamento, tokenSolicitante)
	conferirStatus(t, respostaUpload, http.StatusCreated)
	arquivo := decodificarEnvelope[solicitacaodto.ArquivoResponse](t, respostaUpload).Dados
	if arquivo.Tamanho != int64(len(conteudoOrcamento)) {
		t.Fatalf("tamanho do arquivo = %d, esperado %d", arquivo.Tamanho, len(conteudoOrcamento))
	}

	prazo := time.Now().UTC().AddDate(0, 0, 10).Format("2006-01-02")
	respostaCriar := enviaComToken(t, servidor, http.MethodPost, "/api/v1/solicitacoes", map[string]any{
		"valor_centavos":  150000,
		"prazo_pagamento": prazo,
		"observacao":      "Compra de cimento para a fase 2",
		"forma_pagamento": "pix",
		"arquivo_ids":     []uuid.UUID{arquivo.ID},
	}, tokenSolicitante)
	conferirStatus(t, respostaCriar, http.StatusCreated)
	solicitacao := decodificarEnvelope[solicitacaodto.SolicitacaoResponse](t, respostaCriar).Dados
	if solicitacao.Status != "pendente_aprovacao" {
		t.Fatalf("status inicial = %q, esperado pendente_aprovacao", solicitacao.Status)
	}
	if solicitacao.ValorCentavos != 150000 || solicitacao.SolicitanteID == uuid.Nil {
		t.Errorf("solicitação criada = %+v, esperado 150000 com solicitante", solicitacao)
	}

	caminho := "/api/v1/solicitacoes/" + solicitacao.ID.String()

	respostaDesignacaoIndevida := enviaComToken(t, servidor, http.MethodPost, "/api/v1/aprovadores", map[string]any{
		"usuario_id": idAprovador,
	}, tokenSolicitante)
	conferirStatus(t, respostaDesignacaoIndevida, http.StatusForbidden)

	respostaDesignacao := envia(t, servidor, http.MethodPost, "/api/v1/aprovadores", map[string]any{
		"usuario_id": idAprovador,
	})
	conferirStatus(t, respostaDesignacao, http.StatusCreated)

	respostaDesignacaoRepetida := envia(t, servidor, http.MethodPost, "/api/v1/aprovadores", map[string]any{
		"usuario_id": idAprovador,
	})
	conferirStatus(t, respostaDesignacaoRepetida, http.StatusConflict)

	respostaSemPermissao := enviaComToken(t, servidor, http.MethodGet, "/api/v1/solicitacoes?escopo=aprovacao", nil, tokenSolicitante)
	conferirStatus(t, respostaSemPermissao, http.StatusForbidden)

	respostaAprovacaoIndevida := enviaComToken(t, servidor, http.MethodPost, caminho+"/aprovar", nil, tokenSolicitante)
	conferirStatus(t, respostaAprovacaoIndevida, http.StatusForbidden)

	respostaFila := enviaComToken(t, servidor, http.MethodGet, "/api/v1/solicitacoes?escopo=aprovacao", nil, tokenAprovador)
	conferirStatus(t, respostaFila, http.StatusOK)
	fila := decodificarPagina[solicitacaodto.SolicitacaoResponse](t, respostaFila)
	if len(fila.Dados) != 1 || fila.Dados[0].ID != solicitacao.ID {
		t.Fatalf("fila do aprovador = %+v, esperado a solicitação criada", fila.Dados)
	}

	respostaAprovar := enviaComToken(t, servidor, http.MethodPost, caminho+"/aprovar", nil, tokenAprovador)
	conferirStatus(t, respostaAprovar, http.StatusOK)
	aprovada := decodificarEnvelope[solicitacaodto.SolicitacaoResponse](t, respostaAprovar).Dados
	if aprovada.Status != "aprovado" || aprovada.AprovadorID == nil || *aprovada.AprovadorID != idAprovador {
		t.Fatalf("solicitação aprovada = %+v, esperado aprovada pelo aprovador designado", aprovada)
	}

	respostaFinanceiro := enviaComToken(t, servidor, http.MethodGet, "/api/v1/solicitacoes?escopo=financeiro", nil, tokenFinanceiro)
	conferirStatus(t, respostaFinanceiro, http.StatusOK)
	if pagina := decodificarPagina[solicitacaodto.SolicitacaoResponse](t, respostaFinanceiro); len(pagina.Dados) != 1 {
		t.Fatalf("fila do financeiro = %d, esperado 1", len(pagina.Dados))
	}

	conteudoComprovante := []byte("%PDF-1.4 comprovante do pagamento")
	respostaComprovante := enviaMultipart(t, servidor, "/api/v1/arquivos", "comprovante.pdf", conteudoComprovante, tokenFinanceiro)
	conferirStatus(t, respostaComprovante, http.StatusCreated)
	comprovante := decodificarEnvelope[solicitacaodto.ArquivoResponse](t, respostaComprovante).Dados

	respostaPagamento := enviaComToken(t, servidor, http.MethodPost, caminho+"/pagamento", map[string]any{
		"valor_centavos":         148500,
		"comprovante_arquivo_id": comprovante.ID,
	}, tokenFinanceiro)
	conferirStatus(t, respostaPagamento, http.StatusCreated)
	pagamento := decodificarEnvelope[solicitacaodto.PagamentoResponse](t, respostaPagamento).Dados
	if pagamento.ValorCentavos != 148500 {
		t.Errorf("valor pago = %d, esperado 148500", pagamento.ValorCentavos)
	}
	if pagamento.ComprovanteArquivoID == nil || *pagamento.ComprovanteArquivoID != comprovante.ID {
		t.Errorf("comprovante = %v, esperado %s", pagamento.ComprovanteArquivoID, comprovante.ID)
	}

	respostaSalva := enviaComToken(t, servidor, http.MethodGet, caminho, nil, tokenSolicitante)
	conferirStatus(t, respostaSalva, http.StatusOK)
	salva := decodificarEnvelope[solicitacaodto.SolicitacaoResponse](t, respostaSalva).Dados
	if salva.Status != "pago" {
		t.Errorf("status final = %q, esperado pago", salva.Status)
	}
	if salva.ValorCentavos != 150000 {
		t.Errorf("valor estimado = %d, esperado 150000", salva.ValorCentavos)
	}

	respostaPagamentoSalvo := enviaComToken(t, servidor, http.MethodGet, caminho+"/pagamento", nil, tokenFinanceiro)
	conferirStatus(t, respostaPagamentoSalvo, http.StatusOK)
	pagamentoSalvo := decodificarEnvelope[solicitacaodto.PagamentoResponse](t, respostaPagamentoSalvo).Dados
	if pagamentoSalvo.ID != pagamento.ID || pagamentoSalvo.ValorCentavos != 148500 {
		t.Errorf("pagamento consultado = %+v, esperado o registrado", pagamentoSalvo)
	}

	respostaHistorico := enviaComToken(t, servidor, http.MethodGet, caminho+"/historico", nil, tokenSolicitante)
	conferirStatus(t, respostaHistorico, http.StatusOK)
	historico := decodificarEnvelope[[]solicitacaodto.HistoricoResponse](t, respostaHistorico).Dados
	if len(historico) != 3 {
		t.Fatalf("histórico = %d registros, esperado 3", len(historico))
	}
	if historico[0].ParaStatus != "pago" || historico[2].ParaStatus != "pendente_aprovacao" {
		t.Errorf("ordem do histórico inesperada: %q ... %q", historico[0].ParaStatus, historico[2].ParaStatus)
	}

	respostaArquivos := enviaComToken(t, servidor, http.MethodGet, caminho+"/arquivos", nil, tokenSolicitante)
	conferirStatus(t, respostaArquivos, http.StatusOK)
	arquivosSolicitacao := decodificarEnvelope[[]solicitacaodto.ArquivoResponse](t, respostaArquivos).Dados
	if len(arquivosSolicitacao) != 2 {
		t.Fatalf("arquivos da solicitação = %d, esperado 2 (anexo + comprovante)", len(arquivosSolicitacao))
	}
	if arquivosSolicitacao[0].ID != arquivo.ID || arquivosSolicitacao[1].ID != comprovante.ID {
		t.Errorf("arquivos da solicitação = %+v, esperado o anexo e depois o comprovante", arquivosSolicitacao)
	}

	respostaDownload := enviaComToken(t, servidor, http.MethodGet, "/api/v1/arquivos/"+arquivo.ID.String(), nil, tokenAprovador)
	conferirStatus(t, respostaDownload, http.StatusOK)
	if disponivel := respostaDownload.Header.Get("Content-Disposition"); !strings.Contains(disponivel, "orcamento.pdf") {
		t.Errorf("content-disposition %q sem o nome do arquivo", disponivel)
	}
	conteudoBaixado, err := io.ReadAll(respostaDownload.Body)
	if err != nil {
		t.Fatalf("não li o arquivo baixado: %v", err)
	}
	if !bytes.Equal(conteudoBaixado, conteudoOrcamento) {
		t.Errorf("conteúdo baixado = %q, esperado %q", conteudoBaixado, conteudoOrcamento)
	}

	respostaRemocaoVinculada := enviaComToken(t, servidor, http.MethodDelete, "/api/v1/arquivos/"+arquivo.ID.String(), nil, tokenSolicitante)
	conferirStatus(t, respostaRemocaoVinculada, http.StatusConflict)

	respostaRemocaoComprovante := enviaComToken(t, servidor, http.MethodDelete, "/api/v1/arquivos/"+comprovante.ID.String(), nil, tokenFinanceiro)
	conferirStatus(t, respostaRemocaoComprovante, http.StatusConflict)
	if erro := decodificarErro(t, respostaRemocaoComprovante); !strings.Contains(erro.Erro.Mensagem, "vinculado a uma solicitação") {
		t.Errorf("mensagem %q sem a regra de vínculo", erro.Erro.Mensagem)
	}

	respostaCancelamento := enviaComToken(t, servidor, http.MethodPost, caminho+"/cancelar", nil, tokenSolicitante)
	conferirStatus(t, respostaCancelamento, http.StatusConflict)

	respostaTodas := enviaComToken(t, servidor, http.MethodGet, "/api/v1/solicitacoes?escopo=todas", nil, tokenFinanceiro)
	conferirStatus(t, respostaTodas, http.StatusForbidden)
}

func criarSolicitacaoBase(t *testing.T, servidor *httptest.Server, token string) uuid.UUID {
	t.Helper()

	resposta := enviaComToken(t, servidor, http.MethodPost, "/api/v1/solicitacoes", map[string]any{
		"valor_centavos":  99000,
		"prazo_pagamento": time.Now().UTC().AddDate(0, 0, 5).Format("2006-01-02"),
		"observacao":      "Compra de areia para a alvenaria",
		"forma_pagamento": "boleto",
	}, token)
	conferirStatus(t, resposta, http.StatusCreated)

	return decodificarEnvelope[solicitacaodto.SolicitacaoResponse](t, resposta).Dados.ID
}

func usuariosParaSolicitacao(t *testing.T, servidor *httptest.Server, apelido string) (solicitante, aprovador string) {
	t.Helper()

	cargo := criarCargo(t, servidor, "Operador "+apelido, false)
	criarUsuario(t, servidor, "Solicitante "+apelido, "solicitante."+apelido+"@exemplo.com", "senhaForte123", cargo)
	idAprovador := criarUsuario(t, servidor, "Aprovador "+apelido, "aprovador."+apelido+"@exemplo.com", "senhaForte123", cargo)

	tokenSolicitante := autenticar(t, servidor, "solicitante."+apelido+"@exemplo.com", "senhaForte123").Token
	tokenAprovador := autenticar(t, servidor, "aprovador."+apelido+"@exemplo.com", "senhaForte123").Token

	respostaDesignacao := envia(t, servidor, http.MethodPost, "/api/v1/aprovadores", map[string]any{
		"usuario_id": idAprovador,
	})
	conferirStatus(t, respostaDesignacao, http.StatusCreated)

	return tokenSolicitante, tokenAprovador
}

func TestCriarSolicitacaoInvalidaERetornoInexistente(t *testing.T) {
	servidor, _ := servidorDoTeste(t)

	cargo := criarCargo(t, servidor, "Operador de validação", false)
	criarUsuario(t, servidor, "Fábio Neves", "fabio.neves@exemplo.com", "senhaForte123", cargo)
	token := autenticar(t, servidor, "fabio.neves@exemplo.com", "senhaForte123").Token

	respostaPrazo := enviaComToken(t, servidor, http.MethodPost, "/api/v1/solicitacoes", map[string]any{
		"valor_centavos":  1000,
		"prazo_pagamento": "2001-01-01",
		"forma_pagamento": "pix",
	}, token)
	conferirStatus(t, respostaPrazo, http.StatusBadRequest)
	if erro := decodificarErro(t, respostaPrazo); erro.Erro.Codigo != http.StatusBadRequest {
		t.Errorf("código do envelope %d, esperado %d", erro.Erro.Codigo, http.StatusBadRequest)
	}

	respostaForma := enviaComToken(t, servidor, http.MethodPost, "/api/v1/solicitacoes", map[string]any{
		"valor_centavos":  1000,
		"prazo_pagamento": time.Now().UTC().AddDate(0, 0, 5).Format("2006-01-02"),
		"forma_pagamento": "dinheiro",
	}, token)
	conferirStatus(t, respostaForma, http.StatusBadRequest)

	respostaInexistente := enviaComToken(t, servidor, http.MethodGet, "/api/v1/solicitacoes/"+uuid.NewString(), nil, token)
	conferirStatus(t, respostaInexistente, http.StatusNotFound)
	if erro := decodificarErro(t, respostaInexistente); !strings.HasPrefix(erro.Erro.Mensagem, "registro não encontrado") {
		t.Errorf("mensagem %q sem o marcador registro não encontrado", erro.Erro.Mensagem)
	}
}

func TestListarSolicitacoesDevolvePaginacao(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	tokenSolicitante, _ := usuariosParaSolicitacao(t, servidor, "paginacao")

	criarSolicitacaoBase(t, servidor, tokenSolicitante)
	criarSolicitacaoBase(t, servidor, tokenSolicitante)

	resposta := enviaComToken(t, servidor, http.MethodGet,
		"/api/v1/solicitacoes?escopo=minhas&pagina=1&tamanho=1", nil, tokenSolicitante)
	conferirStatus(t, resposta, http.StatusOK)

	pagina := decodificarPagina[solicitacaodto.SolicitacaoResponse](t, resposta)
	if len(pagina.Dados) != 1 {
		t.Errorf("%d solicitações na página, esperado 1", len(pagina.Dados))
	}
	if pagina.Pagina != 1 || pagina.Tamanho != 1 {
		t.Errorf("página = {%d, %d}, esperada {1, 1}", pagina.Pagina, pagina.Tamanho)
	}
}

func TestFiltroDeStatusNaoUltrapassaEscopo(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	tokenSolicitante, tokenAprovador := usuariosParaSolicitacao(t, servidor, "escopo")

	cargoFinanceiro := criarCargoComPerfil(t, servidor, "Financeiro do escopo", false, true)
	criarUsuario(t, servidor, "Iara Lopes", "iara.lopes@exemplo.com", "senhaForte123", cargoFinanceiro)
	tokenFinanceiro := autenticar(t, servidor, "iara.lopes@exemplo.com", "senhaForte123").Token

	criarSolicitacaoBase(t, servidor, tokenSolicitante)

	respostaForaDoEscopo := enviaComToken(t, servidor, http.MethodGet,
		"/api/v1/solicitacoes?escopo=aprovacao&status=pago", nil, tokenAprovador)
	conferirStatus(t, respostaForaDoEscopo, http.StatusBadRequest)
	if erro := decodificarErro(t, respostaForaDoEscopo); !strings.Contains(erro.Erro.Mensagem, "status não permitido") {
		t.Errorf("mensagem %q sem a regra de escopo", erro.Erro.Mensagem)
	}

	respostaFila := enviaComToken(t, servidor, http.MethodGet,
		"/api/v1/solicitacoes?escopo=aprovacao&status=pendente_aprovacao", nil, tokenAprovador)
	conferirStatus(t, respostaFila, http.StatusOK)

	respostaFinanceiroPendente := enviaComToken(t, servidor, http.MethodGet,
		"/api/v1/solicitacoes?escopo=financeiro&status=pendente_aprovacao", nil, tokenFinanceiro)
	conferirStatus(t, respostaFinanceiroPendente, http.StatusBadRequest)

	respostaFinanceiroAprovado := enviaComToken(t, servidor, http.MethodGet,
		"/api/v1/solicitacoes?escopo=financeiro&status=aprovado", nil, tokenFinanceiro)
	conferirStatus(t, respostaFinanceiroAprovado, http.StatusOK)
}

func TestRejeitarSolicitacaoERemoverAprovador(t *testing.T) {
	servidor, _ := servidorDoTeste(t)
	tokenSolicitante, tokenAprovador := usuariosParaSolicitacao(t, servidor, "rejeicao")

	solicitacaoID := criarSolicitacaoBase(t, servidor, tokenSolicitante)
	caminho := "/api/v1/solicitacoes/" + solicitacaoID.String()

	respostaSemMotivo := enviaComToken(t, servidor, http.MethodPost, caminho+"/rejeitar", map[string]any{}, tokenAprovador)
	conferirStatus(t, respostaSemMotivo, http.StatusBadRequest)

	respostaRejeitar := enviaComToken(t, servidor, http.MethodPost, caminho+"/rejeitar",
		map[string]any{"motivo": "fora do orçamento do trimestre"}, tokenAprovador)
	conferirStatus(t, respostaRejeitar, http.StatusOK)

	rejeitada := decodificarEnvelope[solicitacaodto.SolicitacaoResponse](t, respostaRejeitar).Dados
	if rejeitada.Status != "rejeitado" {
		t.Errorf("status = %q, esperado rejeitado", rejeitada.Status)
	}
	if rejeitada.MotivoRejeicao == nil || *rejeitada.MotivoRejeicao != "fora do orçamento do trimestre" {
		t.Errorf("motivo = %v, esperado o enviado no corpo", rejeitada.MotivoRejeicao)
	}

	respostaFilaVazia := enviaComToken(t, servidor, http.MethodGet,
		"/api/v1/solicitacoes?escopo=aprovacao", nil, tokenAprovador)
	conferirStatus(t, respostaFilaVazia, http.StatusOK)
	if fila := decodificarPagina[solicitacaodto.SolicitacaoResponse](t, respostaFilaVazia); len(fila.Dados) != 0 {
		t.Errorf("fila da aprovação = %d, esperado vazia após a rejeição", len(fila.Dados))
	}

	aprovadores := decodificarPagina[solicitacaodto.AprovadorResponse](
		t, envia(t, servidor, http.MethodGet, "/api/v1/aprovadores", nil),
	)
	if len(aprovadores.Dados) == 0 {
		t.Fatal("nenhum aprovador designado para remover")
	}
	aprovadorID := aprovadores.Dados[0].ID

	conferirStatus(t, envia(t, servidor, http.MethodDelete, "/api/v1/aprovadores/"+aprovadorID.String(), nil),
		http.StatusNoContent)
	conferirStatus(t, envia(t, servidor, http.MethodDelete, "/api/v1/aprovadores/"+aprovadorID.String(), nil),
		http.StatusNotFound)
}
