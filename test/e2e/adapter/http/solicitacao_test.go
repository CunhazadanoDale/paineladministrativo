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

	respostaPagamento := enviaComToken(t, servidor, http.MethodPost, caminho+"/pagamentos", map[string]any{
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
	if len(arquivosSolicitacao) != 1 || arquivosSolicitacao[0].ID != arquivo.ID {
		t.Errorf("arquivos da solicitação = %+v, esperado apenas o anexo enviado", arquivosSolicitacao)
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

	respostaCancelamento := enviaComToken(t, servidor, http.MethodPost, caminho+"/cancelar", nil, tokenSolicitante)
	conferirStatus(t, respostaCancelamento, http.StatusConflict)

	respostaTodas := enviaComToken(t, servidor, http.MethodGet, "/api/v1/solicitacoes?escopo=todas", nil, tokenFinanceiro)
	conferirStatus(t, respostaTodas, http.StatusForbidden)
}
