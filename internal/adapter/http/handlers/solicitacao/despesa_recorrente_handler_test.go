package solicitacao

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	solicitacaodto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/solicitacao"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/solicitacao"
	"github.com/google/uuid"
)

type despesaRecorrenteUseCaseFalso struct {
	erro      error
	despesa   *domainsolicitacao.DespesaRecorrente
	resultado portsin.ResultadoGeracao

	inputCriar    portsin.CriarDespesaRecorrenteInput
	inputAlterar  portsin.AlterarDespesaRecorrenteInput
	inputListar   portsin.ListarDespesasRecorrentesInput
	inputSituacao portsin.MudarSituacaoDespesaRecorrenteInput
	inputGerar    portsin.GerarDespesasRecorrentesInput
}

var _ portsin.DespesaRecorrenteUseCase = (*despesaRecorrenteUseCaseFalso)(nil)

func (f *despesaRecorrenteUseCaseFalso) Criar(_ context.Context, input portsin.CriarDespesaRecorrenteInput) (*domainsolicitacao.DespesaRecorrente, error) {
	f.inputCriar = input
	return f.despesa, f.erro
}

func (f *despesaRecorrenteUseCaseFalso) Obter(_ context.Context, _, _ uuid.UUID) (*domainsolicitacao.DespesaRecorrente, error) {
	return f.despesa, f.erro
}

func (f *despesaRecorrenteUseCaseFalso) Listar(_ context.Context, input portsin.ListarDespesasRecorrentesInput) ([]*domainsolicitacao.DespesaRecorrente, error) {
	f.inputListar = input
	if f.erro != nil {
		return nil, f.erro
	}
	return []*domainsolicitacao.DespesaRecorrente{f.despesa}, nil
}

func (f *despesaRecorrenteUseCaseFalso) Alterar(_ context.Context, input portsin.AlterarDespesaRecorrenteInput) (*domainsolicitacao.DespesaRecorrente, error) {
	f.inputAlterar = input
	return f.despesa, f.erro
}

func (f *despesaRecorrenteUseCaseFalso) Ativar(_ context.Context, input portsin.MudarSituacaoDespesaRecorrenteInput) (*domainsolicitacao.DespesaRecorrente, error) {
	f.inputSituacao = input
	return f.despesa, f.erro
}

func (f *despesaRecorrenteUseCaseFalso) Desativar(_ context.Context, input portsin.MudarSituacaoDespesaRecorrenteInput) (*domainsolicitacao.DespesaRecorrente, error) {
	f.inputSituacao = input
	return f.despesa, f.erro
}

func (f *despesaRecorrenteUseCaseFalso) Remover(_ context.Context, _, _ uuid.UUID) error {
	return f.erro
}

func (f *despesaRecorrenteUseCaseFalso) Gerar(_ context.Context, input portsin.GerarDespesasRecorrentesInput) (portsin.ResultadoGeracao, error) {
	f.inputGerar = input
	return f.resultado, f.erro
}

func despesaRecorrenteDeTeste() *domainsolicitacao.DespesaRecorrente {
	fim := domainsolicitacao.CompetenciaDe(time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC))

	return &domainsolicitacao.DespesaRecorrente{
		ID:                    uuid.New(),
		EmpresaID:             uuid.New(),
		CategoriaFinanceiraID: uuid.New(),
		Descricao:             domainsolicitacao.ObservacaoDe("aluguel"),
		Valor:                 domainsolicitacao.ValorDe(420000),
		FormaPagamento:        domainsolicitacao.FormaPagamentoTransferencia,
		DiaVencimento:         domainsolicitacao.DiaVencimentoDe(10),
		Inicio:                domainsolicitacao.CompetenciaDe(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)),
		Fim:                   &fim,
		Ativa:                 true,
		Versao:                1,
	}
}

