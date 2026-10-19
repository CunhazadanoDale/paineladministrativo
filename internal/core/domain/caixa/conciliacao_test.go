package caixa_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	"github.com/google/uuid"
)

func lancamentoDe(contaID uuid.UUID, dia time.Time, valor int64) *domaincaixa.LancamentoExtrato {
	return &domaincaixa.LancamentoExtrato{ID: uuid.New(), ContaBancariaID: contaID, Data: dia, ValorCentavos: valor}
}

func movimentoDe(tipo domaincaixa.TipoMovimento, contaID uuid.UUID, dia time.Time, valor int64) *domaincaixa.MovimentoSistema {
	return &domaincaixa.MovimentoSistema{Tipo: tipo, ID: uuid.New(), ContaBancariaID: contaID, Data: dia, ValorCentavos: valor}
}

func conciliar(lancamento *domaincaixa.LancamentoExtrato, movimentos ...*domaincaixa.MovimentoSistema) (*domaincaixa.Conciliacao, error) {
	return domaincaixa.NovaConciliacao(domaincaixa.DadosConciliacao{
		Lancamento: lancamento,
		Movimentos: movimentos,
		Modo:       domaincaixa.ConciliacaoManual,
		UsuarioID:  uuid.New(),
		Agora:      agoraDoTeste,
	})
}

func TestNovaConciliacaoSomaVariosMovimentos(t *testing.T) {
	conta := uuid.New()
	lancamento := lancamentoDe(conta, data(2026, 10, 10), -150000)

	conciliacao, err := conciliar(lancamento,
		movimentoDe(domaincaixa.MovimentoPagamento, conta, data(2026, 10, 7), -100000),
		movimentoDe(domaincaixa.MovimentoPagamento, conta, data(2026, 10, 13), -50000),
	)
	if err != nil {
		t.Fatalf("NovaConciliacao = %v", err)
	}
	if !conciliacao.Vigente || len(conciliacao.Itens) != 2 || conciliacao.ExtratoLancamentoID != lancamento.ID ||
		conciliacao.ContaBancariaID != conta || conciliacao.Modo != domaincaixa.ConciliacaoManual {
		t.Errorf("conciliação = %+v", conciliacao)
	}
	if conciliacao.Invalidada() {
		t.Error("conciliação nova não está invalidada")
	}
	conciliacao.Itens[1].Estornado = true
	if !conciliacao.Invalidada() {
		t.Error("item estornado invalida a conciliação")
	}
}

func TestNovaConciliacaoRecusaMovimentosIncompativeis(t *testing.T) {
	conta := uuid.New()
	dia := data(2026, 10, 10)
	novoLancamento := func() *domaincaixa.LancamentoExtrato { return lancamentoDe(conta, dia, -1000) }
	pagamento := func(valor int64) *domaincaixa.MovimentoSistema {
		return movimentoDe(domaincaixa.MovimentoPagamento, conta, dia, valor)
	}

	estornado := pagamento(-1000)
	estornado.Estornado = true
	jaConciliado := pagamento(-1000)
	outraConciliacao := uuid.New()
	jaConciliado.ConciliacaoID = &outraConciliacao
	repetido := pagamento(-500)
	conciliadoAntes := novoLancamento()
	conciliadoAntes.ConciliacaoID = &outraConciliacao

	casos := []struct {
		nome       string
		lancamento *domaincaixa.LancamentoExtrato
		movimentos []*domaincaixa.MovimentoSistema
		sentinela  error
		trecho     string
	}{
		{"sem movimentos", novoLancamento(), nil, domain.ErrValidacao, "ao menos um movimento"},
		{"outra conta", novoLancamento(), []*domaincaixa.MovimentoSistema{movimentoDe(domaincaixa.MovimentoPagamento, uuid.New(), dia, -1000)}, domain.ErrValidacao, "outra conta"},
		{"estornado", novoLancamento(), []*domaincaixa.MovimentoSistema{estornado}, domain.ErrValidacao, "estornado"},
		{"já conciliado", novoLancamento(), []*domaincaixa.MovimentoSistema{jaConciliado}, domain.ErrConflito, "já conciliado com outro lançamento"},
		{"sinal trocado", novoLancamento(), []*domaincaixa.MovimentoSistema{movimentoDe(domaincaixa.MovimentoRecebimento, conta, dia, 1000)}, domain.ErrValidacao, "só concilia com crédito"},
		{"fora da janela", novoLancamento(), []*domaincaixa.MovimentoSistema{movimentoDe(domaincaixa.MovimentoPagamento, conta, data(2026, 10, 14), -1000)}, domain.ErrValidacao, "fora da janela de 3 dias"},
		{"soma diferente", novoLancamento(), []*domaincaixa.MovimentoSistema{pagamento(-999)}, domain.ErrValidacao, "a soma dos movimentos (-9,99) é diferente do valor do lançamento (-10,00)"},
		{"repetido", novoLancamento(), []*domaincaixa.MovimentoSistema{repetido, repetido}, domain.ErrValidacao, "movimento repetido"},
		{"lançamento já conciliado", conciliadoAntes, []*domaincaixa.MovimentoSistema{pagamento(-1000)}, domain.ErrConflito, "desfaça a conciliação antes"},
	}

	for _, caso := range casos {
		_, err := conciliar(caso.lancamento, caso.movimentos...)
		if !errors.Is(err, caso.sentinela) || !strings.Contains(err.Error(), caso.trecho) {
			t.Errorf("%s: erro = %v, esperado %v com %q", caso.nome, err, caso.sentinela, caso.trecho)
		}
	}

	muitos := make([]*domaincaixa.MovimentoSistema, 0, domaincaixa.MaximoDeMovimentosPorConciliacao+1)
	for range domaincaixa.MaximoDeMovimentosPorConciliacao + 1 {
		muitos = append(muitos, pagamento(-1))
	}
	if _, err := conciliar(lancamentoDe(conta, dia, int64(-len(muitos))), muitos...); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("movimentos demais = %v", err)
	}
}

