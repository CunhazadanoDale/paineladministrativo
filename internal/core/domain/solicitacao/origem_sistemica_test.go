package solicitacao_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

func dadosDeOrigem() domainsolicitacao.DadosSolicitacaoDeOrigem {
	fornecedor := uuid.New()

	return domainsolicitacao.DadosSolicitacaoDeOrigem{
		EmpresaID:             uuid.New(),
		SolicitanteID:         uuid.New(),
		FornecedorID:          &fornecedor,
		CategoriaFinanceiraID: uuid.New(),
		Origem:                domainsolicitacao.OrigemRecompra,
		OrigemID:              uuid.New(),
		ValorCentavos:         31500,
		Vencimento:            time.Date(2026, 9, 30, 15, 0, 0, 0, time.UTC),
		Observacao:            "recompra do título 1234/1",
	}
}

func TestNovaSolicitacaoDeOrigemNasceClassificadaEPendente(t *testing.T) {
	dados := dadosDeOrigem()
	agora := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)

	solicitacao, historico, err := domainsolicitacao.NovaSolicitacaoDeOrigem(dados, agora)
	if err != nil {
		t.Fatalf("NovaSolicitacaoDeOrigem falhou: %v", err)
	}
	if solicitacao.Status != domainsolicitacao.StatusPendenteAprovacao || solicitacao.Origem != domainsolicitacao.OrigemRecompra || *solicitacao.OrigemID != dados.OrigemID {
		t.Fatalf("solicitação = %+v", solicitacao)
	}
	if *solicitacao.CategoriaFinanceiraID != dados.CategoriaFinanceiraID || solicitacao.FormaPagamento != domainsolicitacao.FormaPagamentoTransferencia ||
		solicitacao.Competencia.String() != "2026-09" || solicitacao.Prazo.String() != "2026-09-30" || solicitacao.Valor.Centavos() != 31500 {
		t.Fatalf("classificação = %+v", solicitacao)
	}
	if historico.Evento != domainsolicitacao.EventoCriada || historico.SolicitacaoID != solicitacao.ID || !strings.Contains(historico.Descricao, "recompra") {
		t.Fatalf("histórico = %+v", historico)
	}

	competencia, _ := domainsolicitacao.NovaCompetencia("2026-10")
	dados.Competencia = &competencia
	comCompetencia, _, err := domainsolicitacao.NovaSolicitacaoDeOrigem(dados, agora)
	if err != nil || comCompetencia.Competencia.String() != "2026-10" {
		t.Fatalf("competência informada = %+v, %v", comCompetencia, err)
	}
}

func TestNovaSolicitacaoDeOrigemValida(t *testing.T) {
	casos := map[string]func(*domainsolicitacao.DadosSolicitacaoDeOrigem){
		"origem manual":  func(d *domainsolicitacao.DadosSolicitacaoDeOrigem) { d.Origem = domainsolicitacao.OrigemManual },
		"sem origem id":  func(d *domainsolicitacao.DadosSolicitacaoDeOrigem) { d.OrigemID = uuid.Nil },
		"sem categoria":  func(d *domainsolicitacao.DadosSolicitacaoDeOrigem) { d.CategoriaFinanceiraID = uuid.Nil },
		"sem vencimento": func(d *domainsolicitacao.DadosSolicitacaoDeOrigem) { d.Vencimento = time.Time{} },
		"valor zero":     func(d *domainsolicitacao.DadosSolicitacaoDeOrigem) { d.ValorCentavos = 0 },
		"sem observação": func(d *domainsolicitacao.DadosSolicitacaoDeOrigem) { d.Observacao = " " },
		"forma inválida": func(d *domainsolicitacao.DadosSolicitacaoDeOrigem) { d.FormaPagamento = "cheque" },
	}

	for nome, alterar := range casos {
		t.Run(nome, func(t *testing.T) {
			dados := dadosDeOrigem()
			alterar(&dados)
			if _, _, err := domainsolicitacao.NovaSolicitacaoDeOrigem(dados, time.Now()); !errors.Is(err, domain.ErrValidacao) {
				t.Fatalf("erro = %v", err)
			}
		})
	}
}
