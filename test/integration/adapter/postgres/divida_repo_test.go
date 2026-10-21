package postgres_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domaindivida "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/divida"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsdivida "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/divida"
	"github.com/google/uuid"
)

var (
	_ portsdivida.FornecedorRepository          = (*postgres.FornecedorRepository)(nil)
	_ portsdivida.CategoriaFinanceiraRepository = (*postgres.CategoriaFinanceiraRepository)(nil)
	_ portsdivida.ContaBancariaRepository       = (*postgres.ContaBancariaRepository)(nil)
	_ portsdivida.EmpresaRepository             = (*postgres.EmpresaRepository)(nil)
)

type cenarioBancoDividas struct {
	*cenarioBancoLocais
	dividas     *postgres.DividaRepository
	credorID    uuid.UUID
	amortizacao uuid.UUID
	juros       uuid.UUID
	hoje        time.Time
}

func novoCenarioDividas(t *testing.T) *cenarioBancoDividas {
	t.Helper()

	base := novoCenarioLocais(t)
	documento, err := domaincliente.NovoDocumento("01.181.521/0001-55")
	if err != nil {
		t.Fatalf("documento de teste inválido: %v", err)
	}
	credor, err := domainfornecedor.NovoFornecedor(domainfornecedor.PaisBrasil, documento, domainfornecedor.DadosFornecedor{
		Nome: "Banco Cooperativo Sicredi", MoedaPadrao: domainfornecedor.MoedaReal,
	})
	if err != nil {
		t.Fatalf("credor de teste inválido: %v", err)
	}
	if _, err := postgres.NewFornecedorRepository(base.banco).Criar(base.ctx, credor); err != nil {
		t.Fatalf("criação do credor falhou: %v", err)
	}

	c := &cenarioBancoDividas{
		cenarioBancoLocais: base,
		dividas:            postgres.NewDividaRepository(base.banco),
		credorID:           credor.ID,
		hoje:               domaindivida.Hoje(time.Now()),
	}
	c.amortizacao = c.subcategoriaDeDespesa(t, "Empréstimos", "fora_dre", "Amortização")
	c.juros = c.subcategoriaDeDespesa(t, "Financeiras", "resultado_financeiro", "Juros de empréstimos")

	return c
}

func (c *cenarioBancoDividas) subcategoriaDeDespesa(t *testing.T, nomeRaiz, linha, nome string) uuid.UUID {
	t.Helper()

	categorias := postgres.NewCategoriaFinanceiraRepository(c.banco)
	classificacao, err := domainfinanceiro.NovaClassificacao("despesa", linha)
	if err != nil {
		t.Fatalf("classificação de teste inválida: %v", err)
	}
	raiz, err := domainfinanceiro.NovaCategoriaFinanceira(domainfinanceiro.DadosCategoriaFinanceira{Nome: nomeRaiz}, &classificacao, nil)
	if err != nil {
		t.Fatalf("raiz de teste inválida: %v", err)
	}
	if err := categorias.Criar(c.ctx, raiz); err != nil {
		t.Fatalf("criação da raiz falhou: %v", err)
	}
	sub, err := domainfinanceiro.NovaCategoriaFinanceira(domainfinanceiro.DadosCategoriaFinanceira{Nome: nome}, nil, raiz)
	if err != nil {
		t.Fatalf("subcategoria de teste inválida: %v", err)
	}
	if err := categorias.Criar(c.ctx, sub); err != nil {
		t.Fatalf("criação da subcategoria falhou: %v", err)
	}

	return sub.ID
}

