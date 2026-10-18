package receber_test

import (
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/receber"
	"github.com/google/uuid"
)

func (c *cenario) receberInput(titulo *domainreceber.Titulo, valor int64) portsin.ReceberInput {
	return portsin.ReceberInput{
		UsuarioID:       c.financeiro,
		TituloID:        titulo.ID,
		Versao:          titulo.Versao,
		ContaBancariaID: &c.contaMia,
		RecebidoEm:      domainreceber.Hoje(time.Now().UTC()),
		ValorCentavos:   valor,
	}
}

func TestReceberEEstornarPeloUsecase(t *testing.T) {
	c := novoCenario(t)
	titulo := c.novoTitulo(t)

	recebido, recebimento, err := c.baixas.Receber(c.ctx, c.receberInput(titulo, 150000))
	if err != nil {
		t.Fatalf("Receber falhou: %v", err)
	}
	if recebido.Status != domainreceber.StatusRecebido || recebido.Versao != 2 || recebimento.RegistrouEntradaPedido || c.m.baixas[0].Entrada != nil {
		t.Fatalf("título = %+v, recebimento = %+v", recebido, recebimento)
	}

	recebimentos, err := c.baixas.ListarRecebimentos(c.ctx, titulo.ID, c.financeiro)
	if err != nil || len(recebimentos) != 1 {
		t.Fatalf("recebimentos = %+v, %v", recebimentos, err)
	}

	estornado, err := c.baixas.EstornarRecebimento(c.ctx, portsin.EstornarRecebimentoInput{
		UsuarioID: c.financeiro, TituloID: titulo.ID, RecebimentoID: recebimento.ID, Versao: 2,
		Motivo: "pix devolvido", EstornadoEm: domainreceber.Hoje(time.Now().UTC()),
	})
	if err != nil {
		t.Fatalf("EstornarRecebimento falhou: %v", err)
	}
	if estornado.Status != domainreceber.StatusCarteira || estornado.RecebidoCentavos != 0 || estornado.Versao != 3 {
		t.Fatalf("título estornado = %+v", estornado)
	}

	_, err = c.baixas.EstornarRecebimento(c.ctx, portsin.EstornarRecebimentoInput{
		UsuarioID: c.financeiro, TituloID: titulo.ID, RecebimentoID: uuid.New(), Versao: 3, Motivo: "x", EstornadoEm: time.Now(),
	})
	esperarErro(t, err, domain.ErrNotFound, "recebimento não encontrado")
}

func TestReceberConfereEstadoVersaoContaEAcesso(t *testing.T) {
	c := novoCenario(t)
	titulo := c.novoTitulo(t)

	input := c.receberInput(titulo, 1000)
	input.ContaBancariaID = &c.contaRed
	_, _, err := c.baixas.Receber(c.ctx, input)
	esperarErro(t, err, domain.ErrValidacao, "mesma empresa")

	input = c.receberInput(titulo, 1000)
	input.Versao = 9
	_, _, err = c.baixas.Receber(c.ctx, input)
	esperarErro(t, err, domain.ErrConflito, "alterado por outra pessoa")

	input = c.receberInput(titulo, 1000)
	input.UsuarioID = c.financeiroRed
	_, _, err = c.baixas.Receber(c.ctx, input)
	esperarErro(t, err, domain.ErrPermissao, "sem acesso")
	_, err = c.baixas.ListarRecebimentos(c.ctx, titulo.ID, c.comercial)
	esperarErro(t, err, domain.ErrPermissao, "contas a receber")

	cancelado, err := c.titulos.Cancelar(c.ctx, portsin.CancelarTituloInput{UsuarioID: c.financeiro, TituloID: titulo.ID, Versao: 1, Motivo: "errado"})
	if err != nil {
		t.Fatal(err)
	}
	input = c.receberInput(cancelado, 0)
	input.Tipo = "cheque"
	_, _, err = c.baixas.Receber(c.ctx, input)
	esperarErro(t, err, domain.ErrConflito, "recebe baixa")
}