func TestCriarDespesaRecorrenteRepassaCorpoERespondeCreated(t *testing.T) {
	usecase := &despesaRecorrenteUseCaseFalso{despesa: despesaRecorrenteDeTeste()}
	empresaID := uuid.New()
	categoriaID := uuid.New()

	resposta := executaProtegido(t, NewDespesaRecorrenteHandler(usecase).Criar, http.MethodPost, "/api/v1/despesas-recorrentes", "", `{
		"empresa_id": "`+empresaID.String()+`",
		"categoria_financeira_id": "`+categoriaID.String()+`",
		"descricao": "aluguel",
		"valor_centavos": 420000,
		"estimado": true,
		"forma_pagamento": "transferencia",
		"dia_vencimento": 31,
		"vencimento_mes_seguinte": true,
		"inicio": "2026-01",
		"fim": "2026-12"
	}`)

	if resposta.Code != http.StatusCreated {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusCreated, resposta.Body.String())
	}
	entrada := usecase.inputCriar
	if entrada.EmpresaID != empresaID || entrada.UsuarioID == uuid.Nil || entrada.Dados.CategoriaFinanceiraID != categoriaID {
		t.Errorf("entrada = %+v", entrada)
	}
	if entrada.Dados.DiaVencimento != 31 || !entrada.Dados.VencimentoMesSeguinte || !entrada.Dados.Estimado ||
		entrada.Dados.Inicio != "2026-01" || entrada.Dados.Fim != "2026-12" || entrada.Dados.ValorCentavos != 420000 {
		t.Errorf("dados = %+v", entrada.Dados)
	}

	var corpo dto.Resposta[solicitacaodto.DespesaRecorrenteResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	if corpo.Dados.Inicio != "2026-01" || corpo.Dados.Fim == nil || *corpo.Dados.Fim != "2026-12" || corpo.Dados.DiaVencimento != 10 {
		t.Errorf("resposta = %+v", corpo.Dados)
	}
}

func TestAlterarEDesativarDespesaRecorrenteRepassamAVersao(t *testing.T) {
	usecase := &despesaRecorrenteUseCaseFalso{despesa: despesaRecorrenteDeTeste()}
	id := uuid.New()
	handler := NewDespesaRecorrenteHandler(usecase)

	resposta := executaProtegido(t, handler.Alterar, http.MethodPut, "/api/v1/despesas-recorrentes/"+id.String(), id.String(),
		`{"descricao": "aluguel novo", "versao": 3}`)
	if resposta.Code != http.StatusOK || usecase.inputAlterar.DespesaID != id || usecase.inputAlterar.Versao != 3 ||
		usecase.inputAlterar.Dados.Descricao != "aluguel novo" {
		t.Errorf("alterar = %d, %+v", resposta.Code, usecase.inputAlterar)
	}

	resposta = executaProtegido(t, handler.Desativar, http.MethodPatch, "/api/v1/despesas-recorrentes/"+id.String()+"/desativar", id.String(),
		`{"versao": 4}`)
	if resposta.Code != http.StatusOK || usecase.inputSituacao.DespesaID != id || usecase.inputSituacao.Versao != 4 {
		t.Errorf("desativar = %d, %+v", resposta.Code, usecase.inputSituacao)
	}
}

func TestListarDespesasRecorrentesRepassaFiltrosEValidaConsulta(t *testing.T) {
	usecase := &despesaRecorrenteUseCaseFalso{despesa: despesaRecorrenteDeTeste()}
	empresaID := uuid.New()

	resposta := executaProtegido(t, NewDespesaRecorrenteHandler(usecase).Listar, http.MethodGet,
		"/api/v1/despesas-recorrentes?empresa="+empresaID.String()+"&ativa=false&pagina=2&tamanho=5", "", "")
	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d: %s", resposta.Code, resposta.Body.String())
	}
	entrada := usecase.inputListar
	if entrada.Empresa == nil || *entrada.Empresa != empresaID || entrada.Ativa == nil || *entrada.Ativa || entrada.Filtro.Page != 2 {
		t.Errorf("entrada = %+v", entrada)
	}

	for _, consulta := range []string{"empresa=x", "fornecedor=x", "ativa=talvez"} {
		resposta := executaProtegido(t, NewDespesaRecorrenteHandler(&despesaRecorrenteUseCaseFalso{}).Listar, http.MethodGet,
			"/api/v1/despesas-recorrentes?"+consulta, "", "")
		apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
	}
}

