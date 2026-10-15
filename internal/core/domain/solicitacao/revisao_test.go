package solicitacao_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

func solicitacaoGeradaDeTeste(t *testing.T) *domainsolicitacao.Solicitacao {
	t.Helper()

	despesa, err := domainsolicitacao.NovaDespesaRecorrente(uuid.New(), dadosDeRecorrencia(t, 10, true, "2026-01", nil))
	if err != nil {
		t.Fatalf("recorrência de teste inválida: %v", err)
	}
	solicitacao, err := despesa.GerarSolicitacao(uuid.New(), *competenciaDeTeste(t, "2026-10"), time.Now().UTC())
	if err != nil {
		t.Fatalf("geração de teste falhou: %v", err)
	}

	return solicitacao
}

func TestEditarSolicitacaoGeradaMantemACompetencia(t *testing.T) {
	solicitacao := solicitacaoGeradaDeTeste(t)
	dados := dadosDaSolicitacao(solicitacao)
	dados.Classificacao.Competencia = nil
	dados.Prazo = domainsolicitacao.PrazoPagamentoDe(solicitacao.Prazo.Data().AddDate(0, 1, 0))

	if _, err := solicitacao.Editar(dados, time.Now().UTC()); err != nil {
		t.Fatalf("edição falhou: %v", err)
	}
	if solicitacao.Competencia.String() != "2026-10" {
		t.Errorf("competência = %s, esperado 2026-10 mantida", solicitacao.Competencia)
	}

	dados.Classificacao.Competencia = competenciaDeTeste(t, "2026-11")
	if _, err := solicitacao.Editar(dados, time.Now().UTC()); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("trocar a competência da gerada = %v, esperado erro de validação", err)
	}
}

func TestClassificarSolicitacaoGeradaNaoTrocaACompetencia(t *testing.T) {
	solicitacao := solicitacaoGeradaDeTeste(t)
	if err := solicitacao.Aprovar(uuid.New(), time.Now().UTC()); err != nil {
		t.Fatalf("aprovação falhou: %v", err)
	}
	categoriaID := uuid.New()

	outra := domainsolicitacao.Classificacao{CategoriaFinanceiraID: &categoriaID, Competencia: competenciaDeTeste(t, "2026-11")}
	if err := solicitacao.Classificar(outra, time.Now().UTC()); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("trocar a competência = %v, esperado erro de validação", err)
	}

	mesma := domainsolicitacao.Classificacao{CategoriaFinanceiraID: &categoriaID, Competencia: competenciaDeTeste(t, "2026-10")}
	if err := solicitacao.Classificar(mesma, time.Now().UTC()); err != nil {
		t.Errorf("reclassificar na mesma competência = %v", err)
	}
}

func TestExigirPagavelDaPrioridadeAoStatus(t *testing.T) {
	pendente := novaSolicitacaoDeTeste(t)
	if err := pendente.ExigirPagavel(); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("pendente = %v, esperado conflito", err)
	}
	if pendente.Status != domainsolicitacao.StatusPendenteAprovacao {
		t.Errorf("conferir mudou o status para %q", pendente.Status)
	}

	semCategoria := novaSolicitacaoDeTeste(t)
	if err := semCategoria.Aprovar(uuid.New(), time.Now().UTC()); err != nil {
		t.Fatalf("aprovação falhou: %v", err)
	}
	if err := semCategoria.ExigirPagavel(); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("aprovada sem categoria = %v, esperado erro de validação", err)
	}
}

func TestMesmoVencimentoComparaODia(t *testing.T) {
	solicitacao := novaSolicitacaoDeTeste(t)
	mesmoDiaOutraHora := domainsolicitacao.PrazoPagamentoDe(solicitacao.Prazo.Data().Truncate(24 * time.Hour).Add(20 * time.Hour))

	if !solicitacao.MesmoVencimento(mesmoDiaOutraHora) {
		t.Error("mesmo dia deveria ser o mesmo vencimento")
	}
	if solicitacao.MesmoVencimento(domainsolicitacao.PrazoPagamentoDe(solicitacao.Prazo.Data().AddDate(0, 0, 1))) {
		t.Error("dia seguinte não é o mesmo vencimento")
	}
}
