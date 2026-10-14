package solicitacao_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

func competenciaDeTeste(t *testing.T, texto string) *domainsolicitacao.Competencia {
	t.Helper()

	competencia, err := domainsolicitacao.NovaCompetencia(texto)
	if err != nil {
		t.Fatalf("competência de teste inválida: %v", err)
	}

	return &competencia
}

func TestNovaSolicitacaoNasceManualNaVersaoUmComCompetenciaDoVencimento(t *testing.T) {
	solicitacao := novaSolicitacaoDeTeste(t)

	if solicitacao.Origem != domainsolicitacao.OrigemManual || solicitacao.OrigemID != nil {
		t.Errorf("origem = %q/%v, esperado manual sem id", solicitacao.Origem, solicitacao.OrigemID)
	}
	if solicitacao.Versao != 1 {
		t.Errorf("versão = %d, esperado 1", solicitacao.Versao)
	}
	if solicitacao.Competencia != domainsolicitacao.CompetenciaDe(solicitacao.Prazo.Data()) {
		t.Errorf("competência = %s, esperado o mês do vencimento %s", solicitacao.Competencia, solicitacao.Prazo.Data())
	}
	if solicitacao.FornecedorID != nil || solicitacao.CategoriaFinanceiraID != nil || solicitacao.Estimado {
		t.Error("solicitação sem classificação nasceu classificada")
	}
}

func TestNovaSolicitacaoGuardaClassificacaoInformada(t *testing.T) {
	base := novaSolicitacaoDeTeste(t)
	fornecedorID := uuid.New()
	categoriaID := uuid.New()

	solicitacao, err := domainsolicitacao.NovaSolicitacao(
		base.SolicitanteID, base.EmpresaID, base.Valor, base.Prazo, base.Observacao, base.FormaPagamento,
		domainsolicitacao.Classificacao{
			FornecedorID:          &fornecedorID,
			CategoriaFinanceiraID: &categoriaID,
			Competencia:           competenciaDeTeste(t, "2026-09"),
		},
		true,
		nil,
	)
	if err != nil {
		t.Fatalf("criação falhou: %v", err)
	}

	if *solicitacao.FornecedorID != fornecedorID || *solicitacao.CategoriaFinanceiraID != categoriaID {
		t.Error("classificação informada não foi guardada")
	}
	if solicitacao.Competencia.String() != "2026-09" || !solicitacao.Estimado {
		t.Errorf("competência = %s, estimado = %v", solicitacao.Competencia, solicitacao.Estimado)
	}
}

func TestClassificarSoEmAprovadaEAvancaAVersao(t *testing.T) {
	solicitacao := novaSolicitacaoDeTeste(t)
	categoriaID := uuid.New()
	classificacao := domainsolicitacao.Classificacao{
		CategoriaFinanceiraID: &categoriaID,
		Competencia:           competenciaDeTeste(t, "2026-08"),
	}

	if err := solicitacao.Classificar(classificacao, time.Now().UTC()); !errors.Is(err, domain.ErrConflito) {
		t.Fatalf("classificar pendente = %v, esperado erro de conflito", err)
	}

	if err := solicitacao.Aprovar(uuid.New(), time.Now().UTC()); err != nil {
		t.Fatalf("aprovação falhou: %v", err)
	}
	versaoAprovada := solicitacao.Versao

	if err := solicitacao.Classificar(classificacao, time.Now().UTC()); err != nil {
		t.Fatalf("classificação falhou: %v", err)
	}
	if solicitacao.Versao != versaoAprovada+1 {
		t.Errorf("versão = %d, esperado %d", solicitacao.Versao, versaoAprovada+1)
	}
	if *solicitacao.CategoriaFinanceiraID != categoriaID || solicitacao.Competencia.String() != "2026-08" {
		t.Error("classificação não aplicada")
	}
	if solicitacao.FornecedorID != nil {
		t.Error("fornecedor omitido deveria ficar vazio")
	}
}

func TestClassificarExigeCategoriaECompetencia(t *testing.T) {
	solicitacao := novaSolicitacaoDeTeste(t)
	if err := solicitacao.Aprovar(uuid.New(), time.Now().UTC()); err != nil {
		t.Fatalf("aprovação falhou: %v", err)
	}
	categoriaID := uuid.New()

	casos := map[string]domainsolicitacao.Classificacao{
		"sem categoria":   {Competencia: competenciaDeTeste(t, "2026-08")},
		"sem competência": {CategoriaFinanceiraID: &categoriaID},
	}
	for nome, classificacao := range casos {
		if err := solicitacao.Classificar(classificacao, time.Now().UTC()); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("%s = %v, esperado erro de validação", nome, err)
		}
	}
}

func TestConferirVersao(t *testing.T) {
	solicitacao := novaSolicitacaoDeTeste(t)

	if err := solicitacao.ConferirVersao(1); err != nil {
		t.Errorf("versão atual = %v, esperado sem erro", err)
	}
	if err := solicitacao.ConferirVersao(0); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("versão zero = %v, esperado erro de validação", err)
	}
	if err := solicitacao.ConferirVersao(2); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("versão divergente = %v, esperado erro de conflito", err)
	}
}

func TestTransicoesAvancamAVersao(t *testing.T) {
	solicitacao := novaSolicitacaoDeTeste(t)

	if err := solicitacao.Aprovar(uuid.New(), time.Now().UTC()); err != nil {
		t.Fatalf("aprovação falhou: %v", err)
	}
	if err := solicitacao.MarcarComoPago(time.Now().UTC()); err != nil {
		t.Fatalf("pagamento falhou: %v", err)
	}
	if solicitacao.Versao != 3 {
		t.Errorf("versão = %d, esperado 3", solicitacao.Versao)
	}
}

func TestVencidaSoEmAbertoComVencimentoAntesDeHoje(t *testing.T) {
	solicitacao := novaSolicitacaoDeTeste(t)
	vencimento := solicitacao.Prazo.Data()

	if solicitacao.Vencida(vencimento) {
		t.Error("no dia do vencimento ainda não está vencida")
	}
	if !solicitacao.Vencida(vencimento.AddDate(0, 0, 1)) {
		t.Error("um dia depois do vencimento deveria estar vencida")
	}

	if err := solicitacao.Aprovar(uuid.New(), time.Now().UTC()); err != nil {
		t.Fatalf("aprovação falhou: %v", err)
	}
	if !solicitacao.Vencida(vencimento.AddDate(0, 0, 1)) {
		t.Error("aprovada depois do vencimento deveria estar vencida")
	}

	if err := solicitacao.MarcarComoPago(time.Now().UTC()); err != nil {
		t.Fatalf("pagamento falhou: %v", err)
	}
	if solicitacao.Vencida(vencimento.AddDate(0, 0, 1)) {
		t.Error("paga nunca está vencida")
	}
}