func (c *cenarioBancoDividas) nova(t *testing.T, empresaID uuid.UUID) (*domaindivida.Divida, *domaindivida.Geracao) {
	t.Helper()

	agora := time.Now().UTC()
	divida, err := domaindivida.NovaDivida(empresaID, c.usuarioID, domaindivida.DadosDivida{
		DadosInformativos: domaindivida.DadosInformativos{
			CredorTipo:             domaindivida.CredorBanco,
			Descricao:              "Capital de giro",
			CategoriaAmortizacaoID: c.amortizacao,
			CategoriaJurosID:       c.juros,
		},
		FornecedorID:         c.credorID,
		SaldoInicialCentavos: 300000,
		DataSaldo:            c.hoje.AddDate(0, 0, -1),
		Parcelas: []domaindivida.ParcelaInput{
			{Vencimento: c.hoje, AmortizacaoCentavos: 100000, JurosCentavos: 3000},
			{Vencimento: c.hoje.AddDate(0, 2, 0), AmortizacaoCentavos: 100000, JurosCentavos: 2000},
			{Vencimento: c.hoje.AddDate(0, 4, 0), AmortizacaoCentavos: 100000},
		},
	}, agora)
	if err != nil {
		t.Fatalf("dívida de teste inválida: %v", err)
	}
	geracao, err := divida.GerarContasAte(domaindivida.FimDoMes(c.hoje), c.usuarioID, agora)
	if err != nil {
		t.Fatalf("geração de teste falhou: %v", err)
	}
	if err := c.dividas.Gravar(c.ctx, &portsdivida.Gravacao{
		Divida:       divida,
		Novo:         true,
		ContasAPagar: geracao.Contas,
		Historicos: []*domaindivida.Historico{
			domaindivida.NovoHistorico(divida.ID, c.usuarioID, domaindivida.AcaoCriada, nil, divida.Status, "", agora),
		},
	}); err != nil {
		t.Fatalf("gravação da dívida falhou: %v", err)
	}

	return divida, geracao
}

func (c *cenarioBancoDividas) obter(t *testing.T, id uuid.UUID) *domaindivida.Divida {
	t.Helper()

	divida, err := c.dividas.Obter(c.ctx, id)
	if err != nil || divida == nil {
		t.Fatalf("Obter = %+v, %v", divida, err)
	}

	return divida
}

func (c *cenarioBancoDividas) statusDaConta(t *testing.T, id uuid.UUID) string {
	t.Helper()

	var status string
	if err := c.banco.Get(&status, `SELECT status FROM solicitacao WHERE id = $1`, id); err != nil {
		t.Fatalf("leitura da conta falhou: %v", err)
	}

	return status
}