func TestGerarDespesasRecorrentesRepassaCompetenciaEDevolveOResumo(t *testing.T) {
	gerada := uuid.New()
	recusada := uuid.New()
	usecase := &despesaRecorrenteUseCaseFalso{resultado: portsin.ResultadoGeracao{
		Competencia:  "2026-11",
		Geradas:      []uuid.UUID{gerada},
		JaExistentes: 2,
		Recusadas:    []portsin.RecorrenciaRecusada{{DespesaRecorrenteID: recusada, Motivo: "fornecedor inativo não recebe conta a pagar"}},
	}}
	empresaID := uuid.New()

	resposta := executaProtegido(t, NewDespesaRecorrenteHandler(usecase).Gerar, http.MethodPost,
		"/api/v1/despesas-recorrentes/gerar?competencia=2026-11&empresa="+empresaID.String(), "", "")

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d: %s", resposta.Code, resposta.Body.String())
	}
	if usecase.inputGerar.Competencia != "2026-11" || usecase.inputGerar.Empresa == nil || *usecase.inputGerar.Empresa != empresaID {
		t.Errorf("entrada = %+v", usecase.inputGerar)
	}

	var corpo dto.Resposta[solicitacaodto.ResultadoGeracaoResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	if corpo.Dados.Competencia != "2026-11" || len(corpo.Dados.Geradas) != 1 || corpo.Dados.Geradas[0] != gerada || corpo.Dados.JaExistentes != 2 {
		t.Errorf("resumo = %+v", corpo.Dados)
	}
	if len(corpo.Dados.Recusadas) != 1 || corpo.Dados.Recusadas[0].DespesaRecorrenteID != recusada {
		t.Errorf("recusadas = %+v", corpo.Dados.Recusadas)
	}
}

func TestGerarSemNadaDevolveListasVazias(t *testing.T) {
	usecase := &despesaRecorrenteUseCaseFalso{resultado: portsin.ResultadoGeracao{Competencia: "2026-11"}}

	resposta := executaProtegido(t, NewDespesaRecorrenteHandler(usecase).Gerar, http.MethodPost,
		"/api/v1/despesas-recorrentes/gerar?competencia=2026-11", "", "")

	if resposta.Body.String() == "" || resposta.Code != http.StatusOK {
		t.Fatalf("status %d", resposta.Code)
	}
	var corpo map[string]map[string]any
	if err := json.Unmarshal(resposta.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	if geradas, ok := corpo["dados"]["geradas"].([]any); !ok || len(geradas) != 0 {
		t.Errorf("geradas = %v, esperado lista vazia", corpo["dados"]["geradas"])
	}
	if recusadas, ok := corpo["dados"]["recusadas"].([]any); !ok || len(recusadas) != 0 {
		t.Errorf("recusadas = %v, esperado lista vazia", corpo["dados"]["recusadas"])
	}
}

func TestRemoverDespesaRecorrenteComSolicitacoesResponde400(t *testing.T) {
	usecase := &despesaRecorrenteUseCaseFalso{erro: domainsolicitacao.ErroDespesaRecorrenteComSolicitacoes()}
	id := uuid.NewString()

	resposta := executaProtegido(t, NewDespesaRecorrenteHandler(usecase).Remover, http.MethodDelete,
		"/api/v1/despesas-recorrentes/"+id, id, "")

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
	if apoioteste.MensagemDoEnvelope(t, resposta) != domain.ErroValidacao("despesa recorrente já gerou solicitações e não pode ser excluída; desative-a").Error() {
		t.Errorf("mensagem = %q", apoioteste.MensagemDoEnvelope(t, resposta))
	}
}
