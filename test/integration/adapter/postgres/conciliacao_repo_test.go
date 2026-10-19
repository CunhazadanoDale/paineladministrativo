package postgres_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portscaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/caixa"
	"github.com/google/uuid"
)

type cenarioBancoConciliacao struct {
	*cenarioBancoTitulos
	conciliacoes *postgres.ConciliacaoRepository
	extratos     *postgres.ExtratoRepository
	abertura     *domaincaixa.Abertura
	diaDoPago    time.Time
}

func novoCenarioConciliacao(t *testing.T) *cenarioBancoConciliacao {
	t.Helper()

	base := novoCenarioTitulos(t)
	c := &cenarioBancoConciliacao{
		cenarioBancoTitulos: base,
		conciliacoes:        postgres.NewConciliacaoRepository(base.banco),
		extratos:            postgres.NewExtratoRepository(base.banco),
		diaDoPago:           domaincaixa.Hoje(time.Now().UTC().Add(-time.Hour)),
	}

	abertura, err := domaincaixa.NovaAbertura(domaincaixa.DadosAbertura{
		ContaBancariaID: c.contaID, DataCorte: diaDoTitulo(10, 1), SaldoCentavos: 0, UsuarioID: c.solicitanteID,
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("NovaAbertura = %v", err)
	}
	if err := postgres.NewCaixaAberturaRepository(base.banco).Criar(c.ctx, abertura); err != nil {
		t.Fatalf("criar abertura = %v", err)
	}
	c.abertura = abertura

	return c
}

func (c *cenarioBancoConciliacao) novoArquivoDoCaixa(t *testing.T) uuid.UUID {
	t.Helper()

	arquivoID := uuid.New()
	if _, err := postgres.NewArquivoRepository(c.banco).Criar(c.ctx, &domainsolicitacao.Arquivo{
		ID: arquivoID, ProprietarioID: c.solicitanteID, Nome: "extrato.ofx",
		Chave: c.solicitanteID.String() + "/" + arquivoID.String() + "/extrato.ofx", ContentType: "application/x-ofx",
		Tamanho: 100, CriadoEm: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("criação do arquivo falhou: %v", err)
	}

	return arquivoID
}

func (c *cenarioBancoConciliacao) importarLinhas(t *testing.T, transacoes ...domaincaixa.TransacaoOFX) *domaincaixa.ExtratoPreparado {
	t.Helper()

	preparado, err := domaincaixa.PrepararImportacao(domaincaixa.DadosImportacao{
		Extrato: &domaincaixa.ExtratoOFX{Transacoes: transacoes}, ContaBancariaID: c.contaID, ArquivoID: c.novoArquivoDoCaixa(t),
		UsuarioID: c.solicitanteID, Abertura: c.abertura, Agora: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("PrepararImportacao = %v", err)
	}
	if err := c.extratos.Importar(c.ctx, preparado, c.abertura.Versao); err != nil {
		t.Fatalf("Importar = %v", err)
	}

	return preparado
}

func (c *cenarioBancoConciliacao) movimentos(t *testing.T, filtro portscaixa.MovimentoFiltro) map[domaincaixa.TipoMovimento]*domaincaixa.MovimentoSistema {
	t.Helper()

	filtro.ContaBancariaID = c.contaID
	movimentos, err := c.conciliacoes.Movimentos(c.ctx, filtro)
	if err != nil {
		t.Fatalf("Movimentos = %v", err)
	}

	porTipo := map[domaincaixa.TipoMovimento]*domaincaixa.MovimentoSistema{}
	for _, movimento := range movimentos {
		porTipo[movimento.Tipo] = movimento
	}

	return porTipo
}

func (c *cenarioBancoConciliacao) lancamento(t *testing.T, id uuid.UUID) *domaincaixa.LancamentoExtrato {
	t.Helper()

	lancamento, err := c.conciliacoes.ObterLancamento(c.ctx, id)
	if err != nil || lancamento == nil {
		t.Fatalf("ObterLancamento = %+v, %v", lancamento, err)
	}

	return lancamento
}

func (c *cenarioBancoConciliacao) conciliar(t *testing.T, lancamento *domaincaixa.LancamentoExtrato, movimentos ...*domaincaixa.MovimentoSistema) *domaincaixa.Conciliacao {
	t.Helper()

	conciliacao, err := domaincaixa.NovaConciliacao(domaincaixa.DadosConciliacao{
		Lancamento: lancamento, Movimentos: movimentos, Modo: domaincaixa.ConciliacaoManual,
		UsuarioID: c.solicitanteID, Agora: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("NovaConciliacao = %v", err)
	}

	return conciliacao
}

func transacaoNoDia(fitid string, dia time.Time, valor int64) domaincaixa.TransacaoOFX {
	return domaincaixa.TransacaoOFX{FITID: fitid, Data: dia, ValorCentavos: valor, Descricao: fitid}
}

func TestConciliacaoLeMovimentosDoM7EDoM9(t *testing.T) {
	c := novoCenarioConciliacao(t)

	solicitacao := c.aprovadaClassificada(t)
	pagamento, err := c.pagar(t, solicitacao, c.contaID, nil)
	if err != nil {
		t.Fatalf("pagar = %v", err)
	}
	titulo := c.novoTitulo(t, "NF-1", diaDoTitulo(11, 10))
	recebimento, err := c.receber(t, titulo, 100000, nil)
	if err != nil {
		t.Fatalf("receber = %v", err)
	}
	bordero := c.novoBordero(t, c.novoTitulo(t, "NF-2", diaDoTitulo(11, 20)))

	movimentos := c.movimentos(t, portscaixa.MovimentoFiltro{})
	if len(movimentos) != 3 {
		t.Fatalf("movimentos = %+v", movimentos)
	}
	pago := movimentos[domaincaixa.MovimentoPagamento]
	if pago.ID != pagamento.ID || pago.ValorCentavos != -148500 || !pago.Data.Equal(c.diaDoPago) || pago.OrigemID != solicitacao.ID ||
		pago.EmpresaID != c.empresaID || pago.Estornado || pago.ConciliacaoID != nil {
		t.Errorf("pagamento = %+v", pago)
	}
	recebido := movimentos[domaincaixa.MovimentoRecebimento]
	if recebido.ID != recebimento.ID || recebido.ValorCentavos != 100050 || !recebido.Data.Equal(diaDoTitulo(10, 15)) ||
		recebido.Descricao != "Construtora Alfa - título NF-1/1" || recebido.OrigemID != titulo.ID {
		t.Errorf("recebimento = %+v", recebido)
	}
	antecipado := movimentos[domaincaixa.MovimentoAntecipacao]
	if antecipado.ID != bordero.ID || antecipado.ValorCentavos != 149000 || !antecipado.Data.Equal(diaDoTitulo(10, 14)) ||
		antecipado.Descricao != "Borderô B-1 - Cimentos do Vale" {
		t.Errorf("antecipação = %+v", antecipado)
	}

	valor := int64(100050)
	if soRecebimento := c.movimentos(t, portscaixa.MovimentoFiltro{ValorCentavos: &valor}); len(soRecebimento) != 1 {
		t.Errorf("filtro por valor = %+v", soRecebimento)
	}
	de, ate := diaDoTitulo(10, 14), diaDoTitulo(10, 15)
	doPeriodo := c.movimentos(t, portscaixa.MovimentoFiltro{De: &de, Ate: &ate})
	pagoNoPeriodo := !c.diaDoPago.Before(de) && !c.diaDoPago.After(ate)
	if doPeriodo[domaincaixa.MovimentoRecebimento] == nil || doPeriodo[domaincaixa.MovimentoAntecipacao] == nil ||
		(doPeriodo[domaincaixa.MovimentoPagamento] != nil) != pagoNoPeriodo {
		t.Errorf("filtro por período = %+v", doPeriodo)
	}

	porReferencia, err := c.conciliacoes.MovimentosPorReferencia(c.ctx, c.contaID, []domaincaixa.ReferenciaMovimento{
		recebido.Referencia(), antecipado.Referencia(), {Tipo: domaincaixa.MovimentoPagamento, ID: uuid.New()},
	})
	if err != nil || len(porReferencia) != 2 {
		t.Errorf("MovimentosPorReferencia = %+v, %v", porReferencia, err)
	}

	if _, err := c.banco.ExecContext(c.ctx, `UPDATE caixa_abertura SET data_corte = $1 WHERE conta_bancaria_id = $2`,
		diaDoTitulo(10, 14), c.contaID); err != nil {
		t.Fatalf("mover o corte falhou: %v", err)
	}
	if depoisDoCorte := c.movimentos(t, portscaixa.MovimentoFiltro{}); depoisDoCorte[domaincaixa.MovimentoAntecipacao] != nil {
		t.Errorf("borderô no dia do corte não entra na conciliação: %+v", depoisDoCorte)
	}
}

func TestConciliacaoGravaInvalidaPorEstornoEDesfaz(t *testing.T) {
	c := novoCenarioConciliacao(t)

	solicitacao := c.aprovadaClassificada(t)
	pagamento, err := c.pagar(t, solicitacao, c.contaID, nil)
	if err != nil {
		t.Fatalf("pagar = %v", err)
	}
	preparado := c.importarLinhas(t,
		transacaoNoDia("P", c.diaDoPago, -148500),
		transacaoNoDia("X", c.diaDoPago, -148500),
	)
	linha := c.lancamento(t, preparado.Lancamentos[0].ID)
	outraLinha := c.lancamento(t, preparado.Lancamentos[1].ID)
	if linha.Situacao != domaincaixa.LancamentoPendente || linha.ConciliacaoID != nil {
		t.Errorf("linha antes = %+v", linha)
	}

	pago := c.movimentos(t, portscaixa.MovimentoFiltro{})[domaincaixa.MovimentoPagamento]
	conciliacao := c.conciliar(t, linha, pago)
	if err := c.conciliacoes.Criar(c.ctx, conciliacao); err != nil {
		t.Fatalf("Criar = %v", err)
	}

	if conciliada := c.lancamento(t, linha.ID); conciliada.Situacao != domaincaixa.LancamentoConciliado || *conciliada.ConciliacaoID != conciliacao.ID {
		t.Errorf("linha conciliada = %+v", conciliada)
	}
	if pendentes := c.movimentos(t, portscaixa.MovimentoFiltro{SoPendentes: true}); len(pendentes) != 0 {
		t.Errorf("pendentes depois de conciliar = %+v", pendentes)
	}
	if err := c.conciliacoes.Criar(c.ctx, c.conciliar(t, outraLinha, pago)); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("mesmo pagamento em outra linha = %v", err)
	}
	pagoDeNovo := *pago
	pagoDeNovo.ConciliacaoID = nil
	repetida := c.conciliar(t, outraLinha, &pagoDeNovo)
	repetida.ExtratoLancamentoID = linha.ID
	if err := c.conciliacoes.Criar(c.ctx, repetida); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("segunda conciliação da linha = %v", err)
	}

	pendentes, err := c.extratos.ListarLancamentos(c.ctx, portscaixa.LancamentoFiltro{
		PaginacaoFiltro: domain.PaginacaoFiltro{Page: 1, Size: 10}, ContaBancariaID: c.contaID,
		Situacao: situacaoDoFiltro(domaincaixa.LancamentoPendente),
	})
	if err != nil || len(pendentes) != 1 || pendentes[0].ID != outraLinha.ID {
		t.Errorf("lançamentos pendentes = %+v, %v", pendentes, err)
	}
	if err := c.extratos.DesfazerImportacao(c.ctx, preparado.Importacao); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("desfazer importação conciliada = %v", err)
	}

	if err := c.cenarioBancoClassificacao.estornar(t, solicitacao, pagamento); err != nil {
		t.Fatalf("estornar pagamento = %v", err)
	}
	if invalidada := c.lancamento(t, linha.ID); invalidada.Situacao != domaincaixa.LancamentoInvalidado {
		t.Errorf("linha depois do estorno = %+v", invalidada)
	}
	salva, err := c.conciliacoes.Obter(c.ctx, conciliacao.ID)
	if err != nil || salva == nil || !salva.Invalidada() || len(salva.Itens) != 1 || salva.Itens[0].ValorCentavos != -148500 {
		t.Fatalf("conciliação invalidada = %+v, %v", salva, err)
	}

	if err := salva.Desfazer(c.solicitanteID, "pagamento estornado", time.Now().UTC()); err != nil {
		t.Fatalf("Desfazer no domínio = %v", err)
	}
	if err := c.conciliacoes.Desfazer(c.ctx, salva); err != nil {
		t.Fatalf("Desfazer = %v", err)
	}
	if err := c.conciliacoes.Desfazer(c.ctx, salva); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("desfazer de novo = %v", err)
	}
	desfeita, _ := c.conciliacoes.Obter(c.ctx, conciliacao.ID)
	if desfeita.Vigente || desfeita.MotivoDesfazer == nil || *desfeita.MotivoDesfazer != "pagamento estornado" || desfeita.DesfeitaPor == nil {
		t.Errorf("conciliação desfeita = %+v", desfeita)
	}
	if voltou := c.lancamento(t, linha.ID); voltou.Situacao != domaincaixa.LancamentoPendente || voltou.ConciliacaoID != nil {
		t.Errorf("linha depois de desfazer = %+v", voltou)
	}

	estornado := c.movimentos(t, portscaixa.MovimentoFiltro{})[domaincaixa.MovimentoPagamento]
	if !estornado.Estornado {
		t.Fatalf("pagamento estornado = %+v", estornado)
	}
	estornado.Estornado = false
	if err := c.conciliacoes.Criar(c.ctx, c.conciliar(t, c.lancamento(t, linha.ID), estornado)); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("conciliar pagamento estornado = %v", err)
	}
}

