package receber_test

import (
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	"github.com/google/uuid"
)

func parcelasDeTeste() []domainreceber.ParcelaDeOrigem {
	return []domainreceber.ParcelaDeOrigem{
		{OrigemID: uuid.New(), Numero: " 1234 ", Parcela: 1, Vencimento: data(2026, 9, 10), ValorCentavos: 50000},
		{OrigemID: uuid.New(), Numero: "1234", Parcela: 2, Vencimento: data(2026, 11, 10), ValorCentavos: 50000},
	}
}

func TestNovaGeracaoMontaOsTitulosEOHistorico(t *testing.T) {
	empresa, cliente, usuario, conta := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	parcelas := parcelasDeTeste()

	geracao, err := domainreceber.NovaGeracao(domainreceber.OrigemNotaFiscalDuplicata, empresa, cliente, &conta, parcelas, usuario, agora)
	if err != nil {
		t.Fatalf("NovaGeracao falhou: %v", err)
	}
	if len(geracao.Titulos) != 2 || len(geracao.Historicos) != 2 {
		t.Fatalf("geração = %+v", geracao)
	}

	primeiro := geracao.Titulos[0]
	if primeiro.Origem != domainreceber.OrigemNotaFiscalDuplicata || *primeiro.OrigemID != parcelas[0].OrigemID || primeiro.Numero != "1234" {
		t.Fatalf("primeiro título = %+v", primeiro)
	}
	if primeiro.Status != domainreceber.StatusEmBanco || primeiro.CategoriaFinanceiraID != nil || primeiro.Competencia.String() != "2026-09" {
		t.Fatalf("título gerado = %+v", primeiro)
	}
	if geracao.Historicos[0].Evento != domainreceber.EventoGerado || geracao.Historicos[0].TituloID != primeiro.ID {
		t.Fatalf("histórico = %+v", geracao.Historicos[0])
	}
	if ids := geracao.OrigemIDs(); len(ids) != 2 || ids[1] != parcelas[1].OrigemID {
		t.Fatalf("ids de origem = %v", ids)
	}
}

func TestNovaGeracaoAceitaVencimentoNoPassadoESemConta(t *testing.T) {
	parcelas := parcelasDeTeste()[:1]

	geracao, err := domainreceber.NovaGeracao(domainreceber.OrigemPedido, uuid.New(), uuid.New(), nil, parcelas, uuid.New(), agora)
	if err != nil {
		t.Fatalf("NovaGeracao falhou: %v", err)
	}
	if geracao.Titulos[0].Status != domainreceber.StatusCarteira || !geracao.Titulos[0].Vencido(domainreceber.Hoje(agora)) {
		t.Fatalf("título = %+v", geracao.Titulos[0])
	}
}

func TestNovaGeracaoValida(t *testing.T) {
	casos := map[string]struct {
		origem   domainreceber.Origem
		parcelas func() []domainreceber.ParcelaDeOrigem
		trecho   string
	}{
		"manual":        {domainreceber.OrigemManual, parcelasDeTeste, "diferente de manual"},
		"sem parcelas":  {domainreceber.OrigemPedido, func() []domainreceber.ParcelaDeOrigem { return nil }, "pelo menos uma parcela"},
		"sem origem id": {domainreceber.OrigemPedido, func() []domainreceber.ParcelaDeOrigem { p := parcelasDeTeste(); p[1].OrigemID = uuid.Nil; return p }, "parcela 2 sem o id"},
		"origem repetida": {domainreceber.OrigemPedido, func() []domainreceber.ParcelaDeOrigem {
			p := parcelasDeTeste()
			p[1].OrigemID = p[0].OrigemID
			return p
		}, "parcela 2 repete"},
		"sem número":   {domainreceber.OrigemPedido, func() []domainreceber.ParcelaDeOrigem { p := parcelasDeTeste(); p[0].Numero = ""; return p }, "número do título é obrigatório"},
		"parcela zero": {domainreceber.OrigemPedido, func() []domainreceber.ParcelaDeOrigem { p := parcelasDeTeste(); p[0].Parcela = 0; return p }, "parcela deve estar"},
		"sem vencimento": {domainreceber.OrigemPedido, func() []domainreceber.ParcelaDeOrigem {
			p := parcelasDeTeste()
			p[0].Vencimento = time.Time{}
			return p
		}, "vencimento é obrigatório"},
		"valor zero": {domainreceber.OrigemPedido, func() []domainreceber.ParcelaDeOrigem { p := parcelasDeTeste(); p[0].ValorCentavos = 0; return p }, "maior que zero"},
	}

	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			_, err := domainreceber.NovaGeracao(caso.origem, uuid.New(), uuid.New(), nil, caso.parcelas(), uuid.New(), agora)
			esperarErro(t, err, domain.ErrValidacao, caso.trecho)
		})
	}

	_, err := domainreceber.NovaGeracao(domainreceber.OrigemPedido, uuid.Nil, uuid.New(), nil, parcelasDeTeste(), uuid.New(), agora)
	esperarErro(t, err, domain.ErrValidacao, "empresa do título")
	_, err = domainreceber.NovaGeracao(domainreceber.OrigemPedido, uuid.New(), uuid.Nil, nil, parcelasDeTeste(), uuid.New(), agora)
	esperarErro(t, err, domain.ErrValidacao, "cliente do título")
}

