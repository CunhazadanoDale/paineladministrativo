package comissao_test

import (
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/comissao"
	"github.com/google/uuid"
)

type cenarioFechamento struct {
	*cenario
	despesa     uuid.UUID
	receita     uuid.UUID
	fornecedor  uuid.UUID
	competencia domaincomissao.Competencia
}

func novoCenarioFechamento(t *testing.T) *cenarioFechamento {
	t.Helper()

	c := &cenarioFechamento{cenario: novoCenario(t), fornecedor: uuid.New()}
	raiz := uuid.New()
	c.despesa = c.novaCategoria(raiz, domainfinanceiro.NaturezaDespesa)
	c.receita = c.novaCategoria(raiz, domainfinanceiro.NaturezaReceita)
	c.carlos.FornecedorID = &c.fornecedor
	c.competencia = domaincomissao.CompetenciaDe(domaincomissao.CompetenciaDe(domaincomissao.Hoje(time.Now().UTC())).Inicio().AddDate(0, -1, 0))

	return c
}

func (c *cenarioFechamento) novaCategoria(paiID uuid.UUID, natureza domainfinanceiro.Natureza) uuid.UUID {
	categoria := &domainfinanceiro.CategoriaFinanceira{ID: uuid.New(), CategoriaPaiID: &paiID, Natureza: natureza, Ativa: true}
	c.m.categorias[categoria.ID] = categoria

	return categoria.ID
}

func (c *cenarioFechamento) input(usuarioID uuid.UUID) portsin.FecharInput {
	return portsin.FecharInput{
		UsuarioID: usuarioID, EmpresaID: c.mia, Competencia: c.competencia.String(), CategoriaFinanceiraID: c.despesa,
	}
}

func TestFecharGeraUmFechamentoComContaAPagarPorRepresentante(t *testing.T) {
	c := novoCenarioFechamento(t)
	c.m.saldos = []domaincomissao.SaldoAberto{
		{RepresentanteID: c.carlos.ID, ComissoesCentavos: 10000, AjustesCentavos: -1500},
	}

	resultado, err := c.fechamentos.Fechar(c.ctx, c.input(c.financeiro))
	if err != nil {
		t.Fatalf("Fechar: %v", err)
	}
	if len(resultado.Fechados) != 1 || len(resultado.Falhas) != 0 || resultado.Competencia != c.competencia {
		t.Fatalf("resultado = %+v", resultado)
	}

	gravacao := c.m.fechamentos[0]
	fechamento, conta := gravacao.Fechamento, gravacao.Solicitacao
	if fechamento.TotalCentavos != 8500 || fechamento.RepresentanteID != c.carlos.ID || fechamento.SolicitacaoID != conta.ID {
		t.Fatalf("fechamento = %+v", fechamento)
	}
	if !fechamento.Vencimento.Equal(c.competencia.Seguinte().Inicio().AddDate(0, 0, 14)) {
		t.Fatalf("vencimento = %v", fechamento.Vencimento)
	}
	if conta.Origem != domainsolicitacao.OrigemComissao || *conta.OrigemID != fechamento.ID || *conta.FornecedorID != c.fornecedor ||
		*conta.CategoriaFinanceiraID != c.despesa || conta.Valor.Centavos() != 8500 || conta.Status != domainsolicitacao.StatusPendenteAprovacao ||
		conta.Competencia.Inicio() != c.competencia.Inicio() || conta.SolicitanteID != c.financeiro {
		t.Fatalf("conta a pagar = %+v", conta)
	}
	if gravacao.Historico == nil || gravacao.Historico.SolicitacaoID != conta.ID {
		t.Fatalf("histórico = %+v", gravacao.Historico)
	}

	filtro := c.m.ultimoFiltroEntregas
	if filtro.Empresa == nil || *filtro.Empresa != c.mia || !filtro.SaidaAntesDe.Equal(c.competencia.Seguinte().Inicio()) {
		t.Fatalf("a apuração deveria rodar até a competência: %+v", filtro)
	}
	if c.m.ultimoFiltroSaldos.EmpresaID != c.mia || c.m.ultimoFiltroSaldos.Ate != c.competencia {
		t.Fatalf("filtro dos saldos = %+v", c.m.ultimoFiltroSaldos)
	}
}

func TestFecharApuraAsSaidasPendentesAntes(t *testing.T) {
	c := novoCenarioFechamento(t)
	c.novaEntrega(c.pedidoDoCarlos, c.competencia.Inicio().AddDate(0, 0, 3), domaincomissao.ItemSaido{Quantidade: 1, PrecoUnitarioCentavos: 10000})
	c.m.saldos = []domaincomissao.SaldoAberto{{RepresentanteID: c.carlos.ID, ComissoesCentavos: 250}}

	resultado, err := c.fechamentos.Fechar(c.ctx, c.input(c.administrador))
	if err != nil {
		t.Fatalf("Fechar: %v", err)
	}
	if len(resultado.Apuracao.Apuradas) != 1 || len(c.m.comissoes) != 1 || len(resultado.Fechados) != 1 {
		t.Fatalf("resultado = %+v", resultado)
	}
}