func TestConciliacaoRecebimentoEBorderoEPendencias(t *testing.T) {
	c := novoCenarioConciliacao(t)

	if _, err := c.receber(t, c.novoTitulo(t, "NF-1", diaDoTitulo(11, 10)), 100000, nil); err != nil {
		t.Fatalf("receber = %v", err)
	}
	c.novoBordero(t, c.novoTitulo(t, "NF-2", diaDoTitulo(11, 20)))
	preparado := c.importarLinhas(t,
		transacaoNoDia("R", diaDoTitulo(10, 16), 249050),
		transacaoNoDia("T", diaDoTitulo(10, 16), -1290),
		transacaoNoDia("U", diaDoTitulo(10, 17), -10),
	)

	pendentes, err := c.conciliacoes.LancamentosPendentes(c.ctx, c.contaID, diaDoTitulo(10, 1), diaDoTitulo(10, 16), 10)
	if err != nil || len(pendentes) != 2 {
		t.Fatalf("LancamentosPendentes = %+v, %v", pendentes, err)
	}

	movimentos := c.movimentos(t, portscaixa.MovimentoFiltro{})
	credito := c.lancamento(t, preparado.Lancamentos[0].ID)
	conciliacao := c.conciliar(t, credito, movimentos[domaincaixa.MovimentoRecebimento], movimentos[domaincaixa.MovimentoAntecipacao])
	if err := c.conciliacoes.Criar(c.ctx, conciliacao); err != nil {
		t.Fatalf("Criar = %v", err)
	}

	salva, err := c.conciliacoes.Obter(c.ctx, conciliacao.ID)
	if err != nil || len(salva.Itens) != 2 || salva.Invalidada() || salva.Modo != domaincaixa.ConciliacaoManual {
		t.Errorf("conciliação = %+v, %v", salva, err)
	}
	if restantes, _ := c.conciliacoes.LancamentosPendentes(c.ctx, c.contaID, diaDoTitulo(10, 1), diaDoTitulo(10, 31), 1); len(restantes) != 1 ||
		restantes[0].ValorCentavos != -1290 {
		t.Errorf("pendentes com limite = %+v", restantes)
	}

	saldos, err := postgres.NewCaixaSaldoRepository(c.banco).Saldos(c.ctx, nil)
	if err != nil || len(saldos) != 1 || saldos[0].LancamentosPendentes != 2 {
		t.Errorf("saldos = %+v, %v", saldos, err)
	}
}

