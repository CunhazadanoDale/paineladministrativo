package divida_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaindivida "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/divida"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/divida"
	"github.com/google/uuid"
)

type cenario struct {
	memoria     *memoria
	empresa     uuid.UUID
	financeiro  uuid.UUID
	credor      uuid.UUID
	amortizacao uuid.UUID
	juros       uuid.UUID
	hoje        time.Time
}

func novoCenario() *cenario {
	m := novaMemoria()
	empresa := m.empresa(true)

	return &cenario{
		memoria:     m,
		empresa:     empresa,
		financeiro:  m.usuario(true, empresa),
		credor:      m.credor(true),
		amortizacao: m.categoria(domainfinanceiro.LinhaForaDRE),
		juros:       m.categoria(domainfinanceiro.LinhaResultadoFinanceiro),
		hoje:        domaindivida.Hoje(time.Now()),
	}
}

func (c *cenario) entrada() portsin.CriarDividaInput {
	return portsin.CriarDividaInput{
		UsuarioID: c.financeiro,
		EmpresaID: c.empresa,
		NovaDividaInput: portsin.NovaDividaInput{
			FornecedorID: c.credor,
			Dados: portsin.DadosInformativosInput{
				CredorTipo:             "banco",
				Descricao:              "Capital de giro",
				CategoriaAmortizacaoID: c.amortizacao,
				CategoriaJurosID:       c.juros,
			},
			SaldoInicialCentavos: 200000,
			DataSaldo:            c.hoje.AddDate(0, 0, -1),
			Parcelas: []portsin.ParcelaInput{
				{Vencimento: c.hoje, AmortizacaoCentavos: 100000, JurosCentavos: 2000},
				{Vencimento: c.hoje.AddDate(0, 2, 0), AmortizacaoCentavos: 100000, JurosCentavos: 1000},
			},
		},
	}
}

func (c *cenario) criar(t *testing.T) *portsin.Visao {
	t.Helper()

	visao, err := c.memoria.usecase().Criar(context.Background(), c.entrada())
	if err != nil {
		t.Fatalf("Criar falhou: %v", err)
	}

	return visao
}

func TestCriarGeraAsContasDoMesCorrente(t *testing.T) {
	c := novoCenario()
	visao := c.criar(t)
	divida := visao.Divida

	if divida.Versao != 1 || divida.Status != domaindivida.StatusAtiva || len(divida.Parcelas) != 2 {
		t.Fatalf("dívida = %+v", divida)
	}
	primeira, segunda := divida.Parcelas[0], divida.Parcelas[1]
	if primeira.Status != domaindivida.StatusParcelaGerada || primeira.ContaAmortizacao == nil || primeira.ContaJuros == nil {
		t.Fatalf("parcela do mês = %+v", primeira)
	}
	if segunda.Status != domaindivida.StatusParcelaPrevista || segunda.ContaJuros != nil {
		t.Fatalf("parcela futura = %+v", segunda)
	}
	if len(c.memoria.solicitacoes) != 2 {
		t.Fatalf("contas a pagar = %d", len(c.memoria.solicitacoes))
	}
	juros := c.memoria.solicitacoes[primeira.ContaJuros.SolicitacaoID]
	if juros.Origem != domainsolicitacao.OrigemDividaJuros || *juros.CategoriaFinanceiraID != c.juros || *juros.FornecedorID != c.credor ||
		juros.SolicitanteID != c.financeiro || juros.Valor.Centavos() != 2000 {
		t.Errorf("conta de juros = %+v", juros)
	}
	if len(c.memoria.historicos) != 2 || c.memoria.historicos[0].Acao != domaindivida.AcaoCriada ||
		c.memoria.historicos[1].Acao != domaindivida.AcaoContasGeradas {
		t.Errorf("histórico = %+v", c.memoria.historicos)
	}
}

