package caixa_test

import (
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
	"github.com/google/uuid"
)

func (c *cenario) lancamento(contaID uuid.UUID, deslocamento int, valor int64) *domaincaixa.LancamentoExtrato {
	lancamento := &domaincaixa.LancamentoExtrato{
		ID: uuid.New(), ContaBancariaID: contaID, Data: dia(deslocamento), ValorCentavos: valor,
		Descricao: "linha", Situacao: domaincaixa.LancamentoPendente,
	}
	c.conciliacoesRepo.lancamentos[lancamento.ID] = lancamento

	return lancamento
}

func (c *cenario) movimento(tipo domaincaixa.TipoMovimento, contaID uuid.UUID, deslocamento int, valor int64) *domaincaixa.MovimentoSistema {
	movimento := &domaincaixa.MovimentoSistema{
		Tipo: tipo, ID: uuid.New(), ContaBancariaID: contaID, EmpresaID: c.contasRepo.itens[contaID].EmpresaID,
		Data: dia(deslocamento), ValorCentavos: valor, Descricao: "movimento",
	}
	c.conciliacoesRepo.movimentos = append(c.conciliacoesRepo.movimentos, movimento)

	return movimento
}

func referencia(movimento *domaincaixa.MovimentoSistema) portsin.ReferenciaInput {
	return portsin.ReferenciaInput{Tipo: movimento.Tipo.String(), ID: movimento.ID}
}

func TestSugestoesDaConciliacao(t *testing.T) {
	c := novoCenario(t)
	lancamento := c.lancamento(c.sicrediMIA, -5, -1500)
	perto := c.movimento(domaincaixa.MovimentoPagamento, c.sicrediMIA, -4, -1500)
	mesmoDia := c.movimento(domaincaixa.MovimentoPagamento, c.sicrediMIA, -5, -1500)
	c.movimento(domaincaixa.MovimentoPagamento, c.sicrediMIA, -1, -1500)
	c.movimento(domaincaixa.MovimentoPagamento, c.itauRed, -5, -1500)

	sugestoes, err := c.conciliacao.Sugestoes(c.ctx, lancamento.ID, c.financeiroMIA)
	if err != nil {
		t.Fatalf("Sugestoes = %v", err)
	}
	if len(sugestoes) != 2 || sugestoes[0].ID != mesmoDia.ID || sugestoes[1].ID != perto.ID {
		t.Errorf("sugestões = %+v", sugestoes)
	}
	filtro := c.conciliacoesRepo.filtros[0]
	if !filtro.SoPendentes || filtro.ValorCentavos == nil || *filtro.ValorCentavos != -1500 || !filtro.De.Equal(dia(-8)) || !filtro.Ate.Equal(dia(-2)) {
		t.Errorf("filtro das sugestões = %+v", filtro)
	}

	_, err = c.conciliacao.Sugestoes(c.ctx, lancamento.ID, c.financeiroRed)
	esperarErro(t, err, domain.ErrPermissao, "perfil sem acesso")
	_, err = c.conciliacao.Sugestoes(c.ctx, lancamento.ID, c.comum)
	esperarErro(t, err, domain.ErrPermissao, "perfil sem permissão para o financeiro")
	_, err = c.conciliacao.Sugestoes(c.ctx, uuid.New(), c.financeiroMIA)
	esperarErro(t, err, domain.ErrNotFound, "lançamento do extrato não encontrado")
}

