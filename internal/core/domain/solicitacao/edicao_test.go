package solicitacao_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

func dadosDaSolicitacao(s *domainsolicitacao.Solicitacao) domainsolicitacao.DadosEdicao {
	competencia := s.Competencia

	return domainsolicitacao.DadosEdicao{
		Valor:          s.Valor,
		Prazo:          s.Prazo,
		Observacao:     s.Observacao,
		FormaPagamento: s.FormaPagamento,
		Classificacao: domainsolicitacao.Classificacao{
			FornecedorID:          s.FornecedorID,
			CategoriaFinanceiraID: s.CategoriaFinanceiraID,
			Competencia:           &competencia,
		},
		Estimado: s.Estimado,
	}
}

func prazoDeTeste(t *testing.T, dias int) domainsolicitacao.PrazoPagamento {
	t.Helper()

	prazo, err := domainsolicitacao.NovoPrazoPagamento(time.Now().UTC().AddDate(0, 0, dias))
	if err != nil {
		t.Fatalf("prazo de teste inválido: %v", err)
	}

	return prazo
}

func TestEditarAplicaOsCamposEListaOsAlterados(t *testing.T) {
	solicitacao := novaSolicitacaoDeTeste(t)
	fornecedorID := uuid.New()
	valor, _ := domainsolicitacao.NovoValor(99000)
	observacao, _ := domainsolicitacao.NovaObservacao("compra de areia")

	dados := dadosDaSolicitacao(solicitacao)
	dados.Valor = valor
	dados.Observacao = observacao
	dados.FormaPagamento = domainsolicitacao.FormaPagamentoBoleto
	dados.Classificacao.FornecedorID = &fornecedorID
	dados.Estimado = true

	alterados, err := solicitacao.Editar(dados, time.Now().UTC())
	if err != nil {
		t.Fatalf("edição falhou: %v", err)
	}

	esperados := []string{"valor", "observação", "forma de pagamento", "fornecedor", "estimado"}
	if !reflect.DeepEqual(alterados, esperados) {
		t.Errorf("alterados = %v, esperado %v", alterados, esperados)
	}
	if solicitacao.Valor.Centavos() != 99000 || solicitacao.Observacao.Texto() != "compra de areia" || !solicitacao.Estimado {
		t.Error("campos editados não foram aplicados")
	}
	if solicitacao.FornecedorID == nil || *solicitacao.FornecedorID != fornecedorID {
		t.Error("fornecedor não foi aplicado")
	}
	if solicitacao.Versao != 2 {
		t.Errorf("versão = %d, esperado 2", solicitacao.Versao)
	}
}

func TestEditarSemMudancaNaoAvancaAVersao(t *testing.T) {
	solicitacao := novaSolicitacaoDeTeste(t)

	alterados, err := solicitacao.Editar(dadosDaSolicitacao(solicitacao), time.Now().UTC())
	if err != nil {
		t.Fatalf("edição falhou: %v", err)
	}
	if len(alterados) != 0 || solicitacao.Versao != 1 {
		t.Errorf("alterados = %v, versão = %d; esperado nenhum e versão 1", alterados, solicitacao.Versao)
	}
}

func TestEditarVencimentoSemCompetenciaRecalculaACompetencia(t *testing.T) {
	solicitacao := novaSolicitacaoDeTeste(t)
	novoPrazo := domainsolicitacao.PrazoPagamentoDe(solicitacao.Prazo.Data().AddDate(0, 2, 0))

	dados := dadosDaSolicitacao(solicitacao)
	dados.Prazo = novoPrazo
	dados.Classificacao.Competencia = nil

	alterados, err := solicitacao.Editar(dados, time.Now().UTC())
	if err != nil {
		t.Fatalf("edição falhou: %v", err)
	}
	if !reflect.DeepEqual(alterados, []string{"vencimento", "competência"}) {
		t.Errorf("alterados = %v", alterados)
	}
	if solicitacao.Competencia != domainsolicitacao.CompetenciaDe(novoPrazo.Data()) {
		t.Errorf("competência = %s, esperado o mês do novo vencimento", solicitacao.Competencia)
	}
}