func TestCriarValidaPerfilEmpresaCredorECategorias(t *testing.T) {
	casos := map[string]struct {
		alterar func(*cenario, *portsin.CriarDividaInput)
		erro    error
	}{
		"sem perfil financeiro": {func(c *cenario, e *portsin.CriarDividaInput) { e.UsuarioID = c.memoria.usuario(false, c.empresa) }, domain.ErrPermissao},
		"sem acesso à empresa":  {func(c *cenario, e *portsin.CriarDividaInput) { e.EmpresaID = c.memoria.empresa(true) }, domain.ErrPermissao},
		"empresa inativa": {func(c *cenario, e *portsin.CriarDividaInput) {
			e.EmpresaID = c.memoria.empresa(false)
			e.UsuarioID = c.memoria.usuario(true, e.EmpresaID)
		}, domain.ErrValidacao},
		"credor inativo":          {func(c *cenario, e *portsin.CriarDividaInput) { e.FornecedorID = c.memoria.credor(false) }, domain.ErrValidacao},
		"credor inexistente":      {func(_ *cenario, e *portsin.CriarDividaInput) { e.FornecedorID = uuid.New() }, domain.ErrValidacao},
		"tipo de credor inválido": {func(_ *cenario, e *portsin.CriarDividaInput) { e.Dados.CredorTipo = "agiota" }, domain.ErrValidacao},
		"juros fora do resultado financeiro": {func(c *cenario, e *portsin.CriarDividaInput) {
			e.Dados.CategoriaJurosID = c.memoria.categoria(domainfinanceiro.LinhaDespesaFixa)
		}, domain.ErrValidacao},
		"amortização dentro da DRE": {func(c *cenario, e *portsin.CriarDividaInput) {
			e.Dados.CategoriaAmortizacaoID = c.juros
		}, domain.ErrValidacao},
		"conta de outra empresa": {func(c *cenario, e *portsin.CriarDividaInput) {
			conta := &domainfinanceiro.ContaBancaria{ID: uuid.New(), EmpresaID: uuid.New(), Ativa: true}
			c.memoria.contas[conta.ID] = conta
			liberada := c.hoje
			e.Dados.ContaLiberacaoID, e.Dados.LiberadaEm = &conta.ID, &liberada
		}, domain.ErrValidacao},
		"taxa sem período": {func(_ *cenario, e *portsin.CriarDividaInput) { bp := 185; e.Dados.TaxaBP = &bp }, domain.ErrValidacao},
	}

	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			c := novoCenario()
			entrada := c.entrada()
			caso.alterar(c, &entrada)
			if _, err := c.memoria.usecase().Criar(context.Background(), entrada); !errors.Is(err, caso.erro) {
				t.Fatalf("erro = %v, esperado %v", err, caso.erro)
			}
			if c.memoria.gravacoes != 0 {
				t.Fatalf("gravou com erro")
			}
		})
	}
}

func TestObterRespeitaEscopoEPerfil(t *testing.T) {
	c := novoCenario()
	visao := c.criar(t)
	usecase := c.memoria.usecase()

	if _, err := usecase.Obter(context.Background(), visao.Divida.ID, c.memoria.usuario(true, c.memoria.empresa(true))); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("outra empresa: %v", err)
	}
	if _, err := usecase.Obter(context.Background(), visao.Divida.ID, c.memoria.usuario(false, c.empresa)); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("sem financeiro: %v", err)
	}
	if _, err := usecase.Obter(context.Background(), uuid.New(), c.financeiro); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("inexistente: %v", err)
	}
	lida, err := usecase.Obter(context.Background(), visao.Divida.ID, c.financeiro)
	if err != nil || lida.Divida.SaldoDevedor() != 200000 || !lida.Hoje.Equal(c.hoje) {
		t.Errorf("leitura = %+v, %v", lida, err)
	}

	outra := c.memoria.usuario(true, c.memoria.empresa(true))
	lista, err := usecase.Listar(context.Background(), portsin.ListarDividasInput{UsuarioID: outra})
	if err != nil || len(lista) != 0 {
		t.Errorf("lista de outra empresa = %d, %v", len(lista), err)
	}
	lista, err = usecase.Listar(context.Background(), portsin.ListarDividasInput{UsuarioID: c.financeiro, CredorTipo: "banco"})
	if err != nil || len(lista) != 1 {
		t.Errorf("lista = %d, %v", len(lista), err)
	}
	if _, err := usecase.Listar(context.Background(), portsin.ListarDividasInput{UsuarioID: c.financeiro, Status: "quitada"}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("status inválido: %v", err)
	}
}

func TestAlterarConfereVersaoEMantemCronograma(t *testing.T) {
	c := novoCenario()
	visao := c.criar(t)
	usecase := c.memoria.usecase()
	dados := portsin.DadosInformativosInput{
		CredorTipo: "investidor", Descricao: "Aporte do sócio", CategoriaAmortizacaoID: c.amortizacao, CategoriaJurosID: c.juros,
	}

	if _, err := usecase.Alterar(context.Background(), portsin.AlterarDividaInput{
		UsuarioID: c.financeiro, DividaID: visao.Divida.ID, Versao: 2, Dados: dados,
	}); !errors.Is(err, domain.ErrConflito) {
		t.Fatalf("versão divergente: %v", err)
	}
	alterada, err := usecase.Alterar(context.Background(), portsin.AlterarDividaInput{
		UsuarioID: c.financeiro, DividaID: visao.Divida.ID, Versao: 1, Dados: dados,
	})
	if err != nil || alterada.Divida.Versao != 2 || alterada.Divida.CredorTipo != domaindivida.CredorInvestidor ||
		alterada.Divida.SaldoInicialCentavos != 200000 || len(alterada.Divida.Parcelas) != 2 {
		t.Fatalf("alterada = %+v, %v", alterada, err)
	}
}