func TestConciliarDesfazerEConciliarDeNovo(t *testing.T) {
	c := novoCenario(t)
	lancamento := c.lancamento(c.sicrediMIA, -5, -1500)
	boleto := c.movimento(domaincaixa.MovimentoPagamento, c.sicrediMIA, -5, -1000)
	tarifa := c.movimento(domaincaixa.MovimentoPagamento, c.sicrediMIA, -4, -500)
	daRed := c.movimento(domaincaixa.MovimentoPagamento, c.itauRed, -5, -500)

	_, err := c.conciliacao.Conciliar(c.ctx, portsin.ConciliarInput{UsuarioID: c.financeiroMIA, LancamentoID: lancamento.ID})
	esperarErro(t, err, domain.ErrValidacao, "ao menos um movimento")

	_, err = c.conciliacao.Conciliar(c.ctx, portsin.ConciliarInput{
		UsuarioID: c.financeiroMIA, LancamentoID: lancamento.ID,
		Movimentos: []portsin.ReferenciaInput{{Tipo: "cheque", ID: boleto.ID}},
	})
	esperarErro(t, err, domain.ErrValidacao, "tipo do movimento")

	_, err = c.conciliacao.Conciliar(c.ctx, portsin.ConciliarInput{
		UsuarioID: c.financeiroMIA, LancamentoID: lancamento.ID,
		Movimentos: []portsin.ReferenciaInput{referencia(boleto), referencia(daRed)},
	})
	esperarErro(t, err, domain.ErrValidacao, "movimento não encontrado nesta conta")

	_, err = c.conciliacao.Conciliar(c.ctx, portsin.ConciliarInput{
		UsuarioID: c.financeiroMIA, LancamentoID: lancamento.ID,
		Movimentos: []portsin.ReferenciaInput{referencia(boleto), referencia(boleto)},
	})
	esperarErro(t, err, domain.ErrValidacao, "movimento repetido")

	conciliacao, err := c.conciliacao.Conciliar(c.ctx, portsin.ConciliarInput{
		UsuarioID: c.financeiroMIA, LancamentoID: lancamento.ID,
		Movimentos: []portsin.ReferenciaInput{referencia(boleto), referencia(tarifa)},
	})
	if err != nil {
		t.Fatalf("Conciliar = %v", err)
	}
	if conciliacao.Modo != domaincaixa.ConciliacaoManual || len(conciliacao.Itens) != 2 || conciliacao.UsuarioID != c.financeiroMIA {
		t.Errorf("conciliação = %+v", conciliacao)
	}

	_, err = c.conciliacao.Conciliar(c.ctx, portsin.ConciliarInput{
		UsuarioID: c.financeiroMIA, LancamentoID: lancamento.ID,
		Movimentos: []portsin.ReferenciaInput{referencia(boleto), referencia(tarifa)},
	})
	esperarErro(t, err, domain.ErrConflito, "já conciliado")

	_, err = c.conciliacao.Obter(c.ctx, conciliacao.ID, c.financeiroRed)
	esperarErro(t, err, domain.ErrPermissao, "perfil sem acesso")
	_, err = c.conciliacao.Desfazer(c.ctx, portsin.DesfazerConciliacaoInput{UsuarioID: c.financeiroMIA, ConciliacaoID: conciliacao.ID})
	esperarErro(t, err, domain.ErrValidacao, "motivo")

	desfeita, err := c.conciliacao.Desfazer(c.ctx, portsin.DesfazerConciliacaoInput{
		UsuarioID: c.financeiroMIA, ConciliacaoID: conciliacao.ID, Motivo: "tarifa era de outro dia",
	})
	if err != nil || desfeita.Vigente || *desfeita.MotivoDesfazer != "tarifa era de outro dia" {
		t.Fatalf("Desfazer = %+v, %v", desfeita, err)
	}
	_, err = c.conciliacao.Desfazer(c.ctx, portsin.DesfazerConciliacaoInput{
		UsuarioID: c.financeiroMIA, ConciliacaoID: conciliacao.ID, Motivo: "de novo",
	})
	esperarErro(t, err, domain.ErrConflito, "já desfeita")
	_, err = c.conciliacao.Obter(c.ctx, uuid.New(), c.financeiroMIA)
	esperarErro(t, err, domain.ErrNotFound, "conciliação não encontrada")

	outroBoleto := c.movimento(domaincaixa.MovimentoPagamento, c.sicrediMIA, -6, -500)
	if _, err := c.conciliacao.Conciliar(c.ctx, portsin.ConciliarInput{
		UsuarioID: c.financeiroMIA, LancamentoID: lancamento.ID,
		Movimentos: []portsin.ReferenciaInput{referencia(boleto), referencia(outroBoleto)},
	}); err != nil {
		t.Errorf("conciliar de novo depois de desfazer = %v", err)
	}
}