func (c *cenario) tituloDeEntrada(t *testing.T, pedido *domainpedido.Pedido) *domainreceber.Titulo {
	t.Helper()

	titulo, _, err := c.entradas.GerarTituloDeEntrada(c.ctx, c.entradaInput(c.financeiro, pedido.ID))
	if err != nil {
		t.Fatalf("GerarTituloDeEntrada falhou: %v", err)
	}

	return titulo
}

func TestBaixaTotalDoTituloDeEntradaRegistraAEntradaNoPedido(t *testing.T) {
	c := novoCenario(t)
	pedido := c.novoPedido(c.mia, domainpedido.ModeloImportacao, domainpedido.StatusConfirmado)
	titulo := c.tituloDeEntrada(t, pedido)

	parcial, _, err := c.baixas.Receber(c.ctx, c.receberInput(titulo, 10000))
	if err != nil {
		t.Fatal(err)
	}
	if c.m.baixas[0].Entrada != nil || c.m.pedidos[pedido.ID].EntradaPagaEm != nil {
		t.Fatal("a baixa parcial não registra a entrada")
	}

	_, recebimento, err := c.baixas.Receber(c.ctx, c.receberInput(parcial, 20000))
	if err != nil {
		t.Fatalf("baixa total falhou: %v", err)
	}
	baixa := c.m.baixas[1]
	if baixa.Entrada == nil || !recebimento.RegistrouEntradaPedido || baixa.Entrada.VersaoLida != 3 ||
		baixa.Entrada.Historico.Acao != domainpedido.AcaoEntradaRegistrada {
		t.Fatalf("baixa = %+v", baixa)
	}
	atualizado := c.m.pedidos[pedido.ID]
	if atualizado.EntradaPagaEm == nil || *atualizado.EntradaPagaCentavos != 30000 || atualizado.PrazoLimite == nil {
		t.Fatalf("pedido = %+v", atualizado)
	}
	if !strings.HasSuffix(baixa.Historico.Descricao, "; entrada paga registrada no pedido") {
		t.Fatalf("histórico = %q", baixa.Historico.Descricao)
	}

	_, err = c.baixas.EstornarRecebimento(c.ctx, portsin.EstornarRecebimentoInput{
		UsuarioID: c.financeiro, TituloID: titulo.ID, RecebimentoID: recebimento.ID, Versao: 3, Motivo: "x", EstornadoEm: time.Now(),
	})
	esperarErro(t, err, domain.ErrConflito, "entrada paga do pedido")
}

func TestBaixaDoTituloDeEntradaSemEfeitoNoPedido(t *testing.T) {
	c := novoCenario(t)

	cancelado := c.novoPedido(c.mia, domainpedido.ModeloImportacao, domainpedido.StatusConfirmado)
	titulo := c.tituloDeEntrada(t, cancelado)
	c.m.pedidos[cancelado.ID].Status = domainpedido.StatusCancelado
	if _, _, err := c.baixas.Receber(c.ctx, c.receberInput(titulo, 30000)); err != nil {
		t.Fatalf("a baixa do pedido cancelado é aceita: %v", err)
	}
	if c.m.baixas[0].Entrada != nil || !strings.HasSuffix(c.m.baixas[0].Historico.Descricao, "pedido cancelado: a entrada não foi registrada") {
		t.Fatalf("baixa = %+v, %q", c.m.baixas[0], c.m.baixas[0].Historico.Descricao)
	}

	jaPago := c.novoPedido(c.mia, domainpedido.ModeloImportacao, domainpedido.StatusConfirmado)
	outro := c.tituloDeEntrada(t, jaPago)
	pagaEm := time.Now().UTC()
	c.m.pedidos[jaPago.ID].EntradaPagaEm = &pagaEm
	if _, _, err := c.baixas.Receber(c.ctx, c.receberInput(outro, 30000)); err != nil {
		t.Fatal(err)
	}
	if c.m.baixas[1].Entrada != nil || !strings.HasSuffix(c.m.baixas[1].Historico.Descricao, "já estava registrada; o pedido não mudou") {
		t.Fatalf("baixa = %q", c.m.baixas[1].Historico.Descricao)
	}
}