func TestNovoCancelamentoDeOrigem(t *testing.T) {
	repetido := uuid.New()

	cancelamento, err := domainreceber.NovoCancelamentoDeOrigem(domainreceber.OrigemNotaFiscalDuplicata, []uuid.UUID{repetido, repetido, uuid.New()}, " NF cancelada na SEFAZ ", uuid.New(), agora)
	if err != nil {
		t.Fatalf("NovoCancelamentoDeOrigem falhou: %v", err)
	}
	if len(cancelamento.OrigemIDs) != 2 || cancelamento.Motivo != "NF cancelada na SEFAZ" {
		t.Fatalf("cancelamento = %+v", cancelamento)
	}

	_, err = domainreceber.NovoCancelamentoDeOrigem(domainreceber.OrigemManual, []uuid.UUID{uuid.New()}, "x", uuid.New(), agora)
	esperarErro(t, err, domain.ErrValidacao, "diferente de manual")
	_, err = domainreceber.NovoCancelamentoDeOrigem(domainreceber.OrigemPedido, nil, "x", uuid.New(), agora)
	esperarErro(t, err, domain.ErrValidacao, "pelo menos um id")
	_, err = domainreceber.NovoCancelamentoDeOrigem(domainreceber.OrigemPedido, []uuid.UUID{uuid.New()}, " ", uuid.New(), agora)
	esperarErro(t, err, domain.ErrValidacao, "motivo do cancelamento da origem")
	_, err = domainreceber.NovoCancelamentoDeOrigem(domainreceber.OrigemPedido, []uuid.UUID{uuid.Nil}, "x", uuid.New(), agora)
	esperarErro(t, err, domain.ErrValidacao, "id de origem vazio")
}

func TestAplicarCancelamentoDaOrigem(t *testing.T) {
	cancelamento, err := domainreceber.NovoCancelamentoDeOrigem(domainreceber.OrigemNotaFiscalDuplicata, []uuid.UUID{uuid.New()}, "NF cancelada", uuid.New(), agora)
	if err != nil {
		t.Fatal(err)
	}
	titulo := func(status domainreceber.Status, recebido int64) *domainreceber.Titulo {
		novo := novoTituloDeTeste(t)
		novo.Status, novo.RecebidoCentavos = status, recebido
		return novo
	}

	semMovimento := titulo(domainreceber.StatusEmBanco, 0)
	efeito, historico := semMovimento.AplicarCancelamentoDaOrigem(cancelamento)
	if efeito != domainreceber.EfeitoCancelado || semMovimento.Status != domainreceber.StatusCancelado || *semMovimento.CanceladoMotivo != "NF cancelada" {
		t.Fatalf("sem movimento = %s, %+v", efeito, semMovimento)
	}
	if historico.Evento != domainreceber.EventoCancelado || *historico.DeStatus != domainreceber.StatusEmBanco {
		t.Fatalf("histórico = %+v", historico)
	}

	for nome, movimentado := range map[string]*domainreceber.Titulo{
		"antecipado": titulo(domainreceber.StatusAntecipado, 0),
		"parcial":    titulo(domainreceber.StatusCarteira, 100),
		"recebido":   titulo(domainreceber.StatusRecebido, 150000),
	} {
		anterior := movimentado.Status
		efeito, historico := movimentado.AplicarCancelamentoDaOrigem(cancelamento)
		if efeito != domainreceber.EfeitoMarcado || movimentado.Status != anterior || movimentado.OrigemCanceladaEm == nil {
			t.Fatalf("%s = %s, %+v", nome, efeito, movimentado)
		}
		if historico.Evento != domainreceber.EventoOrigemCancelada || historico.ParaStatus != anterior {
			t.Fatalf("%s histórico = %+v", nome, historico)
		}
		if efeito, historico := movimentado.AplicarCancelamentoDaOrigem(cancelamento); efeito != domainreceber.EfeitoIgnorado || historico != nil {
			t.Fatalf("%s marcado de novo = %s", nome, efeito)
		}
	}

	if efeito, _ := semMovimento.AplicarCancelamentoDaOrigem(cancelamento); efeito != domainreceber.EfeitoIgnorado {
		t.Fatalf("cancelado de novo = %s", efeito)
	}
}
