package postgres_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	portsreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/receber"
	"github.com/google/uuid"
)

var _ portsreceber.PedidoRepository = (*postgres.PedidoRepository)(nil)

func (c *cenarioBancoTitulos) receber(t *testing.T, titulo *domainreceber.Titulo, valor int64, entrada *portsreceber.EntradaDoPedido) (*domainreceber.Recebimento, error) {
	t.Helper()

	versao, anterior := titulo.Versao, titulo.Status
	recebimento, historico, err := titulo.Receber(domainreceber.DadosRecebimento{
		ContaBancariaID: &c.contaID, RecebidoEm: diaDoTitulo(10, 15), ValorCentavos: valor, JurosCentavos: 50,
	}, c.solicitanteID, agoraDoTitulo)
	if err != nil {
		t.Fatalf("recebimento de teste inválido: %v", err)
	}
	if entrada != nil {
		recebimento.RegistrouEntradaPedido = true
	}

	return recebimento, c.titulos.Receber(c.ctx, &portsreceber.Baixa{
		Titulo: titulo, VersaoLida: versao, StatusAnterior: anterior, Recebimento: recebimento, Historico: historico, Entrada: entrada,
	})
}

func (c *cenarioBancoTitulos) estornar(t *testing.T, titulo *domainreceber.Titulo, recebimento *domainreceber.Recebimento) error {
	t.Helper()

	versao, anterior := titulo.Versao, titulo.Status
	_, historico, err := titulo.EstornarRecebimento(recebimento, "cheque devolvido", diaDoTitulo(10, 15), c.solicitanteID, agoraDoTitulo)
	if err != nil {
		t.Fatalf("estorno de teste inválido: %v", err)
	}

	return c.titulos.EstornarRecebimento(c.ctx, &portsreceber.EstornoDaBaixa{
		Titulo: titulo, VersaoLida: versao, StatusAnterior: anterior, Recebimento: recebimento, Historico: historico,
	})
}

func TestReceberParcialTotalEEstornar(t *testing.T) {
	c := novoCenarioTitulos(t)
	titulo := c.novoTitulo(t, "ALG-10", diaDoTitulo(11, 10))

	parcial, err := c.receber(t, titulo, 50000, nil)
	if err != nil {
		t.Fatalf("Receber parcial falhou: %v", err)
	}
	if obtido := c.obterTitulo(t, titulo.ID); obtido.RecebidoCentavos != 50000 || obtido.Status != domainreceber.StatusCarteira || obtido.Versao != 2 {
		t.Fatalf("título depois da parcial = %+v", obtido)
	}

	total, err := c.receber(t, titulo, 100000, nil)
	if err != nil {
		t.Fatalf("Receber total falhou: %v", err)
	}
	if obtido := c.obterTitulo(t, titulo.ID); obtido.Status != domainreceber.StatusRecebido || obtido.SaldoCentavos() != 0 {
		t.Fatalf("título quitado = %+v", obtido)
	}

	if err := c.estornar(t, titulo, total); err != nil {
		t.Fatalf("EstornarRecebimento falhou: %v", err)
	}
	obtido := c.obterTitulo(t, titulo.ID)
	if obtido.Status != domainreceber.StatusCarteira || obtido.RecebidoCentavos != 50000 || obtido.Versao != 4 {
		t.Fatalf("título estornado = %+v", obtido)
	}

	recebimentos, err := c.titulos.ListarRecebimentos(c.ctx, titulo.ID)
	if err != nil || len(recebimentos) != 2 {
		t.Fatalf("recebimentos = %+v, %v", recebimentos, err)
	}
	if recebimentos[0].ID != parcial.ID || recebimentos[0].Estornado || recebimentos[0].Estorno != nil || recebimentos[0].JurosCentavos != 50 {
		t.Fatalf("parcial = %+v", recebimentos[0])
	}
	if !recebimentos[1].Estornado || recebimentos[1].Estorno == nil || recebimentos[1].Estorno.Motivo != "cheque devolvido" ||
		!recebimentos[1].Estorno.EstornadoEm.Equal(diaDoTitulo(10, 15)) || *recebimentos[1].ContaBancariaID != c.contaID {
		t.Fatalf("total estornado = %+v", recebimentos[1])
	}

	recebimentos[1].Estornado = false
	recebimentos[1].Estorno = nil
	esperarErro(t, c.estornar(t, obtido, recebimentos[1]), domain.ErrConflito, "já foi estornado")

	historico, _ := c.titulos.ListarHistorico(c.ctx, titulo.ID)
	if len(historico) != 4 || historico[3].Evento != domainreceber.EventoRecebimentoEstornado {
		t.Fatalf("histórico = %+v", historico)
	}
}