func TestConciliacaoAutomaticaSoNosParesUnicos(t *testing.T) {
	c := novoCenario(t)
	unico := c.lancamento(c.sicrediMIA, -5, -1000)
	c.movimento(domaincaixa.MovimentoPagamento, c.sicrediMIA, -5, -1000)
	c.lancamento(c.sicrediMIA, -4, 2000)
	c.movimento(domaincaixa.MovimentoRecebimento, c.sicrediMIA, -4, 2000)
	c.movimento(domaincaixa.MovimentoAntecipacao, c.sicrediMIA, -4, 2000)
	c.lancamento(c.sicrediMIA, -3, -12)
	c.lancamento(c.sicrediMIA, -40, -7)
	c.movimento(domaincaixa.MovimentoPagamento, c.sicrediMIA, -40, -7)

	resultado, err := c.conciliacao.ConciliarAutomatico(c.ctx, portsin.ConciliarAutomaticoInput{
		UsuarioID: c.financeiroMIA, ContaBancariaID: c.sicrediMIA, De: dia(-10), Ate: dia(0),
	})
	if err != nil {
		t.Fatalf("ConciliarAutomatico = %v", err)
	}
	if resultado.Conciliadas != 1 || resultado.Ambiguos != 1 || resultado.SemPar != 1 || resultado.Falhas != 0 || resultado.RestamLinhas {
		t.Errorf("resultado = %+v", resultado)
	}
	if c.conciliacoesRepo.lancamentos[unico.ID].ConciliacaoID == nil {
		t.Error("o par único deveria estar conciliado")
	}
	for _, conciliacao := range c.conciliacoesRepo.conciliacoes {
		if conciliacao.Modo != domaincaixa.ConciliacaoAutomatica {
			t.Errorf("modo = %s", conciliacao.Modo)
		}
	}

	c.conciliacoesRepo.falharCriar = domain.ErroConflito("concorrência")
	c.lancamento(c.sicrediMIA, -2, -55)
	c.movimento(domaincaixa.MovimentoPagamento, c.sicrediMIA, -2, -55)
	resultado, err = c.conciliacao.ConciliarAutomatico(c.ctx, portsin.ConciliarAutomaticoInput{
		UsuarioID: c.financeiroMIA, ContaBancariaID: c.sicrediMIA, De: dia(-10), Ate: dia(0),
	})
	if err != nil || resultado.Falhas != 1 || resultado.Conciliadas != 0 {
		t.Errorf("com conflito = %+v, %v", resultado, err)
	}

	_, err = c.conciliacao.ConciliarAutomatico(c.ctx, portsin.ConciliarAutomaticoInput{UsuarioID: c.financeiroMIA, ContaBancariaID: c.sicrediMIA})
	esperarErro(t, err, domain.ErrValidacao, "informe o período")
	_, err = c.conciliacao.ConciliarAutomatico(c.ctx, portsin.ConciliarAutomaticoInput{
		UsuarioID: c.financeiroMIA, ContaBancariaID: c.sicrediMIA, De: dia(-100), Ate: dia(0),
	})
	esperarErro(t, err, domain.ErrValidacao, "no máximo 92 dias")
	_, err = c.conciliacao.ConciliarAutomatico(c.ctx, portsin.ConciliarAutomaticoInput{
		UsuarioID: c.financeiroRed, ContaBancariaID: c.sicrediMIA, De: dia(-10), Ate: dia(0),
	})
	esperarErro(t, err, domain.ErrPermissao, "perfil sem acesso")
}

func TestConciliacaoAutomaticaLimitaAsLinhasSemCortarODia(t *testing.T) {
	c := novoCenario(t)
	for range 400 {
		c.lancamento(c.sicrediMIA, -4, -1)
	}
	for range 200 {
		c.lancamento(c.sicrediMIA, -3, -1)
	}
	c.movimento(domaincaixa.MovimentoPagamento, c.sicrediMIA, -3, -1)

	resultado, err := c.conciliacao.ConciliarAutomatico(c.ctx, portsin.ConciliarAutomaticoInput{
		UsuarioID: c.financeiroMIA, ContaBancariaID: c.sicrediMIA, De: dia(-10), Ate: dia(0),
	})
	if err != nil || !resultado.RestamLinhas || resultado.SemPar != 400 || resultado.Conciliadas != 0 || resultado.Ambiguos != 0 {
		t.Fatalf("resultado = %+v, %v", resultado, err)
	}
	if filtro := c.conciliacoesRepo.filtros[len(c.conciliacoesRepo.filtros)-1]; !filtro.Ate.Equal(dia(-4)) {
		t.Errorf("movimentos consultados até %v, esperado o último dia completo", filtro.Ate)
	}

	for range 501 {
		c.lancamento(c.sicrediMIA, -6, -2)
	}
	_, err = c.conciliacao.ConciliarAutomatico(c.ctx, portsin.ConciliarAutomaticoInput{
		UsuarioID: c.financeiroMIA, ContaBancariaID: c.sicrediMIA, De: dia(-10), Ate: dia(0),
	})
	esperarErro(t, err, domain.ErrValidacao, "mais de 500 lançamentos pendentes no mesmo dia")
}