func TestDividaGravaParcelasEContasNaMesmaTransacao(t *testing.T) {
	c := novoCenarioDividas(t)
	divida, geracao := c.nova(t, c.mia)

	lida := c.obter(t, divida.ID)
	if lida.Versao != 1 || lida.SaldoInicialCentavos != 300000 || !lida.DataSaldo.Equal(c.hoje.AddDate(0, 0, -1)) ||
		lida.CredorTipo != domaindivida.CredorBanco || len(lida.Parcelas) != 3 {
		t.Fatalf("dívida lida = %+v", lida)
	}
	primeira := lida.Parcelas[0]
	if primeira.Status != domaindivida.StatusParcelaGerada || primeira.GeradaEm == nil || primeira.ContaAmortizacao == nil ||
		primeira.ContaJuros == nil || primeira.ContaJuros.Status != domainsolicitacao.StatusPendenteAprovacao {
		t.Fatalf("primeira parcela = %+v", primeira)
	}
	if lida.Parcelas[1].Status != domaindivida.StatusParcelaPrevista || lida.Parcelas[1].ContaJuros != nil {
		t.Fatalf("segunda parcela = %+v", lida.Parcelas[1])
	}
	if len(geracao.Contas) != 2 {
		t.Fatalf("contas geradas = %d", len(geracao.Contas))
	}

	var origens []string
	if err := c.banco.Select(&origens, `SELECT origem FROM solicitacao WHERE origem_id = $1 ORDER BY origem`, primeira.ID); err != nil {
		t.Fatalf("leitura das origens falhou: %v", err)
	}
	if len(origens) != 2 || origens[0] != "divida_amortizacao" || origens[1] != "divida_juros" {
		t.Errorf("origens = %v", origens)
	}

	repetida, _, err := domainsolicitacao.NovaSolicitacaoDeOrigem(domainsolicitacao.DadosSolicitacaoDeOrigem{
		EmpresaID: c.mia, SolicitanteID: c.usuarioID, CategoriaFinanceiraID: c.juros, Origem: domainsolicitacao.OrigemDividaJuros,
		OrigemID: primeira.ID, ValorCentavos: 1, Vencimento: c.hoje, Observacao: "repetida",
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("conta repetida inválida: %v", err)
	}
	err = c.dividas.Gravar(c.ctx, &portsdivida.Gravacao{
		Divida: lida, VersaoLida: 1, StatusAnterior: lida.Status,
		ContasAPagar: []domaindivida.ContaGerada{{Solicitacao: repetida, Historico: domainsolicitacao.NovoHistorico(repetida.ID, c.usuarioID, nil, repetida.Status, domainsolicitacao.EventoCriada, "")}},
	})
	if !errors.Is(err, domain.ErrConflito) {
		t.Fatalf("conta repetida: %v", err)
	}
	if depois := c.obter(t, divida.ID); depois.Versao != 1 {
		t.Errorf("versão depois do conflito = %d", depois.Versao)
	}
}

func TestDividaGuardaDeVersao(t *testing.T) {
	c := novoCenarioDividas(t)
	divida, _ := c.nova(t, c.mia)
	lida := c.obter(t, divida.ID)

	if err := lida.Alterar(lida.DadosInformativosAtuais(), time.Now().UTC()); err != nil {
		t.Fatalf("Alterar: %v", err)
	}
	if err := c.dividas.Gravar(c.ctx, &portsdivida.Gravacao{Divida: lida, VersaoLida: 1, StatusAnterior: domaindivida.StatusAtiva}); err != nil {
		t.Fatalf("primeira gravação: %v", err)
	}
	if lida.Versao != 2 {
		t.Fatalf("versão = %d", lida.Versao)
	}
	if err := c.dividas.Gravar(c.ctx, &portsdivida.Gravacao{Divida: lida, VersaoLida: 1, StatusAnterior: domaindivida.StatusAtiva}); !errors.Is(err, domain.ErrConflito) {
		t.Fatalf("versão antiga: %v", err)
	}
	fantasma := *lida
	fantasma.ID = uuid.New()
	if err := c.dividas.Gravar(c.ctx, &portsdivida.Gravacao{Divida: &fantasma, VersaoLida: 2, StatusAnterior: domaindivida.StatusAtiva}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("dívida inexistente: %v", err)
	}
}

func TestDividaCanceladaCancelaContasAbertasComHistorico(t *testing.T) {
	c := novoCenarioDividas(t)
	divida, _ := c.nova(t, c.mia)
	lida := c.obter(t, divida.ID)
	agora := time.Now().UTC()

	parcelas, contas, err := lida.Cancelar("lançada em duplicidade", agora)
	if err != nil {
		t.Fatalf("Cancelar: %v", err)
	}
	gravacao := &portsdivida.Gravacao{
		Divida: lida, VersaoLida: 1, StatusAnterior: domaindivida.StatusAtiva, ParcelasAlteradas: parcelas,
		Historicos: []*domaindivida.Historico{
			domaindivida.NovoHistorico(lida.ID, c.usuarioID, domaindivida.AcaoCancelada, nil, lida.Status, "", agora),
		},
	}
	for _, conta := range contas {
		anterior := conta.StatusAnterior
		gravacao.ContasCanceladas = append(gravacao.ContasCanceladas, portsdivida.ContaCancelada{
			ContaCancelada: conta,
			Historico: domainsolicitacao.NovoHistorico(conta.SolicitacaoID, c.usuarioID, &anterior, domainsolicitacao.StatusCancelado,
				domainsolicitacao.EventoCancelada, "dívida cancelada"),
		})
	}
	if err := c.dividas.Gravar(c.ctx, gravacao); err != nil {
		t.Fatalf("gravação do cancelamento: %v", err)
	}

	depois := c.obter(t, divida.ID)
	if depois.Status != domaindivida.StatusCancelada || depois.Motivo == nil || depois.Versao != 2 {
		t.Fatalf("dívida cancelada = %+v", depois)
	}
	for _, parcela := range depois.Parcelas {
		if parcela.Status != domaindivida.StatusParcelaCancelada {
			t.Errorf("parcela %d = %s", parcela.Numero, parcela.Status)
		}
	}
	for _, conta := range contas {
		if status := c.statusDaConta(t, conta.SolicitacaoID); status != "cancelado" {
			t.Errorf("conta %s = %s", conta.SolicitacaoID, status)
		}
	}
	var eventos int
	if err := c.banco.Get(&eventos, `
		SELECT count(*) FROM solicitacao_historico WHERE evento = 'cancelada' AND solicitacao_id = ANY($1::uuid[])
	`, "{"+contas[0].SolicitacaoID.String()+","+contas[1].SolicitacaoID.String()+"}"); err != nil || eventos != 2 {
		t.Errorf("histórico das contas = %d, %v", eventos, err)
	}
}

func TestDividaCancelamentoRecusaContaPagaDepoisDaLeitura(t *testing.T) {
	c := novoCenarioDividas(t)
	divida, geracao := c.nova(t, c.mia)
	lida := c.obter(t, divida.ID)
	agora := time.Now().UTC()

	parcelas, contas, err := lida.Cancelar("erro", agora)
	if err != nil {
		t.Fatalf("Cancelar: %v", err)
	}
	if _, err := c.banco.Exec(`UPDATE solicitacao SET status = 'aprovado' WHERE id = $1`, geracao.Contas[0].Solicitacao.ID); err != nil {
		t.Fatalf("aprovação concorrente falhou: %v", err)
	}

	gravacao := &portsdivida.Gravacao{Divida: lida, VersaoLida: 1, StatusAnterior: domaindivida.StatusAtiva, ParcelasAlteradas: parcelas}
	for _, conta := range contas {
		anterior := conta.StatusAnterior
		gravacao.ContasCanceladas = append(gravacao.ContasCanceladas, portsdivida.ContaCancelada{
			ContaCancelada: conta,
			Historico: domainsolicitacao.NovoHistorico(conta.SolicitacaoID, c.usuarioID, &anterior, domainsolicitacao.StatusCancelado,
				domainsolicitacao.EventoCancelada, "dívida cancelada"),
		})
	}
	if err := c.dividas.Gravar(c.ctx, gravacao); !errors.Is(err, domain.ErrConflito) {
		t.Fatalf("conta alterada depois da leitura: %v", err)
	}

	depois := c.obter(t, divida.ID)
	if depois.Status != domaindivida.StatusAtiva || depois.Versao != 1 || c.statusDaConta(t, geracao.Contas[1].Solicitacao.ID) != "pendente_aprovacao" {
		t.Errorf("cancelamento parcial: %+v", depois)
	}
}

func TestDividaComParcelasAGerarPorLimiteEEmpresa(t *testing.T) {
	c := novoCenarioDividas(t)
	mia, _ := c.nova(t, c.mia)
	red, _ := c.nova(t, c.red)

	ids, err := c.dividas.ComParcelasAGerar(c.ctx, domaindivida.FimDoMes(c.hoje), nil)
	if err != nil || len(ids) != 0 {
		t.Fatalf("mês corrente já gerado = %v, %v", ids, err)
	}
	limite := domaindivida.FimDoMes(c.hoje.AddDate(0, 2, 0))
	ids, err = c.dividas.ComParcelasAGerar(c.ctx, limite, nil)
	if err != nil || len(ids) != 2 {
		t.Fatalf("todas as empresas = %v, %v", ids, err)
	}
	ids, err = c.dividas.ComParcelasAGerar(c.ctx, limite, []uuid.UUID{c.red})
	if err != nil || len(ids) != 1 || ids[0] != red.ID {
		t.Fatalf("só a Red = %v, %v", ids, err)
	}
	ids, err = c.dividas.ComParcelasAGerar(c.ctx, limite, []uuid.UUID{})
	if err != nil || len(ids) != 0 {
		t.Fatalf("sem empresas = %v, %v", ids, err)
	}

	lista, err := c.dividas.Listar(c.ctx, portsdivida.DividaFiltro{Empresa: &c.mia})
	if err != nil || len(lista) != 1 || lista[0].ID != mia.ID {
		t.Fatalf("lista da MIA = %+v, %v", lista, err)
	}
	lista, err = c.dividas.Listar(c.ctx, portsdivida.DividaFiltro{CredorTipo: domaindivida.CredorFisco})
	if err != nil || len(lista) != 0 {
		t.Fatalf("lista do fisco = %+v, %v", lista, err)
	}
}

func TestDividaRecusaParcelaInvalidaNoBanco(t *testing.T) {
	c := novoCenarioDividas(t)
	divida, _ := c.nova(t, c.mia)

	_, err := c.banco.Exec(`
		INSERT INTO divida_parcela (divida_id, numero, vencimento, amortizacao_centavos, juros_centavos, status)
		VALUES ($1, 9, $2, 0, 0, 'prevista')
	`, divida.ID, c.hoje.AddDate(1, 0, 0))
	if err == nil {
		t.Fatal("parcela zerada aceita")
	}
	_, err = c.banco.Exec(`
		INSERT INTO divida_parcela (divida_id, numero, vencimento, amortizacao_centavos, juros_centavos, status)
		VALUES ($1, 1, $2, 10, 0, 'prevista')
	`, divida.ID, c.hoje.AddDate(1, 0, 0))
	if err == nil {
		t.Fatal("número de parcela repetido aceito")
	}
}

func canceladasDaDivida(usuarioID uuid.UUID, contas []domaindivida.ContaCancelada, descricao string) []portsdivida.ContaCancelada {
	canceladas := make([]portsdivida.ContaCancelada, 0, len(contas))
	for _, conta := range contas {
		anterior := conta.StatusAnterior
		canceladas = append(canceladas, portsdivida.ContaCancelada{
			ContaCancelada: conta,
			Historico: domainsolicitacao.NovoHistorico(conta.SolicitacaoID, usuarioID, &anterior, domainsolicitacao.StatusCancelado,
				domainsolicitacao.EventoCancelada, descricao),
		})
	}

	return canceladas
}

func TestDividaRenegociadaGravaASubstitutaNaMesmaTransacao(t *testing.T) {
	c := novoCenarioDividas(t)
	divida, geracao := c.nova(t, c.mia)
	if _, err := c.banco.Exec(`UPDATE solicitacao SET status = 'pago' WHERE id = $1`, geracao.Contas[0].Solicitacao.ID); err != nil {
		t.Fatalf("pagamento de teste falhou: %v", err)
	}
	lida := c.obter(t, divida.ID)
	agora := time.Now().UTC()

	renegociacao, err := lida.Renegociar("acordo", domaindivida.DadosDivida{
		DadosInformativos:    lida.DadosInformativosAtuais(),
		FornecedorID:         c.credorID,
		SaldoInicialCentavos: 210000,
		DataSaldo:            c.hoje.AddDate(0, 0, -1),
		Parcelas: []domaindivida.ParcelaInput{
			{Vencimento: c.hoje, AmortizacaoCentavos: 105000, JurosCentavos: 1000},
			{Vencimento: c.hoje.AddDate(0, 1, 1), AmortizacaoCentavos: 105000, JurosCentavos: 500},
		},
	}, c.usuarioID, agora)
	if err != nil {
		t.Fatalf("Renegociar: %v", err)
	}
	substituta := renegociacao.Substituta
	contasNovas, err := substituta.GerarContasAte(domaindivida.FimDoMes(c.hoje), c.usuarioID, agora)
	if err != nil {
		t.Fatalf("geração da substituta: %v", err)
	}
	if err := c.dividas.Gravar(c.ctx, &portsdivida.Gravacao{
		Divida: lida, VersaoLida: 1, StatusAnterior: domaindivida.StatusAtiva,
		ParcelasAlteradas: renegociacao.Parcelas, ConferirContas: true,
		ContasCanceladas: canceladasDaDivida(c.usuarioID, renegociacao.Contas, "dívida renegociada"),
		Substituta:       substituta,
		ContasAPagar:     contasNovas.Contas,
		Historicos: []*domaindivida.Historico{
			domaindivida.NovoHistorico(lida.ID, c.usuarioID, domaindivida.AcaoRenegociada, nil, lida.Status, renegociacao.DetalheOriginal, agora),
			domaindivida.NovoHistorico(substituta.ID, c.usuarioID, domaindivida.AcaoCriada, nil, substituta.Status, renegociacao.DetalheSubstituta, agora),
		},
	}); err != nil {
		t.Fatalf("gravação da renegociação: %v", err)
	}

	original := c.obter(t, divida.ID)
	if original.Status != domaindivida.StatusRenegociada || original.Versao != 2 {
		t.Fatalf("original = %+v", original)
	}
	if original.Parcelas[0].Status != domaindivida.StatusParcelaGerada || original.Parcelas[0].ContaAmortizacao.Status != domainsolicitacao.StatusPago ||
		original.Parcelas[0].ContaJuros.Status != domainsolicitacao.StatusCancelado {
		t.Errorf("parcela parcialmente paga = %+v", original.Parcelas[0])
	}
	if original.Parcelas[1].Status != domaindivida.StatusParcelaCancelada || original.Parcelas[2].Status != domaindivida.StatusParcelaCancelada {
		t.Errorf("parcelas futuras da original: %s %s", original.Parcelas[1].Status, original.Parcelas[2].Status)
	}

	nova := c.obter(t, substituta.ID)
	if nova.DividaOrigemID == nil || *nova.DividaOrigemID != divida.ID || nova.SaldoInicialCentavos != 210000 ||
		len(nova.Parcelas) != 2 || nova.Parcelas[0].ContaAmortizacao == nil {
		t.Fatalf("substituta = %+v", nova)
	}

	segunda, err := lida.Renegociar("outra", domaindivida.DadosDivida{}, c.usuarioID, agora)
	if segunda != nil || !errors.Is(err, domain.ErrConflito) {
		t.Fatalf("segunda renegociação = %v", err)
	}
}

func TestDividaReabreParcelaComOMesmoNumero(t *testing.T) {
	c := novoCenarioDividas(t)
	divida, geracao := c.nova(t, c.mia)
	for _, conta := range geracao.Contas {
		if _, err := c.banco.Exec(`UPDATE solicitacao SET status = 'rejeitado' WHERE id = $1`, conta.Solicitacao.ID); err != nil {
			t.Fatalf("rejeição de teste falhou: %v", err)
		}
	}
	lida := c.obter(t, divida.ID)
	agora := time.Now().UTC()

	antiga, nova, err := lida.ReabrirParcela(lida.Parcelas[0].ID, agora)
	if err != nil {
		t.Fatalf("ReabrirParcela: %v", err)
	}
	if err := c.dividas.Gravar(c.ctx, &portsdivida.Gravacao{
		Divida: lida, VersaoLida: 1, StatusAnterior: domaindivida.StatusAtiva, ConferirContas: true,
		ParcelasAlteradas: []*domaindivida.Parcela{antiga}, ParcelasNovas: []*domaindivida.Parcela{nova},
	}); err != nil {
		t.Fatalf("gravação da reabertura: %v", err)
	}

	depois := c.obter(t, divida.ID)
	if len(depois.Parcelas) != 4 || depois.Parcelas[0].ID != antiga.ID || depois.Parcelas[1].ID != nova.ID ||
		depois.Parcelas[0].SubstitutaID == nil || *depois.Parcelas[0].SubstitutaID != nova.ID ||
		depois.Parcelas[1].Status != domaindivida.StatusParcelaPrevista || depois.Parcelas[1].ContaJuros != nil {
		t.Fatalf("parcelas depois de reabrir = %+v", depois.Parcelas)
	}

	geracaoNova, err := depois.GerarContasAte(domaindivida.FimDoMes(c.hoje), c.usuarioID, agora)
	if err != nil || len(geracaoNova.Contas) != 2 {
		t.Fatalf("geração da reaberta = %+v, %v", geracaoNova, err)
	}
	if err := c.dividas.Gravar(c.ctx, &portsdivida.Gravacao{
		Divida: depois, VersaoLida: 2, StatusAnterior: domaindivida.StatusAtiva,
		ParcelasAlteradas: geracaoNova.Parcelas, ContasAPagar: geracaoNova.Contas,
	}); err != nil {
		t.Fatalf("gravação da geração da reaberta: %v", err)
	}
	if final := c.obter(t, divida.ID); final.Parcelas[1].ContaJuros == nil || final.Parcelas[1].ContaJuros.Status != domainsolicitacao.StatusPendenteAprovacao {
		t.Errorf("reaberta gerada = %+v", final.Parcelas[1])
	}
}