func TestEditarSoEmPendenteDeAprovacao(t *testing.T) {
	solicitacao := novaSolicitacaoDeTeste(t)
	if err := solicitacao.Aprovar(uuid.New(), time.Now().UTC()); err != nil {
		t.Fatalf("aprovação falhou: %v", err)
	}

	if _, err := solicitacao.Editar(dadosDaSolicitacao(solicitacao), time.Now().UTC()); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("editar aprovada = %v, esperado conflito", err)
	}
}

func TestEditarRecusaFormaInvalida(t *testing.T) {
	solicitacao := novaSolicitacaoDeTeste(t)
	dados := dadosDaSolicitacao(solicitacao)
	dados.FormaPagamento = domainsolicitacao.FormaPagamento("dinheiro")

	if _, err := solicitacao.Editar(dados, time.Now().UTC()); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("forma inválida = %v, esperado erro de validação", err)
	}
}

func TestEditarPeloSolicitanteOuAdministrador(t *testing.T) {
	solicitacao := novaSolicitacaoDeTeste(t)

	if !solicitacao.PodeSerEditadaPor(solicitacao.SolicitanteID, false) {
		t.Error("solicitante deveria editar")
	}
	if !solicitacao.PodeSerEditadaPor(uuid.New(), true) {
		t.Error("administrador deveria editar")
	}
	if solicitacao.PodeSerEditadaPor(uuid.New(), false) {
		t.Error("terceiro não deveria editar")
	}
}

func TestAdiarSoEmAprovadaParaDepoisDoVencimentoAtual(t *testing.T) {
	solicitacao := novaSolicitacaoDeTeste(t)
	vencimentoOriginal := solicitacao.Prazo

	if _, err := solicitacao.Adiar(prazoDeTeste(t, 30), time.Now().UTC()); !errors.Is(err, domain.ErrConflito) {
		t.Fatalf("adiar pendente = %v, esperado conflito", err)
	}

	if err := solicitacao.Aprovar(uuid.New(), time.Now().UTC()); err != nil {
		t.Fatalf("aprovação falhou: %v", err)
	}
	if _, err := solicitacao.Adiar(vencimentoOriginal, time.Now().UTC()); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("mesmo dia = %v, esperado erro de validação", err)
	}
	if _, err := solicitacao.Adiar(prazoDeTeste(t, 1), time.Now().UTC()); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("antes do atual = %v, esperado erro de validação", err)
	}

	versaoAprovada := solicitacao.Versao
	competencia := solicitacao.Competencia
	novo := prazoDeTeste(t, 30)

	anterior, err := solicitacao.Adiar(novo, time.Now().UTC())
	if err != nil {
		t.Fatalf("adiamento falhou: %v", err)
	}
	if anterior != vencimentoOriginal || solicitacao.Prazo != novo {
		t.Errorf("anterior = %s, atual = %s", anterior, solicitacao.Prazo)
	}
	if solicitacao.Versao != versaoAprovada+1 || solicitacao.Competencia != competencia {
		t.Errorf("versão = %d, competência = %s; esperado versão avançada e competência mantida", solicitacao.Versao, solicitacao.Competencia)
	}
}

func TestNovoMotivoAdiamento(t *testing.T) {
	motivo, err := domainsolicitacao.NovoMotivoAdiamento("  fornecedor pediu mais prazo  ")
	if err != nil || motivo.Texto() != "fornecedor pediu mais prazo" {
		t.Errorf("motivo = %q, %v", motivo.Texto(), err)
	}

	for _, texto := range []string{"", "   ", strings.Repeat("a", 501)} {
		if _, err := domainsolicitacao.NovoMotivoAdiamento(texto); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("motivo de %d caracteres = %v, esperado erro de validação", len(texto), err)
		}
	}
}

func TestPrazoPagamentoFormataComoData(t *testing.T) {
	prazo := domainsolicitacao.PrazoPagamentoDe(time.Date(2026, time.October, 20, 0, 0, 0, 0, time.UTC))

	if prazo.String() != "2026-10-20" {
		t.Errorf("prazo = %q, esperado 2026-10-20", prazo.String())
	}
}