func TestReceberComVersaoAntigaNaoGrava(t *testing.T) {
	c := novoCenarioTitulos(t)
	titulo := c.novoTitulo(t, "ALG-10", diaDoTitulo(11, 10))
	copia := *titulo

	if _, err := c.receber(t, titulo, 1000, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := c.receber(t, &copia, 1000, nil); !errors.Is(err, domain.ErrConflito) {
		t.Fatalf("baixa concorrente = %v", err)
	}
	if recebimentos, _ := c.titulos.ListarRecebimentos(c.ctx, titulo.ID); len(recebimentos) != 1 {
		t.Fatalf("só a primeira baixa grava: %d", len(recebimentos))
	}
}

func TestReceberConfereAContaNaTransacao(t *testing.T) {
	c := novoCenarioTitulos(t)
	titulo := c.novoTitulo(t, "ALG-10", diaDoTitulo(11, 10))
	if _, err := c.banco.Exec(`UPDATE conta_bancaria SET ativa = false WHERE id = $1`, c.contaID); err != nil {
		t.Fatal(err)
	}

	_, err := c.receber(t, titulo, 1000, nil)
	esperarErro(t, err, domain.ErrValidacao, "conta bancária inativa")
}

func (c *cenarioBancoTitulos) entradaDoPedido(t *testing.T, pedidoID uuid.UUID, pagaEm int, valor int64) *portsreceber.EntradaDoPedido {
	t.Helper()

	pedido, err := postgres.NewPedidoRepository(c.banco).Obter(c.ctx, pedidoID)
	if err != nil || pedido == nil {
		t.Fatalf("obter pedido = %v, %v", pedido, err)
	}
	versao, anterior := pedido.Versao, pedido.Status
	if err := pedido.RegistrarEntrada(diaDoTitulo(10, pagaEm), valor, agoraDoTitulo); err != nil {
		t.Fatalf("registrar entrada = %v", err)
	}

	return &portsreceber.EntradaDoPedido{
		Pedido: pedido, VersaoLida: versao, StatusAnterior: anterior,
		Historico: domainpedido.NovoHistorico(pedido.ID, c.solicitanteID, domainpedido.AcaoEntradaRegistrada, &anterior, pedido.Status, "pelo título", agoraDoTitulo),
	}
}

func TestReceberTituloDeEntradaRegistraAEntradaDoPedido(t *testing.T) {
	c := novoCenarioTitulos(t)
	pedidoID := c.inserirPedidoDeImportacao(t, "confirmado")
	resultado, err := c.titulos.GerarDeOrigem(c.ctx, c.geracao(t, domainreceber.OrigemPedido, nil, parcelaDoPedido(pedidoID, "PED-1")))
	if err != nil {
		t.Fatal(err)
	}
	titulo := c.obterTitulo(t, resultado.Gerados[0])

	entrada := c.entradaDoPedido(t, pedidoID, 15, 30000)
	recebimento, err := c.receber(t, titulo, 30000, entrada)
	if err != nil {
		t.Fatalf("Receber com entrada falhou: %v", err)
	}
	if entrada.Pedido.Versao != 2 {
		t.Fatalf("versão do pedido em memória = %d", entrada.Pedido.Versao)
	}

	pedido, err := postgres.NewPedidoRepository(c.banco).Obter(c.ctx, pedidoID)
	if err != nil || pedido.EntradaPagaEm == nil || *pedido.EntradaPagaCentavos != 30000 || pedido.PrazoLimite == nil || pedido.Versao != 2 {
		t.Fatalf("pedido = %+v, %v", pedido, err)
	}
	historico, _ := postgres.NewPedidoRepository(c.banco).ListarHistorico(c.ctx, pedidoID)
	if len(historico) != 1 || historico[0].Acao != domainpedido.AcaoEntradaRegistrada {
		t.Fatalf("histórico do pedido = %+v", historico)
	}

	recebimentos, _ := c.titulos.ListarRecebimentos(c.ctx, titulo.ID)
	if len(recebimentos) != 1 || !recebimentos[0].RegistrouEntradaPedido || recebimentos[0].ID != recebimento.ID {
		t.Fatalf("recebimentos = %+v", recebimentos)
	}

	if _, err := c.banco.Exec(`UPDATE titulo_recebimento SET registrou_entrada_pedido = true WHERE id = $1`, recebimento.ID); err != nil {
		t.Fatal(err)
	}
	recebimentos[0].RegistrouEntradaPedido = false
	esperarErro(t, c.estornar(t, c.obterTitulo(t, titulo.ID), recebimentos[0]), domain.ErrConflito, "já foi estornado")
}

func TestReceberComPedidoAlteradoDesfazABaixa(t *testing.T) {
	c := novoCenarioTitulos(t)
	pedidoID := c.inserirPedidoDeImportacao(t, "confirmado")
	resultado, err := c.titulos.GerarDeOrigem(c.ctx, c.geracao(t, domainreceber.OrigemPedido, nil, parcelaDoPedido(pedidoID, "PED-1")))
	if err != nil {
		t.Fatal(err)
	}
	titulo := c.obterTitulo(t, resultado.Gerados[0])
	entrada := c.entradaDoPedido(t, pedidoID, 15, 30000)
	if _, err := c.banco.Exec(`UPDATE pedido SET versao = versao + 1 WHERE id = $1`, pedidoID); err != nil {
		t.Fatal(err)
	}

	if _, err := c.receber(t, titulo, 30000, entrada); !errors.Is(err, domain.ErrConflito) {
		t.Fatalf("pedido alterado no meio = %v", err)
	}
	if obtido := c.obterTitulo(t, titulo.ID); obtido.RecebidoCentavos != 0 || obtido.Versao != 1 {
		t.Fatalf("a baixa foi desfeita junto: %+v", obtido)
	}
	if recebimentos, _ := c.titulos.ListarRecebimentos(c.ctx, titulo.ID); len(recebimentos) != 0 {
		t.Fatalf("recebimento gravado sem o pedido: %+v", recebimentos)
	}
}