func TestDesfazerConciliacao(t *testing.T) {
	conta := uuid.New()
	conciliacao, err := conciliar(lancamentoDe(conta, data(2026, 10, 10), 500),
		movimentoDe(domaincaixa.MovimentoRecebimento, conta, data(2026, 10, 10), 500))
	if err != nil {
		t.Fatalf("NovaConciliacao = %v", err)
	}

	usuario := uuid.New()
	if err := conciliacao.Desfazer(usuario, "  ", agoraDoTeste); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("sem motivo = %v", err)
	}
	if err := conciliacao.Desfazer(usuario, " par errado ", agoraDoTeste); err != nil {
		t.Fatalf("Desfazer = %v", err)
	}
	if conciliacao.Vigente || *conciliacao.MotivoDesfazer != "par errado" || *conciliacao.DesfeitaPor != usuario || conciliacao.Invalidada() {
		t.Errorf("desfeita = %+v", conciliacao)
	}
	if err := conciliacao.Desfazer(usuario, "de novo", agoraDoTeste); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("desfazer de novo = %v", err)
	}
}

func TestParesExatosSoUnicosNosDoisSentidos(t *testing.T) {
	conta := uuid.New()
	dia := data(2026, 10, 10)

	unico := lancamentoDe(conta, dia, -100)
	pagamentoUnico := movimentoDe(domaincaixa.MovimentoPagamento, conta, dia, -100)

	duasLinhasA := lancamentoDe(conta, dia, -200)
	duasLinhasB := lancamentoDe(conta, dia, -200)
	pagamentoDeDuas := movimentoDe(domaincaixa.MovimentoPagamento, conta, dia, -200)

	linhaDeDois := lancamentoDe(conta, dia, 300)
	recebimentoA := movimentoDe(domaincaixa.MovimentoRecebimento, conta, dia, 300)
	recebimentoB := movimentoDe(domaincaixa.MovimentoRecebimento, conta, dia, 300)

	semPar := lancamentoDe(conta, dia, -400)
	diaSeguinte := movimentoDe(domaincaixa.MovimentoPagamento, conta, data(2026, 10, 11), -400)

	conciliada := lancamentoDe(conta, dia, -100)
	jaConciliada := uuid.New()
	conciliada.ConciliacaoID = &jaConciliada

	resultado := domaincaixa.ParesExatos(
		[]*domaincaixa.LancamentoExtrato{unico, duasLinhasA, duasLinhasB, linhaDeDois, semPar, conciliada},
		[]*domaincaixa.MovimentoSistema{pagamentoUnico, pagamentoDeDuas, recebimentoA, recebimentoB, diaSeguinte},
	)

	if len(resultado.Pares) != 1 || resultado.Pares[0].Lancamento != unico || resultado.Pares[0].Movimento != pagamentoUnico {
		t.Errorf("pares = %+v", resultado.Pares)
	}
	if resultado.Ambiguos != 3 || resultado.SemPar != 1 {
		t.Errorf("ambíguos %d, sem par %d", resultado.Ambiguos, resultado.SemPar)
	}
}

func TestOrdenarSugestoesPelaDistancia(t *testing.T) {
	conta := uuid.New()
	lancamento := lancamentoDe(conta, data(2026, 10, 10), -100)

	longe := movimentoDe(domaincaixa.MovimentoPagamento, conta, data(2026, 10, 7), -100)
	perto := movimentoDe(domaincaixa.MovimentoPagamento, conta, data(2026, 10, 11), -100)
	mesmoDia := movimentoDe(domaincaixa.MovimentoPagamento, conta, data(2026, 10, 10), -100)
	foraDaJanela := movimentoDe(domaincaixa.MovimentoPagamento, conta, data(2026, 10, 14), -100)
	outroValor := movimentoDe(domaincaixa.MovimentoPagamento, conta, data(2026, 10, 10), -101)
	estornado := movimentoDe(domaincaixa.MovimentoPagamento, conta, data(2026, 10, 10), -100)
	estornado.Estornado = true

	sugestoes := domaincaixa.OrdenarSugestoes(lancamento, []*domaincaixa.MovimentoSistema{longe, perto, foraDaJanela, outroValor, estornado, mesmoDia})
	if len(sugestoes) != 3 || sugestoes[0] != mesmoDia || sugestoes[1] != perto || sugestoes[2] != longe {
		t.Errorf("sugestões = %+v", sugestoes)
	}
}

func TestFormatarCentavos(t *testing.T) {
	casos := map[int64]string{0: "0,00", 7: "0,07", -150000: "-1500,00", 123456: "1234,56"}
	for centavos, esperado := range casos {
		if obtido := domaincaixa.FormatarCentavos(centavos); obtido != esperado {
			t.Errorf("FormatarCentavos(%d) = %q, esperado %q", centavos, obtido, esperado)
		}
	}
}