func TestAlterarParcelaPrevista(t *testing.T) {
	c := novoCenario()
	visao := c.criar(t)
	segunda := visao.Divida.Parcelas[1]

	alterada, err := c.memoria.usecase().AlterarParcela(context.Background(), portsin.AlterarParcelaInput{
		UsuarioID: c.financeiro, DividaID: visao.Divida.ID, ParcelaID: segunda.ID, Versao: 1,
		Vencimento: segunda.Vencimento.AddDate(0, 0, 5), JurosCentavos: 1500,
	})
	if err != nil || alterada.Divida.Parcelas[1].JurosCentavos != 1500 {
		t.Fatalf("parcela alterada = %+v, %v", alterada, err)
	}
	ultimo := c.memoria.historicos[len(c.memoria.historicos)-1]
	if ultimo.Acao != domaindivida.AcaoParcelaAlterada || ultimo.StatusAnterior != nil {
		t.Errorf("histórico = %+v", ultimo)
	}

	if _, err := c.memoria.usecase().AlterarParcela(context.Background(), portsin.AlterarParcelaInput{
		UsuarioID: c.financeiro, DividaID: visao.Divida.ID, ParcelaID: visao.Divida.Parcelas[0].ID, Versao: 2,
		Vencimento: c.hoje, JurosCentavos: 1,
	}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("parcela gerada: %v", err)
	}
}

func TestCancelarCancelaAsContasAbertas(t *testing.T) {
	c := novoCenario()
	visao := c.criar(t)
	primeira := visao.Divida.Parcelas[0]
	c.memoria.solicitacoes[primeira.ContaJuros.SolicitacaoID].Status = domainsolicitacao.StatusAprovado

	cancelada, err := c.memoria.usecase().Cancelar(context.Background(), portsin.CancelarDividaInput{
		UsuarioID: c.financeiro, DividaID: visao.Divida.ID, Versao: 1, Motivo: "lançada em duplicidade",
	})
	if err != nil {
		t.Fatalf("Cancelar falhou: %v", err)
	}
	if cancelada.Divida.Status != domaindivida.StatusCancelada || cancelada.Divida.Versao != 2 {
		t.Fatalf("dívida = %+v", cancelada.Divida)
	}
	if len(c.memoria.canceladas) != 2 {
		t.Fatalf("contas canceladas = %+v", c.memoria.canceladas)
	}
	for _, conta := range c.memoria.canceladas {
		if conta.Historico.Evento != domainsolicitacao.EventoCancelada || conta.Historico.SolicitacaoID != conta.SolicitacaoID ||
			*conta.Historico.DeStatus != conta.StatusAnterior || c.memoria.solicitacoes[conta.SolicitacaoID].Status != domainsolicitacao.StatusCancelado {
			t.Errorf("conta cancelada = %+v", conta)
		}
	}
	ultimo := c.memoria.historicos[len(c.memoria.historicos)-1]
	if ultimo.Acao != domaindivida.AcaoCancelada || ultimo.StatusAnterior == nil || *ultimo.StatusAnterior != domaindivida.StatusAtiva {
		t.Errorf("histórico = %+v", ultimo)
	}
}

func TestCancelarRecusaContaPaga(t *testing.T) {
	c := novoCenario()
	visao := c.criar(t)
	c.memoria.solicitacoes[visao.Divida.Parcelas[0].ContaAmortizacao.SolicitacaoID].Status = domainsolicitacao.StatusPago

	if _, err := c.memoria.usecase().Cancelar(context.Background(), portsin.CancelarDividaInput{
		UsuarioID: c.financeiro, DividaID: visao.Divida.ID, Versao: 1, Motivo: "erro",
	}); !errors.Is(err, domain.ErrConflito) {
		t.Fatalf("erro = %v", err)
	}
}

