package solicitacao_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

func dadosDeRecorrencia(t *testing.T, dia int, mesSeguinte bool, inicio string, fim *string) domainsolicitacao.DadosDespesaRecorrente {
	t.Helper()

	descricao, _ := domainsolicitacao.NovaObservacao("aluguel do galpão")
	valor, _ := domainsolicitacao.NovoValor(420000)
	diaVencimento, err := domainsolicitacao.NovoDiaVencimento(dia)
	if err != nil {
		t.Fatalf("dia de teste inválido: %v", err)
	}

	dados := domainsolicitacao.DadosDespesaRecorrente{
		CategoriaFinanceiraID: uuid.New(),
		Descricao:             descricao,
		Valor:                 valor,
		FormaPagamento:        domainsolicitacao.FormaPagamentoTransferencia,
		DiaVencimento:         diaVencimento,
		VencimentoMesSeguinte: mesSeguinte,
		Inicio:                *competenciaDeTeste(t, inicio),
	}
	if fim != nil {
		dados.Fim = competenciaDeTeste(t, *fim)
	}

	return dados
}

func textoDe(valor string) *string {
	return &valor
}

func TestNovaDespesaRecorrenteNasceAtivaNaVersaoUm(t *testing.T) {
	empresaID := uuid.New()

	despesa, err := domainsolicitacao.NovaDespesaRecorrente(empresaID, dadosDeRecorrencia(t, 10, false, "2026-01", nil))
	if err != nil {
		t.Fatalf("criação falhou: %v", err)
	}
	if despesa.ID == uuid.Nil || despesa.EmpresaID != empresaID || !despesa.Ativa || despesa.Versao != 1 {
		t.Errorf("despesa = %+v", despesa)
	}
}

func TestNovaDespesaRecorrenteRecusaDadosInvalidos(t *testing.T) {
	casos := map[string]struct {
		empresaID uuid.UUID
		ajustar   func(*domainsolicitacao.DadosDespesaRecorrente)
	}{
		"sem empresa":         {uuid.Nil, func(*domainsolicitacao.DadosDespesaRecorrente) {}},
		"sem categoria":       {uuid.New(), func(d *domainsolicitacao.DadosDespesaRecorrente) { d.CategoriaFinanceiraID = uuid.Nil }},
		"forma inválida":      {uuid.New(), func(d *domainsolicitacao.DadosDespesaRecorrente) { d.FormaPagamento = "dinheiro" }},
		"fim antes do início": {uuid.New(), func(d *domainsolicitacao.DadosDespesaRecorrente) { d.Fim = competenciaDeTeste(t, "2025-12") }},
	}

	for nome, caso := range casos {
		dados := dadosDeRecorrencia(t, 10, false, "2026-01", nil)
		caso.ajustar(&dados)

		if _, err := domainsolicitacao.NovaDespesaRecorrente(caso.empresaID, dados); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("%s = %v, esperado erro de validação", nome, err)
		}
	}
}

func TestNovoDiaVencimentoEntreUmETrintaEUm(t *testing.T) {
	for _, dia := range []int{0, 32, -1} {
		if _, err := domainsolicitacao.NovoDiaVencimento(dia); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("dia %d = %v, esperado erro de validação", dia, err)
		}
	}
	for _, dia := range []int{1, 15, 31} {
		if _, err := domainsolicitacao.NovoDiaVencimento(dia); err != nil {
			t.Errorf("dia %d = %v, esperado válido", dia, err)
		}
	}
}

func TestVencimentoNaCompetenciaAjustaMesCurtoEMesSeguinte(t *testing.T) {
	casos := []struct {
		dia         int
		mesSeguinte bool
		competencia string
		esperado    string
	}{
		{10, false, "2026-03", "2026-03-10"},
		{31, false, "2026-02", "2026-02-28"},
		{31, false, "2028-02", "2028-02-29"},
		{30, true, "2026-01", "2026-02-28"},
		{5, true, "2026-12", "2027-01-05"},
	}

	for _, caso := range casos {
		despesa, err := domainsolicitacao.NovaDespesaRecorrente(uuid.New(), dadosDeRecorrencia(t, caso.dia, caso.mesSeguinte, "2026-01", nil))
		if err != nil {
			t.Fatalf("criação falhou: %v", err)
		}

		if obtido := despesa.VencimentoNa(*competenciaDeTeste(t, caso.competencia)).String(); obtido != caso.esperado {
			t.Errorf("dia %d, mês seguinte %v, competência %s = %s, esperado %s", caso.dia, caso.mesSeguinte, caso.competencia, obtido, caso.esperado)
		}
	}
}