func TestPagamentosSemContaBancaria(t *testing.T) {
	c := novoCenarioConciliacao(t)

	comConta := c.aprovadaClassificada(t)
	if _, err := c.pagar(t, comConta, c.contaID, nil); err != nil {
		t.Fatalf("pagar = %v", err)
	}
	semConta := c.aprovadaClassificada(t)
	pagamento, err := c.pagar(t, semConta, c.contaID, nil)
	if err != nil {
		t.Fatalf("pagar = %v", err)
	}
	if _, err := c.banco.ExecContext(c.ctx, `UPDATE pagamento SET conta_bancaria_id = NULL WHERE id = $1`, pagamento.ID); err != nil {
		t.Fatalf("tirar a conta do pagamento falhou: %v", err)
	}

	encontrados, err := c.conciliacoes.PagamentosSemConta(c.ctx, portscaixa.SemContaFiltro{EmpresaIDs: []uuid.UUID{c.empresaID}, Limite: 10})
	if err != nil || len(encontrados) != 1 || encontrados[0].ID != pagamento.ID || encontrados[0].ContaBancariaID != uuid.Nil ||
		encontrados[0].ValorCentavos != -148500 {
		t.Errorf("pagamentos sem conta = %+v, %v", encontrados, err)
	}
	if deOutra, _ := c.conciliacoes.PagamentosSemConta(c.ctx, portscaixa.SemContaFiltro{EmpresaIDs: []uuid.UUID{uuid.New()}, Limite: 10}); len(deOutra) != 0 {
		t.Errorf("pagamentos de outra empresa = %+v", deOutra)
	}
}

func situacaoDoFiltro(situacao domaincaixa.SituacaoLancamento) *domaincaixa.SituacaoLancamento {
	return &situacao
}