func TestGerarContasDaCompetencia(t *testing.T) {
	c := novoCenario()
	visao := c.criar(t)
	outraEmpresa := c.memoria.empresa(true)
	usecase := c.memoria.usecase()
	competencia := visao.Divida.Parcelas[1].Vencimento.Format("2006-01")

	if _, err := usecase.GerarContas(context.Background(), portsin.GerarContasInput{UsuarioID: c.financeiro}); !errors.Is(err, domain.ErrValidacao) {
		t.Fatalf("sem competência: %v", err)
	}
	if _, err := usecase.GerarContas(context.Background(), portsin.GerarContasInput{
		UsuarioID: c.financeiro, Competencia: competencia, Empresa: &outraEmpresa,
	}); !errors.Is(err, domain.ErrPermissao) {
		t.Fatalf("empresa fora do acesso: %v", err)
	}

	resultado, err := usecase.GerarContas(context.Background(), portsin.GerarContasInput{UsuarioID: c.financeiro, Competencia: competencia})
	if err != nil || len(resultado.Geradas) != 1 || resultado.Geradas[0].Contas != 2 || resultado.Geradas[0].Parcelas != 1 ||
		resultado.Competencia != competencia {
		t.Fatalf("resultado = %+v, %v", resultado, err)
	}
	if len(c.memoria.solicitacoes) != 4 || c.memoria.dividas[visao.Divida.ID].Versao != 2 {
		t.Fatalf("contas = %d, versão = %d", len(c.memoria.solicitacoes), c.memoria.dividas[visao.Divida.ID].Versao)
	}

	deNovo, err := usecase.GerarContas(context.Background(), portsin.GerarContasInput{UsuarioID: c.financeiro, Competencia: competencia})
	if err != nil || len(deNovo.Geradas) != 0 || len(deNovo.Recusadas) != 0 {
		t.Fatalf("segunda geração = %+v, %v", deNovo, err)
	}
}

func TestGerarContasRecusaCredorInativo(t *testing.T) {
	c := novoCenario()
	visao := c.criar(t)
	c.memoria.fornecedores[c.credor].Ativo = false
	competencia := visao.Divida.Parcelas[1].Vencimento.Format("2006-01")

	resultado, err := c.memoria.usecase().GerarContas(context.Background(), portsin.GerarContasInput{
		UsuarioID: c.financeiro, Competencia: competencia,
	})
	if err != nil {
		t.Fatalf("GerarContas falhou: %v", err)
	}
	if len(resultado.Geradas) != 0 || len(resultado.Recusadas) != 1 || resultado.Recusadas[0].DividaID != visao.Divida.ID ||
		resultado.Recusadas[0].Motivo == "" {
		t.Fatalf("resultado = %+v", resultado)
	}
	if c.memoria.dividas[visao.Divida.ID].Parcelas[1].Status != domaindivida.StatusParcelaPrevista {
		t.Errorf("parcela recusada ficou gerada")
	}
}

func (c *cenario) novaDaRenegociacao() portsin.NovaDividaInput {
	return portsin.NovaDividaInput{
		FornecedorID: c.memoria.credor(true),
		Dados: portsin.DadosInformativosInput{
			CredorTipo:             "banco",
			Descricao:              "Renegociação",
			CategoriaAmortizacaoID: c.amortizacao,
			CategoriaJurosID:       c.juros,
		},
		SaldoInicialCentavos: 150000,
		DataSaldo:            c.hoje.AddDate(0, 0, -1),
		Parcelas: []portsin.ParcelaInput{
			{Vencimento: c.hoje, AmortizacaoCentavos: 50000, JurosCentavos: 900},
			{Vencimento: c.hoje.AddDate(0, 3, 0), AmortizacaoCentavos: 100000, JurosCentavos: 600},
		},
	}
}

