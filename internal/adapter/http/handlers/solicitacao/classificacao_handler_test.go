package solicitacao

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	solicitacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/solicitacao"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

func TestCriarSolicitacaoRepassaClassificacaoOpcional(t *testing.T) {
	usecase := &solicitacaoUseCaseFalso{solicitacao: novaSolicitacaoDeTeste()}
	fornecedorID := uuid.New()
	categoriaID := uuid.New()

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).Criar, http.MethodPost, "/api/v1/solicitacoes", "", `{
		"valor_centavos": 150000,
		"prazo_pagamento": "2030-01-15",
		"observacao": "Aluguel",
		"forma_pagamento": "debito_automatico",
		"fornecedor_id": "`+fornecedorID.String()+`",
		"categoria_financeira_id": "`+categoriaID.String()+`",
		"competencia": "2030-01",
		"estimado": true
	}`)

	if resposta.Code != http.StatusCreated {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusCreated, resposta.Body.String())
	}
	entrada := usecase.inputCriar
	if entrada.FornecedorID == nil || *entrada.FornecedorID != fornecedorID {
		t.Errorf("fornecedor = %v, esperado %s", entrada.FornecedorID, fornecedorID)
	}
	if entrada.CategoriaFinanceiraID == nil || *entrada.CategoriaFinanceiraID != categoriaID {
		t.Errorf("categoria = %v, esperado %s", entrada.CategoriaFinanceiraID, categoriaID)
	}
	if entrada.Competencia != "2030-01" || !entrada.Estimado {
		t.Errorf("competência = %q, estimado = %v", entrada.Competencia, entrada.Estimado)
	}
}

func TestClassificarRepassaCorpoERespondeASolicitacaoAtualizada(t *testing.T) {
	solicitacao := novaSolicitacaoDeTeste()
	categoriaID := uuid.New()
	solicitacao.CategoriaFinanceiraID = &categoriaID
	solicitacao.Competencia = domainsolicitacao.CompetenciaDe(time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC))
	solicitacao.Origem = domainsolicitacao.OrigemManual
	solicitacao.Versao = 3
	usecase := &solicitacaoUseCaseFalso{solicitacao: solicitacao}
	id := uuid.New()

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).Classificar, http.MethodPatch,
		"/api/v1/solicitacoes/"+id.String()+"/classificacao", id.String(),
		`{"categoria_financeira_id": "`+categoriaID.String()+`", "competencia": "2026-07", "versao": 2}`)

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	entrada := usecase.inputClassificar
	if entrada.SolicitacaoID != id || entrada.UsuarioID == uuid.Nil || entrada.Versao != 2 {
		t.Errorf("entrada = %+v, esperado id do caminho, usuário autenticado e versão 2", entrada)
	}
	if entrada.FornecedorID != nil || entrada.CategoriaFinanceiraID == nil || entrada.Competencia != "2026-07" {
		t.Errorf("classificação repassada = %+v", entrada)
	}

	var corpo dto.Resposta[solicitacaodto.SolicitacaoResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	if corpo.Dados.Competencia != "2026-07" || corpo.Dados.Versao != 3 || corpo.Dados.Origem != "manual" {
		t.Errorf("resposta = %+v", corpo.Dados)
	}
	if corpo.Dados.CategoriaFinanceiraID == nil || *corpo.Dados.CategoriaFinanceiraID != categoriaID {
		t.Errorf("categoria na resposta = %v", corpo.Dados.CategoriaFinanceiraID)
	}
}

func TestClassificarComVersaoDivergenteResponde409(t *testing.T) {
	usecase := &solicitacaoUseCaseFalso{erro: domainsolicitacao.ErroSolicitacaoAlterada()}
	id := uuid.NewString()

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).Classificar, http.MethodPatch,
		"/api/v1/solicitacoes/"+id+"/classificacao", id, `{"versao": 1}`)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusConflict)
	if apoioteste.MensagemDoEnvelope(t, resposta) != "solicitação alterada por outra operação, recarregue e tente novamente" {
		t.Errorf("mensagem = %q", apoioteste.MensagemDoEnvelope(t, resposta))
	}
}