func TestFecharListaFalhasPorRepresentanteESegue(t *testing.T) {
	c := novoCenarioFechamento(t)
	semCadastro := uuid.New()
	c.m.saldos = []domaincomissao.SaldoAberto{
		{RepresentanteID: c.beatriz.ID, ComissoesCentavos: 1000},
		{RepresentanteID: c.carlos.ID, ComissoesCentavos: 500, AjustesCentavos: -500},
		{RepresentanteID: semCadastro, ComissoesCentavos: 100},
	}
	outroComFornecedor := c.novoRepresentante("Dora", nil, 100, 100)
	fornecedorDora := uuid.New()
	outroComFornecedor.FornecedorID = &fornecedorDora
	c.m.saldos = append(c.m.saldos, domaincomissao.SaldoAberto{RepresentanteID: outroComFornecedor.ID, ComissoesCentavos: 700})
	c.m.falhaNoFechar[outroComFornecedor.ID] = domaincomissao.ErroFechamentoRepetido()

	resultado, err := c.fechamentos.Fechar(c.ctx, c.input(c.financeiro))
	if err != nil {
		t.Fatalf("Fechar: %v", err)
	}
	if len(resultado.Fechados) != 0 || len(resultado.Falhas) != 4 {
		t.Fatalf("resultado = %+v", resultado)
	}
	motivos := map[uuid.UUID]string{}
	for _, falha := range resultado.Falhas {
		motivos[falha.RepresentanteID] = falha.Motivo
	}
	esperados := map[uuid.UUID]string{
		c.beatriz.ID:          "representante sem fornecedor",
		c.carlos.ID:           "nada a pagar",
		semCadastro:           "representante do fechamento não encontrado",
		outroComFornecedor.ID: "já existe fechamento",
	}
	for representanteID, trecho := range esperados {
		if !strings.Contains(motivos[representanteID], trecho) {
			t.Errorf("motivo de %s = %q, esperado %q", representanteID, motivos[representanteID], trecho)
		}
	}
}

func TestFecharDevolveErroDeInfraestrutura(t *testing.T) {
	c := novoCenarioFechamento(t)
	c.m.saldos = []domaincomissao.SaldoAberto{{RepresentanteID: c.carlos.ID, ComissoesCentavos: 1000}}
	c.m.falhaNoFechar[c.carlos.ID] = errBancoFora

	if _, err := c.fechamentos.Fechar(c.ctx, c.input(c.financeiro)); err != errBancoFora {
		t.Fatalf("erro = %v", err)
	}
}

func TestFecharRecusaPedidoInvalido(t *testing.T) {
	c := novoCenarioFechamento(t)
	c.m.empresas[c.mia].Ativa = false
	inativa := c.input(c.administrador)
	c2 := novoCenarioFechamento(t)
	mesCorrente := domaincomissao.CompetenciaDe(domaincomissao.Hoje(time.Now().UTC())).String()

	_, err := c.fechamentos.Fechar(c.ctx, inativa)
	esperarErro(t, err, domain.ErrValidacao, "empresa inativa")

	casos := []struct {
		nome      string
		ajustar   func(*portsin.FecharInput)
		sentinela error
		trecho    string
	}{
		{"comercial", func(i *portsin.FecharInput) { i.UsuarioID = c2.comercial }, domain.ErrPermissao, "apurar comissões"},
		{"outra empresa", func(i *portsin.FecharInput) { i.EmpresaID = c2.red }, domain.ErrPermissao, "sem acesso a esta empresa"},
		{"mês corrente", func(i *portsin.FecharInput) { i.Competencia = mesCorrente }, domain.ErrValidacao, "mês já encerrado"},
		{"competência inválida", func(i *portsin.FecharInput) { i.Competencia = "x" }, domain.ErrValidacao, "AAAA-MM"},
		{"categoria de receita", func(i *portsin.FecharInput) { i.CategoriaFinanceiraID = c2.receita }, domain.ErrValidacao, "de despesa"},
		{"categoria inexistente", func(i *portsin.FecharInput) { i.CategoriaFinanceiraID = uuid.New() }, domain.ErrValidacao, "não encontrada"},
		{"representante sem saldo", func(i *portsin.FecharInput) { i.RepresentanteID = &c2.beatriz.ID }, domain.ErrValidacao, "nada a pagar"},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			input := c2.input(c2.financeiro)
			caso.ajustar(&input)

			_, err := c2.fechamentos.Fechar(c2.ctx, input)
			esperarErro(t, err, caso.sentinela, caso.trecho)
		})
	}
}