func TestPendenciasEPagamentosSemConta(t *testing.T) {
	c := novoCenario(t)
	c.movimento(domaincaixa.MovimentoPagamento, c.sicrediMIA, -3, -10)

	pendencias, err := c.conciliacao.Pendencias(c.ctx, portsin.PendenciasInput{
		UsuarioID: c.financeiroMIA, ContaBancariaID: c.sicrediMIA, Filtro: domain.PaginacaoFiltro{Page: 2, Size: 10},
	})
	if err != nil || len(pendencias) != 1 {
		t.Errorf("pendências = %+v, %v", pendencias, err)
	}
	filtro := c.conciliacoesRepo.filtros[len(c.conciliacoesRepo.filtros)-1]
	if !filtro.SoPendentes || filtro.Limite != 10 || filtro.Deslocamento != 10 {
		t.Errorf("filtro das pendências = %+v", filtro)
	}
	_, err = c.conciliacao.Pendencias(c.ctx, portsin.PendenciasInput{UsuarioID: c.financeiroRed, ContaBancariaID: c.sicrediMIA})
	esperarErro(t, err, domain.ErrPermissao, "perfil sem acesso")

	c.conciliacoesRepo.semConta = []*domaincaixa.MovimentoSistema{
		{Tipo: domaincaixa.MovimentoPagamento, ID: uuid.New(), EmpresaID: c.mia, ValorCentavos: -10},
		{Tipo: domaincaixa.MovimentoPagamento, ID: uuid.New(), EmpresaID: c.red, ValorCentavos: -20},
	}
	daRed, err := c.conciliacao.PagamentosSemConta(c.ctx, portsin.PagamentosSemContaInput{UsuarioID: c.financeiroRed})
	if err != nil || len(daRed) != 1 || daRed[0].EmpresaID != c.red {
		t.Errorf("pagamentos sem conta da Red = %+v, %v", daRed, err)
	}
	_, err = c.conciliacao.PagamentosSemConta(c.ctx, portsin.PagamentosSemContaInput{UsuarioID: c.financeiroRed, Empresa: &c.mia})
	esperarErro(t, err, domain.ErrPermissao, "perfil sem acesso")
}

func TestDesfazerTransferenciaExigeAcessoAOutraConta(t *testing.T) {
	c := novoCenario(t)
	linha := c.lancamento(c.sicrediMIA, -2, -1000)
	conciliacao := &domaincaixa.Conciliacao{
		ID: uuid.New(), ExtratoLancamentoID: linha.ID, ContaBancariaID: c.sicrediMIA, Modo: domaincaixa.ConciliacaoTransferencia,
		Vigente: true, ContraparteID: &c.itauRed,
		Itens: []domaincaixa.ItemConciliacao{{Tipo: domaincaixa.MovimentoTransferenciaSaida, AlvoID: uuid.New(), ValorCentavos: -1000}},
	}
	c.conciliacoesRepo.conciliacoes[conciliacao.ID] = conciliacao
	linha.ConciliacaoID = &conciliacao.ID

	_, err := c.conciliacao.Desfazer(c.ctx, portsin.DesfazerConciliacaoInput{
		UsuarioID: c.financeiroMIA, ConciliacaoID: conciliacao.ID, Motivo: "era aplicação",
	})
	esperarErro(t, err, domain.ErrPermissao, "perfil sem acesso")

	if _, err := c.conciliacao.Desfazer(c.ctx, portsin.DesfazerConciliacaoInput{
		UsuarioID: c.administrador, ConciliacaoID: conciliacao.ID, Motivo: "era aplicação",
	}); err != nil {
		t.Errorf("administrador desfaz a transferência = %v", err)
	}
}