func TestClassificarComVersaoQueNaoEhInteiroResponde400(t *testing.T) {
	usecase := &solicitacaoUseCaseFalso{solicitacao: novaSolicitacaoDeTeste()}
	id := uuid.NewString()

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).Classificar, http.MethodPatch,
		"/api/v1/solicitacoes/"+id+"/classificacao", id, `{"versao": "2"}`)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
	if usecase.inputClassificar.SolicitacaoID != uuid.Nil {
		t.Error("não deveria chegar ao usecase com corpo inválido")
	}
}

func TestListarSolicitacoesRepassaFiltrosDeContasAPagar(t *testing.T) {
	usecase := &solicitacaoUseCaseFalso{solicitacao: novaSolicitacaoDeTeste()}
	fornecedorID := uuid.New()
	categoriaID := uuid.New()

	resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).Listar, http.MethodGet,
		"/api/v1/solicitacoes?escopo=financeiro&fornecedor="+fornecedorID.String()+"&categoria="+categoriaID.String()+
			"&competencia=2026-10&origem=manual&vencimento_de=2026-10-01&vencimento_ate=2026-10-31&vencidas=true", "", "")

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	entrada := usecase.inputListar
	if entrada.Fornecedor == nil || *entrada.Fornecedor != fornecedorID || entrada.Categoria == nil || *entrada.Categoria != categoriaID {
		t.Errorf("fornecedor/categoria = %v/%v", entrada.Fornecedor, entrada.Categoria)
	}
	if entrada.Competencia != "2026-10" || entrada.Origem != "manual" || !entrada.Vencidas {
		t.Errorf("competência = %q, origem = %q, vencidas = %v", entrada.Competencia, entrada.Origem, entrada.Vencidas)
	}
	inicio := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
	fim := time.Date(2026, time.October, 31, 0, 0, 0, 0, time.UTC)
	if entrada.VenceDe == nil || !entrada.VenceDe.Equal(inicio) || entrada.VenceAte == nil || !entrada.VenceAte.Equal(fim) {
		t.Errorf("vencimento = %v a %v", entrada.VenceDe, entrada.VenceAte)
	}
}

func TestListarSolicitacoesComConsultaInvalidaResponde400(t *testing.T) {
	for _, consulta := range []string{"fornecedor=abc", "categoria=abc", "vencimento_de=01/10/2026", "vencimento_ate=2026-13-01"} {
		usecase := &solicitacaoUseCaseFalso{solicitacao: novaSolicitacaoDeTeste()}

		resposta := executaProtegido(t, NewSolicitacaoHandler(usecase).Listar, http.MethodGet,
			"/api/v1/solicitacoes?"+consulta, "", "")

		apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
		if usecase.inputListar.UsuarioID != uuid.Nil {
			t.Errorf("%s chegou ao usecase", consulta)
		}
	}
}

func TestRespostaDaSolicitacaoMarcaVencidaEmAberto(t *testing.T) {
	solicitacao := novaSolicitacaoDeTeste()
	solicitacao.Prazo = domainsolicitacao.PrazoPagamentoDe(time.Now().UTC().AddDate(0, 0, -1))

	if !solicitacaodto.NovaSolicitacaoResponse(solicitacao).Vencida {
		t.Error("pendente com vencimento ontem deveria sair como vencida")
	}

	solicitacao.Status = domainsolicitacao.StatusPago
	if solicitacaodto.NovaSolicitacaoResponse(solicitacao).Vencida {
		t.Error("paga não deveria sair como vencida")
	}
}

func TestHistoricoRespondeOEvento(t *testing.T) {
	historico := domainsolicitacao.NovoHistorico(
		uuid.New(), uuid.New(), nil, domainsolicitacao.StatusPendenteAprovacao, domainsolicitacao.EventoCriada, "solicitação criada",
	)

	if resposta := solicitacaodto.NovoHistoricoResponse(historico); resposta.Evento != "criada" {
		t.Errorf("evento = %q, esperado criada", resposta.Evento)
	}
}