func TestFechamentoVisivelSoAoRepresentanteDono(t *testing.T) {
	c := novoCenarioFechamento(t)
	c.m.saldos = []domaincomissao.SaldoAberto{{RepresentanteID: c.carlos.ID, ComissoesCentavos: 1000}}
	resultado, err := c.fechamentos.Fechar(c.ctx, c.input(c.financeiro))
	if err != nil || len(resultado.Fechados) != 1 {
		t.Fatalf("Fechar = %+v, %v", resultado, err)
	}
	id := resultado.Fechados[0].ID

	detalhe, err := c.fechamentos.Obter(c.ctx, id, c.usuarioCarlos)
	if err != nil || detalhe.Fechamento.ID != id {
		t.Fatalf("detalhe do Carlos = %+v, %v", detalhe, err)
	}

	usuarioBeatriz := c.novoUsuario(false, false, true, acessoA(c.mia))
	c.beatriz.UsuarioID = &usuarioBeatriz
	_, err = c.fechamentos.Obter(c.ctx, id, usuarioBeatriz)
	esperarErro(t, err, domain.ErrPermissao, "outro representante")

	_, err = c.fechamentos.Obter(c.ctx, uuid.New(), c.financeiro)
	esperarErro(t, err, domain.ErrNotFound, "fechamento não encontrado")

	if _, err := c.fechamentos.Listar(c.ctx, portsin.ListarFechamentosInput{UsuarioID: c.usuarioCarlos}); err != nil {
		t.Fatalf("Listar: %v", err)
	}
	if *c.m.ultimoFiltroFechados.RepresentanteID != c.carlos.ID {
		t.Fatalf("filtro = %+v", c.m.ultimoFiltroFechados)
	}
	_, err = c.fechamentos.Listar(c.ctx, portsin.ListarFechamentosInput{UsuarioID: c.comercial})
	esperarErro(t, err, domain.ErrPermissao, "ver comissões")
}

func TestApuracaoNaoRepeteAsEntregasQueFalharam(t *testing.T) {
	c := novoCenarioFechamento(t)
	semCadastro := c.novoPedido(c.mia, ptr(uuid.New()), domainpedido.ModeloProntaEntrega, domainpedido.StatusConfirmado)
	c.novaEntrega(semCadastro, data(2026, 9, 1), domaincomissao.ItemSaido{Quantidade: 1, PrecoUnitarioCentavos: 100})
	c.novaEntrega(semCadastro, data(2026, 9, 2), domaincomissao.ItemSaido{Quantidade: 1, PrecoUnitarioCentavos: 100})
	c.novaEntrega(c.pedidoDoCarlos, data(2026, 9, 3), domaincomissao.ItemSaido{Quantidade: 1, PrecoUnitarioCentavos: 100})

	resultado, err := c.comissoes.Apurar(c.ctx, portsin.ApurarInput{UsuarioID: c.financeiro})
	if err != nil {
		t.Fatalf("Apurar: %v", err)
	}
	if len(resultado.Apuradas) != 1 || len(resultado.Falhas) != 2 || resultado.RestamEntregas {
		t.Fatalf("resultado = %+v", resultado)
	}
	if len(c.m.ultimoFiltroEntregas.IgnorarIDs) != 2 || resultado.Falhas[0].RepresentanteID != *semCadastro.RepresentanteID {
		t.Fatalf("as falhas deveriam ser ignoradas na rodada seguinte: %+v / %+v", c.m.ultimoFiltroEntregas, resultado.Falhas)
	}
}

func TestFecharNaoFechaRepresentanteComEntregaSemApurar(t *testing.T) {
	c := novoCenarioFechamento(t)
	c.novaEntrega(c.pedidoDoCarlos, c.competencia.Inicio(), domaincomissao.ItemSaido{Quantidade: 1, PrecoUnitarioCentavos: 100})
	c.m.falhaNaGravacao[c.m.entregas[0].EntregaID] = domaincomissao.ErroPercentualMudouNaApuracao()
	c.m.saldos = []domaincomissao.SaldoAberto{{RepresentanteID: c.carlos.ID, ComissoesCentavos: 1000}}

	resultado, err := c.fechamentos.Fechar(c.ctx, c.input(c.financeiro))
	if err != nil {
		t.Fatalf("Fechar: %v", err)
	}
	if len(resultado.Fechados) != 0 || len(resultado.Falhas) != 1 || !strings.Contains(resultado.Falhas[0].Motivo, "não foram apuradas") {
		t.Fatalf("resultado = %+v", resultado)
	}
}

func TestFecharUmRepresentanteDevolveOMotivoComoErro(t *testing.T) {
	c := novoCenarioFechamento(t)
	c.m.saldos = []domaincomissao.SaldoAberto{{RepresentanteID: c.carlos.ID, ComissoesCentavos: 100, AjustesCentavos: -200}}
	input := c.input(c.financeiro)
	input.RepresentanteID = &c.carlos.ID

	_, err := c.fechamentos.Fechar(c.ctx, input)
	esperarErro(t, err, domain.ErrValidacao, "nada a pagar")

	c.m.saldos = []domaincomissao.SaldoAberto{{RepresentanteID: c.carlos.ID, ComissoesCentavos: 1000}}
	c.m.falhaNoFechar[c.carlos.ID] = domaincomissao.ErroFechamentoRepetido()
	_, err = c.fechamentos.Fechar(c.ctx, input)
	esperarErro(t, err, domain.ErrConflito, "já existe fechamento")
}