func TestRenegociarCriaASubstitutaECancelaAsContasAbertas(t *testing.T) {
	c := novoCenario()
	visao := c.criar(t)
	original := visao.Divida
	c.memoria.solicitacoes[original.Parcelas[0].ContaJuros.SolicitacaoID].Status = domainsolicitacao.StatusPago
	nova := c.novaDaRenegociacao()

	if _, err := c.memoria.usecase().Renegociar(context.Background(), portsin.RenegociarDividaInput{
		UsuarioID: c.financeiro, DividaID: original.ID, Versao: 2, Motivo: "acordo", Nova: nova,
	}); !errors.Is(err, domain.ErrConflito) {
		t.Fatalf("versão divergente: %v", err)
	}
	inativo := nova
	inativo.FornecedorID = c.memoria.credor(false)
	if _, err := c.memoria.usecase().Renegociar(context.Background(), portsin.RenegociarDividaInput{
		UsuarioID: c.financeiro, DividaID: original.ID, Versao: 1, Motivo: "acordo", Nova: inativo,
	}); !errors.Is(err, domain.ErrValidacao) {
		t.Fatalf("credor inativo: %v", err)
	}

	substituta, err := c.memoria.usecase().Renegociar(context.Background(), portsin.RenegociarDividaInput{
		UsuarioID: c.financeiro, DividaID: original.ID, Versao: 1, Motivo: "acordo com o banco", Nova: nova,
	})
	if err != nil {
		t.Fatalf("Renegociar falhou: %v", err)
	}
	if substituta.Divida.ID == original.ID || substituta.Divida.DividaOrigemID == nil || *substituta.Divida.DividaOrigemID != original.ID ||
		substituta.Divida.FornecedorID != nova.FornecedorID || substituta.Divida.EmpresaID != c.empresa {
		t.Fatalf("substituta = %+v", substituta.Divida)
	}
	if substituta.Divida.Parcelas[0].Status != domaindivida.StatusParcelaGerada {
		t.Errorf("primeira parcela da substituta não foi gerada no mês")
	}

	gravada := c.memoria.dividas[original.ID]
	if gravada.Status != domaindivida.StatusRenegociada || gravada.Versao != 2 {
		t.Fatalf("original = %+v", gravada)
	}
	if len(c.memoria.canceladas) != 1 || c.memoria.canceladas[0].SolicitacaoID != original.Parcelas[0].ContaAmortizacao.SolicitacaoID {
		t.Fatalf("canceladas = %+v", c.memoria.canceladas)
	}
	if gravada.Parcelas[0].Status != domaindivida.StatusParcelaGerada || gravada.Parcelas[1].Status != domaindivida.StatusParcelaCancelada {
		t.Errorf("parcelas da original: %s %s", gravada.Parcelas[0].Status, gravada.Parcelas[1].Status)
	}

	acoes := map[uuid.UUID][]domaindivida.Acao{}
	for _, historico := range c.memoria.historicos {
		acoes[historico.DividaID] = append(acoes[historico.DividaID], historico.Acao)
	}
	if ultima := acoes[original.ID][len(acoes[original.ID])-1]; ultima != domaindivida.AcaoRenegociada {
		t.Errorf("histórico da original = %v", acoes[original.ID])
	}
	if doNovo := acoes[substituta.Divida.ID]; len(doNovo) != 2 || doNovo[0] != domaindivida.AcaoCriada || doNovo[1] != domaindivida.AcaoContasGeradas {
		t.Errorf("histórico da substituta = %v", doNovo)
	}

	if _, err := c.memoria.usecase().Renegociar(context.Background(), portsin.RenegociarDividaInput{
		UsuarioID: c.financeiro, DividaID: original.ID, Versao: 2, Motivo: "de novo", Nova: c.novaDaRenegociacao(),
	}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("renegociar a renegociada: %v", err)
	}
}

func TestReabrirParcelaDepoisDeCancelarAsContasNoM7(t *testing.T) {
	c := novoCenario()
	visao := c.criar(t)
	primeira := visao.Divida.Parcelas[0]
	entrada := portsin.ReabrirParcelaInput{UsuarioID: c.financeiro, DividaID: visao.Divida.ID, ParcelaID: primeira.ID, Versao: 1}

	if _, err := c.memoria.usecase().ReabrirParcela(context.Background(), entrada); !errors.Is(err, domain.ErrConflito) {
		t.Fatalf("reabrir com contas abertas: %v", err)
	}
	c.memoria.solicitacoes[primeira.ContaAmortizacao.SolicitacaoID].Status = domainsolicitacao.StatusCancelado
	c.memoria.solicitacoes[primeira.ContaJuros.SolicitacaoID].Status = domainsolicitacao.StatusRejeitado

	reaberta, err := c.memoria.usecase().ReabrirParcela(context.Background(), entrada)
	if err != nil {
		t.Fatalf("ReabrirParcela falhou: %v", err)
	}
	parcelas := reaberta.Divida.Parcelas
	if len(parcelas) != 3 || parcelas[0].Status != domaindivida.StatusParcelaCancelada || parcelas[1].Status != domaindivida.StatusParcelaPrevista ||
		parcelas[1].Numero != 1 || *parcelas[0].SubstitutaID != parcelas[1].ID || reaberta.Divida.Versao != 2 {
		t.Fatalf("parcelas depois de reabrir = %+v", parcelas)
	}
	if ultimo := c.memoria.historicos[len(c.memoria.historicos)-1]; ultimo.Acao != domaindivida.AcaoParcelaReaberta {
		t.Errorf("histórico = %+v", ultimo)
	}
}