func TestVigenciaRespeitaInicioFimEAtiva(t *testing.T) {
	despesa, err := domainsolicitacao.NovaDespesaRecorrente(uuid.New(), dadosDeRecorrencia(t, 10, false, "2026-03", textoDe("2026-06")))
	if err != nil {
		t.Fatalf("criação falhou: %v", err)
	}

	esperados := map[string]bool{"2026-02": false, "2026-03": true, "2026-06": true, "2026-07": false}
	for competencia, vigente := range esperados {
		if obtido := despesa.VigenteEm(*competenciaDeTeste(t, competencia)); obtido != vigente {
			t.Errorf("vigente em %s = %v, esperado %v", competencia, obtido, vigente)
		}
	}

	despesa.Desativar(time.Now().UTC())
	if despesa.VigenteEm(*competenciaDeTeste(t, "2026-04")) {
		t.Error("inativa não deveria estar vigente")
	}
}

func TestGerarSolicitacaoDaRecorrencia(t *testing.T) {
	fornecedorID := uuid.New()
	dados := dadosDeRecorrencia(t, 31, false, "2026-01", nil)
	dados.FornecedorID = &fornecedorID
	dados.Estimado = true
	despesa, err := domainsolicitacao.NovaDespesaRecorrente(uuid.New(), dados)
	if err != nil {
		t.Fatalf("criação falhou: %v", err)
	}
	solicitanteID := uuid.New()

	solicitacao, err := despesa.GerarSolicitacao(solicitanteID, *competenciaDeTeste(t, "2026-04"), time.Now().UTC())
	if err != nil {
		t.Fatalf("geração falhou: %v", err)
	}

	if solicitacao.Origem != domainsolicitacao.OrigemRecorrencia || solicitacao.OrigemID == nil || *solicitacao.OrigemID != despesa.ID {
		t.Errorf("origem = %q/%v", solicitacao.Origem, solicitacao.OrigemID)
	}
	if solicitacao.Status != domainsolicitacao.StatusPendenteAprovacao || solicitacao.Versao != 1 || solicitacao.SolicitanteID != solicitanteID {
		t.Errorf("status = %q, versão = %d, solicitante = %s", solicitacao.Status, solicitacao.Versao, solicitacao.SolicitanteID)
	}
	if solicitacao.Prazo.String() != "2026-04-30" || solicitacao.Competencia.String() != "2026-04" {
		t.Errorf("vencimento = %s, competência = %s", solicitacao.Prazo, solicitacao.Competencia)
	}
	if solicitacao.EmpresaID != despesa.EmpresaID || solicitacao.Valor != despesa.Valor || solicitacao.Observacao != despesa.Descricao ||
		!solicitacao.Estimado || *solicitacao.FornecedorID != fornecedorID || *solicitacao.CategoriaFinanceiraID != despesa.CategoriaFinanceiraID {
		t.Errorf("solicitação = %+v", solicitacao)
	}

	if _, err := despesa.GerarSolicitacao(solicitanteID, *competenciaDeTeste(t, "2025-12"), time.Now().UTC()); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("fora da vigência = %v, esperado erro de validação", err)
	}
}

func TestGerarSolicitacaoAceitaVencimentoNoPassado(t *testing.T) {
	despesa, err := domainsolicitacao.NovaDespesaRecorrente(uuid.New(), dadosDeRecorrencia(t, 5, false, "2020-01", nil))
	if err != nil {
		t.Fatalf("criação falhou: %v", err)
	}

	solicitacao, err := despesa.GerarSolicitacao(uuid.New(), *competenciaDeTeste(t, "2020-03"), time.Now().UTC())
	if err != nil {
		t.Fatalf("competência atrasada = %v, esperado gerada", err)
	}
	if !solicitacao.Vencida(time.Now().UTC()) {
		t.Error("solicitação de competência atrasada deveria nascer vencida")
	}
}

func TestDespesaRecorrenteConferirVersaoEAtivacao(t *testing.T) {
	despesa, err := domainsolicitacao.NovaDespesaRecorrente(uuid.New(), dadosDeRecorrencia(t, 10, false, "2026-01", nil))
	if err != nil {
		t.Fatalf("criação falhou: %v", err)
	}

	if err := despesa.ConferirVersao(0); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("versão zero = %v", err)
	}
	if err := despesa.ConferirVersao(2); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("versão divergente = %v", err)
	}
	if despesa.Ativar(time.Now().UTC()) {
		t.Error("ativar ativa não deveria mudar nada")
	}
	if !despesa.Desativar(time.Now().UTC()) || despesa.Ativa {
		t.Error("desativar deveria mudar a situação")
	}
}
