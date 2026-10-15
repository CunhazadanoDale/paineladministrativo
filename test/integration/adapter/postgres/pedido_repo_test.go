package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portspedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/pedido"
	"github.com/google/uuid"
)

var (
	_ portspedido.EstoqueRepository = (*postgres.EstoqueRepository)(nil)
	_ portspedido.ReservaRepository = (*postgres.ReservaRepository)(nil)
	_ portspedido.LocalRepository   = (*postgres.LocalRepository)(nil)
)

type cenarioBancoPedidos struct {
	*cenarioBancoLocais
	pedidos   *postgres.PedidoRepository
	entregas  *postgres.EntregaRepository
	reservas  *postgres.ReservaRepository
	clienteID uuid.UUID
	segundo   *domainestoque.Produto
	galpao    *domainestoque.Local
	loja      *domainestoque.Local
}

func novoCenarioPedidos(t *testing.T) *cenarioBancoPedidos {
	t.Helper()

	base := novoCenarioLocais(t)
	raiz := base.novaCategoriaRaiz(t, "Acabamento")
	sub := base.novaSubcategoria(t, "Porcelanato", raiz)

	documento, err := domaincliente.NovoDocumento("11.444.777/0001-61")
	if err != nil {
		t.Fatalf("documento de teste inválido: %v", err)
	}
	cliente, err := domaincliente.NovoCliente(documento, domaincliente.DadosCliente{
		Nome:   "Construtora Alfa",
		Tipo:   domaincliente.TipoClienteConstrutora,
		UF:     domainempresa.UFDe("PB"),
		Cidade: "João Pessoa",
	}, nil)
	if err != nil {
		t.Fatalf("cliente de teste inválido: %v", err)
	}
	if _, err := postgres.NewClienteRepository(base.banco).Criar(base.ctx, cliente); err != nil {
		t.Fatalf("criação do cliente falhou: %v", err)
	}

	c := &cenarioBancoPedidos{
		cenarioBancoLocais: base,
		pedidos:            postgres.NewPedidoRepository(base.banco),
		entregas:           postgres.NewEntregaRepository(base.banco),
		reservas:           postgres.NewReservaRepository(base.banco),
		clienteID:          cliente.ID,
		segundo:            base.novoProduto(t, sub, "Porcelanato 60x60"),
	}
	c.galpao = base.novoLocal(t, base.mia, "Galpão")
	c.loja = base.novoLocal(t, base.mia, "Loja")

	return c
}

func (c *cenarioBancoPedidos) dados(modelo domainpedido.Modelo) domainpedido.DadosPedido {
	return domainpedido.DadosPedido{
		ClienteID: c.clienteID,
		Modelo:    modelo,
		Itens: []domainpedido.ItemInput{
			{ProdutoID: c.produto.ID, Quantidade: 10, PrecoUnitarioCentavos: 1500},
			{ProdutoID: c.segundo.ID, Quantidade: 4, PrecoUnitarioCentavos: 2500},
		},
	}
}

func (c *cenarioBancoPedidos) gravar(t *testing.T, gravacao *portspedido.Gravacao) {
	t.Helper()

	if err := c.pedidos.Gravar(c.ctx, gravacao); err != nil {
		t.Fatalf("gravar o pedido falhou: %v", err)
	}
}

func (c *cenarioBancoPedidos) novoPedido(t *testing.T, empresaID uuid.UUID, dados domainpedido.DadosPedido) *domainpedido.Pedido {
	t.Helper()

	pedido, err := domainpedido.NovoPedido(empresaID, c.usuarioID, dados)
	if err != nil {
		t.Fatalf("pedido de teste inválido: %v", err)
	}
	c.gravar(t, &portspedido.Gravacao{
		Pedido: pedido,
		Novo:   true,
		Historicos: []*domainpedido.Historico{
			domainpedido.NovoHistorico(pedido.ID, c.usuarioID, domainpedido.AcaoCriado, nil, pedido.Status, "", pedido.CriadoEm),
		},
	})

	return pedido
}

func (c *cenarioBancoPedidos) obter(t *testing.T, id uuid.UUID) *domainpedido.Pedido {
	t.Helper()

	pedido, err := c.pedidos.Obter(c.ctx, id)
	if err != nil || pedido == nil {
		t.Fatalf("obter o pedido = %+v, %v", pedido, err)
	}

	return pedido
}

func (c *cenarioBancoPedidos) estoqueDe(t *testing.T, produto *domainestoque.Produto) *domainestoque.EstoqueProduto {
	t.Helper()

	estoque, err := c.estoque.Carregar(c.ctx, produto.ID, []uuid.UUID{c.galpao.ID, c.loja.ID})
	if err != nil {
		t.Fatalf("carregar o estoque falhou: %v", err)
	}

	return estoque
}

func (c *cenarioBancoPedidos) entrada(t *testing.T, produto *domainestoque.Produto, local *domainestoque.Local, quantidade int) {
	t.Helper()

	custo := domainestoque.CustoDe(1000)
	operacao, err := c.estoqueDe(t, produto).Movimentar(domainestoque.MovimentoLocalInput{
		Tipo:       domainestoque.TipoMovimentoEntrada,
		Local:      local,
		Quantidade: domainestoque.QuantidadeDe(quantidade),
		Custo:      &custo,
		UsuarioID:  c.usuarioID,
		Agora:      time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("entrada de domínio falhou: %v", err)
	}
	if err := c.estoque.Gravar(c.ctx, operacao); err != nil {
		t.Fatalf("gravar a entrada falhou: %v", err)
	}
}

func (c *cenarioBancoPedidos) confirmarReservando(t *testing.T, pedido *domainpedido.Pedido) {
	t.Helper()

	agora := time.Now().UTC()
	versaoLida := pedido.Versao
	if err := pedido.Confirmar(agora); err != nil {
		t.Fatalf("Confirmar falhou: %v", err)
	}

	operacoes := make([]*domainestoque.OperacaoEstoque, 0)
	locais := map[uuid.UUID]*domainestoque.Local{c.galpao.ID: c.galpao, c.loja.ID: c.loja}
	for _, item := range pedido.Itens {
		produto := c.produto
		if item.ProdutoID == c.segundo.ID {
			produto = c.segundo
		}
		estoque := c.estoqueDe(t, produto)

		disponiveis := []domainpedido.DisponivelNoLocal{
			{LocalID: c.galpao.ID, Disponivel: estoque.SaldoNoLocal(c.galpao.ID).Disponivel()},
			{LocalID: c.loja.ID, Disponivel: estoque.SaldoNoLocal(c.loja.ID).Disponivel()},
		}
		for _, planejada := range domainpedido.DistribuirReserva(item.FaltaCobrir(0), disponiveis) {
			itemID := item.ID
			operacao, err := estoque.Reservar(domainestoque.ReservaInput{
				Local:      locais[planejada.LocalID],
				Quantidade: domainestoque.QuantidadeDe(planejada.Quantidade),
				Origem:     domainestoque.OrigemReservaPedidoItem,
				OrigemID:   &itemID,
				UsuarioID:  c.usuarioID,
				Agora:      agora,
			})
			if err != nil {
				t.Fatalf("reserva de domínio falhou: %v", err)
			}
			if err := estoque.Aplicar(operacao); err != nil {
				t.Fatalf("aplicar a reserva falhou: %v", err)
			}
			operacoes = append(operacoes, operacao)
		}
	}

	anterior := domainpedido.StatusProposta
	c.gravar(t, &portspedido.Gravacao{
		Pedido:           pedido,
		VersaoLida:       versaoLida,
		StatusAnterior:   anterior,
		OperacoesEstoque: operacoes,
		Historicos: []*domainpedido.Historico{
			domainpedido.NovoHistorico(pedido.ID, c.usuarioID, domainpedido.AcaoConfirmado, &anterior, pedido.Status, "", agora),
		},
	})
}

func (c *cenarioBancoPedidos) novaEntrega(t *testing.T, pedido *domainpedido.Pedido, local *domainestoque.Local, quantidade int) *domainpedido.Entrega {
	t.Helper()

	agora := time.Now().UTC()
	entrega, err := pedido.NovaEntrega(c.usuarioID, domainpedido.DadosEntrega{
		Frete:    domainpedido.FreteDe(domainpedido.TipoFreteFOB, nil),
		Previsao: &agora,
		Itens: []domainpedido.EntregaItemInput{
			{PedidoItemID: pedido.Itens[0].ID, Local: local, Quantidade: quantidade},
		},
	}, nil, agora)
	if err != nil {
		t.Fatalf("entrega de teste inválida: %v", err)
	}

	c.gravar(t, &portspedido.Gravacao{
		Pedido:         pedido,
		VersaoLida:     pedido.Versao,
		StatusAnterior: pedido.Status,
		EntregasNovas:  []*domainpedido.Entrega{entrega},
		Historicos: []*domainpedido.Historico{
			domainpedido.NovoHistorico(pedido.ID, c.usuarioID, domainpedido.AcaoEntregaCriada, nil, pedido.Status, "", agora),
		},
	})

	return entrega
}

func (c *cenarioBancoPedidos) gravacaoDaSaida(t *testing.T, pedido *domainpedido.Pedido, entrega *domainpedido.Entrega) *portspedido.Gravacao {
	t.Helper()

	agora := time.Now().UTC()
	versaoDoPedido, statusDoPedido := pedido.Versao, pedido.Status
	versaoDaEntrega := entrega.Versao
	item := entrega.Itens[0]

	reservas, err := c.reservas.AtivasPorOrigem(c.ctx, []uuid.UUID{item.PedidoItemID})
	if err != nil {
		t.Fatalf("reservas do item falharam: %v", err)
	}
	doLocal := make([]*domainestoque.Reserva, 0, len(reservas))
	for _, reserva := range reservas {
		if reserva.LocalID == item.LocalID {
			doLocal = append(doLocal, reserva)
		}
	}
	plano, err := domainpedido.PlanejarBaixa(item, doLocal)
	if err != nil {
		t.Fatalf("PlanejarBaixa falhou: %v", err)
	}

	estoque := c.estoqueDe(t, c.produto)
	operacoes := make([]*domainestoque.OperacaoEstoque, 0, 3)
	aplicar := func(operacao *domainestoque.OperacaoEstoque, err error) *domainestoque.OperacaoEstoque {
		t.Helper()
		if err != nil {
			t.Fatalf("operação de estoque falhou: %v", err)
		}
		if err := estoque.Aplicar(operacao); err != nil {
			t.Fatalf("aplicar falhou: %v", err)
		}
		operacoes = append(operacoes, operacao)
		return operacao
	}

	for _, reserva := range plano.Liberar {
		aplicar(estoque.LiberarReserva(reserva, agora))
	}
	saida := aplicar(estoque.Movimentar(domainestoque.MovimentoLocalInput{
		Tipo:       domainestoque.TipoMovimentoSaida,
		Local:      c.galpao,
		Quantidade: domainestoque.QuantidadeDe(plano.Sair),
		UsuarioID:  c.usuarioID,
		Agora:      agora,
	}))
	if plano.ReservarDeNovo > 0 {
		itemID := item.PedidoItemID
		aplicar(estoque.Reservar(domainestoque.ReservaInput{
			Local:      c.galpao,
			Quantidade: domainestoque.QuantidadeDe(plano.ReservarDeNovo),
			Origem:     domainestoque.OrigemReservaPedidoItem,
			OrigemID:   &itemID,
			UsuarioID:  c.usuarioID,
			Agora:      agora,
		}))
	}
	item.MovimentoID = &saida.Movimento.ID

	if err := pedido.RegistrarSaida(entrega, agora, agora); err != nil {
		t.Fatalf("RegistrarSaida falhou: %v", err)
	}

	return &portspedido.Gravacao{
		Pedido:         pedido,
		VersaoLida:     versaoDoPedido,
		StatusAnterior: statusDoPedido,
		EntregasAlteradas: []portspedido.EntregaAlterada{
			{Entrega: entrega, VersaoLida: versaoDaEntrega, StatusAnterior: domainpedido.StatusEntregaPendente},
		},
		OperacoesEstoque: operacoes,
		Historicos: []*domainpedido.Historico{
			domainpedido.NovoHistorico(pedido.ID, c.usuarioID, domainpedido.AcaoEntregaSaiu, &statusDoPedido, pedido.Status, "", agora),
		},
	}
}

func TestPedidoNumeraPorEmpresaEObtemComItens(t *testing.T) {
	c := novoCenarioPedidos(t)
	prazo := time.Date(2026, 12, 20, 15, 0, 0, 0, time.UTC)
	numeroDoCliente := "OC-778"
	dados := c.dados(domainpedido.ModeloProntaEntrega)
	dados.PrazoLimite = &prazo
	dados.NumeroPedidoCliente = &numeroDoCliente
	dados.Observacoes = "entregar pela manhã"

	primeiro := c.novoPedido(t, c.mia, dados)
	segundo := c.novoPedido(t, c.mia, c.dados(domainpedido.ModeloImportacao))
	daRed := c.novoPedido(t, c.red, c.dados(domainpedido.ModeloProntaEntrega))

	if primeiro.Numero != 1 || segundo.Numero != 2 || daRed.Numero != 1 {
		t.Errorf("números = %d, %d e %d; esperado 1, 2 e 1", primeiro.Numero, segundo.Numero, daRed.Numero)
	}

	salvo := c.obter(t, primeiro.ID)
	if salvo.Numero != 1 || salvo.Versao != 1 || salvo.Status != domainpedido.StatusProposta || salvo.ValorTotalCentavos != 25000 {
		t.Errorf("pedido salvo = número %d, versão %d, status %q, total %d", salvo.Numero, salvo.Versao, salvo.Status, salvo.ValorTotalCentavos)
	}
	if salvo.PrazoLimite == nil || !salvo.PrazoLimite.Equal(time.Date(2026, 12, 20, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("prazo-limite = %v", salvo.PrazoLimite)
	}
	if salvo.NumeroPedidoCliente == nil || *salvo.NumeroPedidoCliente != "OC-778" || salvo.Observacoes != "entregar pela manhã" {
		t.Errorf("textos = %v / %q", salvo.NumeroPedidoCliente, salvo.Observacoes)
	}
	if boleto := salvo.Condicao.PrazoBoletoDias(); boleto == nil || *boleto != 30 || salvo.Condicao.PrazoEntregaDias() != nil {
		t.Errorf("condição da pronta entrega = boleto %v, entrega %v", boleto, salvo.Condicao.PrazoEntregaDias())
	}
	if len(salvo.Itens) != 2 || salvo.Itens[0].ID != primeiro.Itens[0].ID || salvo.Itens[1].ProdutoID != c.segundo.ID ||
		salvo.Itens[1].PrecoUnitario.Centavos() != 2500 || salvo.Itens[0].QuantidadeEntregue != 0 {
		t.Errorf("itens salvos = %+v", salvo.Itens)
	}

	importacao := c.obter(t, segundo.ID)
	if importacao.Condicao.Entrada().PontosBase() != 3000 || *importacao.Condicao.PrazoEntregaDias() != 120 || importacao.Condicao.PrazoBoletoDias() != nil {
		t.Errorf("condição da importação = entrada %d, entrega %v", importacao.Condicao.Entrada().PontosBase(), importacao.Condicao.PrazoEntregaDias())
	}

	if ausente, err := c.pedidos.Obter(c.ctx, uuid.New()); err != nil || ausente != nil {
		t.Errorf("pedido inexistente = %+v, %v; esperado nil sem erro", ausente, err)
	}
}

func TestPedidoAlteraPropostaComGuardaDeVersao(t *testing.T) {
	c := novoCenarioPedidos(t)
	pedido := c.novoPedido(t, c.mia, c.dados(domainpedido.ModeloProntaEntrega))

	dados := c.dados(domainpedido.ModeloProntaEntrega)
	dados.Itens = dados.Itens[1:]
	if err := pedido.Alterar(dados, time.Now().UTC()); err != nil {
		t.Fatalf("Alterar falhou: %v", err)
	}
	c.gravar(t, &portspedido.Gravacao{Pedido: pedido, VersaoLida: 1, StatusAnterior: domainpedido.StatusProposta, SubstituirItens: true})

	if pedido.Versao != 2 {
		t.Errorf("versão depois de gravar = %d, esperado 2", pedido.Versao)
	}
	salvo := c.obter(t, pedido.ID)
	if salvo.Versao != 2 || len(salvo.Itens) != 1 || salvo.Itens[0].ProdutoID != c.segundo.ID || salvo.ValorTotalCentavos != 10000 {
		t.Errorf("pedido alterado = versão %d, itens %+v, total %d", salvo.Versao, salvo.Itens, salvo.ValorTotalCentavos)
	}

	velho := &portspedido.Gravacao{Pedido: pedido, VersaoLida: 1, StatusAnterior: domainpedido.StatusProposta, SubstituirItens: true}
	if err := c.pedidos.Gravar(c.ctx, velho); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("versão velha = %v, esperado conflito", err)
	}

	inexistente, err := domainpedido.NovoPedido(c.mia, c.usuarioID, c.dados(domainpedido.ModeloProntaEntrega))
	if err != nil {
		t.Fatalf("pedido de teste inválido: %v", err)
	}
	if err := c.pedidos.Gravar(c.ctx, &portspedido.Gravacao{Pedido: inexistente, VersaoLida: 1, StatusAnterior: domainpedido.StatusProposta}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("pedido inexistente = %v, esperado não encontrado", err)
	}
}

func TestPedidoConfirmaReservandoEstoqueNaMesmaTransacao(t *testing.T) {
	c := novoCenarioPedidos(t)
	c.entrada(t, c.produto, c.galpao, 6)
	c.entrada(t, c.produto, c.loja, 3)
	pedido := c.novoPedido(t, c.mia, c.dados(domainpedido.ModeloProntaEntrega))

	c.confirmarReservando(t, pedido)

	salvo := c.obter(t, pedido.ID)
	if salvo.Status != domainpedido.StatusConfirmado || salvo.Versao != 2 {
		t.Errorf("pedido confirmado = %q, versão %d", salvo.Status, salvo.Versao)
	}

	reservas, err := c.reservas.AtivasPorOrigem(c.ctx, []uuid.UUID{pedido.Itens[0].ID, pedido.Itens[1].ID})
	if err != nil {
		t.Fatalf("AtivasPorOrigem falhou: %v", err)
	}
	porLocal := map[uuid.UUID]int{}
	for _, reserva := range reservas {
		if reserva.Origem != domainestoque.OrigemReservaPedidoItem || reserva.OrigemID == nil || *reserva.OrigemID != pedido.Itens[0].ID {
			t.Errorf("reserva fora do item 1: %+v", reserva)
		}
		porLocal[reserva.LocalID] += reserva.Quantidade
	}
	if porLocal[c.galpao.ID] != 6 || porLocal[c.loja.ID] != 3 {
		t.Errorf("reservado por local = %v, esperado 6 no galpão e 3 na loja", porLocal)
	}

	estoque := c.estoqueDe(t, c.produto)
	if estoque.Produto.Reservado.Quantidade() != 9 || estoque.SaldoNoLocal(c.galpao.ID).Disponivel() != 0 {
		t.Errorf("estoque depois de confirmar: reservado %d, disponível no galpão %d",
			estoque.Produto.Reservado.Quantidade(), estoque.SaldoNoLocal(c.galpao.ID).Disponivel())
	}

	historico, err := c.pedidos.ListarHistorico(c.ctx, pedido.ID)
	if err != nil || len(historico) != 2 || historico[0].Acao != domainpedido.AcaoCriado || historico[1].Acao != domainpedido.AcaoConfirmado ||
		historico[1].StatusAnterior == nil || *historico[1].StatusAnterior != domainpedido.StatusProposta {
		t.Errorf("histórico = %+v (%v)", historico, err)
	}
}

func TestEntregaParcialLiberaSaiEReservaDeNovo(t *testing.T) {
	c := novoCenarioPedidos(t)
	c.entrada(t, c.produto, c.galpao, 8)
	pedido := c.novoPedido(t, c.mia, c.dados(domainpedido.ModeloProntaEntrega))
	c.confirmarReservando(t, pedido)

	entrega := c.novaEntrega(t, pedido, c.galpao, 5)
	pendente, err := c.entregas.Obter(c.ctx, entrega.ID)
	if err != nil || pendente == nil || pendente.Status != domainpedido.StatusEntregaPendente || len(pendente.Itens) != 1 || pendente.Itens[0].MovimentoID != nil {
		t.Fatalf("entrega pendente = %+v (%v)", pendente, err)
	}

	pedido = c.obter(t, pedido.ID)
	c.gravar(t, c.gravacaoDaSaida(t, pedido, pendente))

	saiu, err := c.entregas.Obter(c.ctx, entrega.ID)
	if err != nil || saiu.Status != domainpedido.StatusEntregaSaiu || saiu.SaiuEm == nil || saiu.Versao != 2 || saiu.Itens[0].MovimentoID == nil {
		t.Errorf("entrega depois da saída = %+v (%v)", saiu, err)
	}
	if pendente.Versao != 2 {
		t.Errorf("versão da entrega em memória = %d, esperado 2", pendente.Versao)
	}

	salvo := c.obter(t, pedido.ID)
	if salvo.Status != domainpedido.StatusParcialmenteEntregue || salvo.Itens[0].QuantidadeEntregue != 5 || salvo.Itens[1].QuantidadeEntregue != 0 {
		t.Errorf("pedido depois da saída = %q, entregues %d e %d", salvo.Status, salvo.Itens[0].QuantidadeEntregue, salvo.Itens[1].QuantidadeEntregue)
	}

	reservas, err := c.reservas.AtivasPorOrigem(c.ctx, []uuid.UUID{pedido.Itens[0].ID})
	if err != nil || len(reservas) != 1 || reservas[0].Quantidade != 3 || reservas[0].LocalID != c.galpao.ID {
		t.Errorf("reservas depois da saída = %+v (%v), esperado uma de 3 no galpão", reservas, err)
	}

	estoque := c.estoqueDe(t, c.produto)
	if saldo := estoque.SaldoNoLocal(c.galpao.ID); saldo.Quantidade != 3 || saldo.Reservado != 3 {
		t.Errorf("galpão depois da saída = %+v, esperado 3 com 3 reservados", saldo)
	}

	movimento := *saiu.Itens[0].MovimentoID
	var tipo string
	var quantidade int
	if err := c.banco.GetContext(c.ctx, &tipo, `SELECT tipo FROM produto_movimento WHERE id = $1`, movimento); err != nil || tipo != "saida" {
		t.Errorf("movimento vinculado = %q (%v), esperado saída", tipo, err)
	}
	if err := c.banco.GetContext(c.ctx, &quantidade, `SELECT quantidade FROM produto_movimento WHERE id = $1`, movimento); err != nil || quantidade != 5 {
		t.Errorf("quantidade do movimento = %d (%v), esperado 5", quantidade, err)
	}
}

func TestGravarDesfazTudoQuandoOEstoqueConflita(t *testing.T) {
	c := novoCenarioPedidos(t)
	c.entrada(t, c.produto, c.galpao, 8)
	pedido := c.novoPedido(t, c.mia, c.dados(domainpedido.ModeloProntaEntrega))
	c.confirmarReservando(t, pedido)
	entrega := c.novaEntrega(t, pedido, c.galpao, 5)

	pedido = c.obter(t, pedido.ID)
	pendente, err := c.entregas.Obter(c.ctx, entrega.ID)
	if err != nil {
		t.Fatalf("obter a entrega falhou: %v", err)
	}
	versaoAntes := pedido.Versao
	gravacao := c.gravacaoDaSaida(t, pedido, pendente)

	c.entrada(t, c.produto, c.loja, 1)

	if err := c.pedidos.Gravar(c.ctx, gravacao); !errors.Is(err, domain.ErrConflito) {
		t.Fatalf("gravar com estoque alterado = %v, esperado conflito", err)
	}

	salvo := c.obter(t, pedido.ID)
	if salvo.Versao != versaoAntes || salvo.Status != domainpedido.StatusConfirmado || salvo.Itens[0].QuantidadeEntregue != 0 {
		t.Errorf("pedido depois do conflito = versão %d, %q, entregue %d", salvo.Versao, salvo.Status, salvo.Itens[0].QuantidadeEntregue)
	}
	intacta, err := c.entregas.Obter(c.ctx, entrega.ID)
	if err != nil || intacta.Status != domainpedido.StatusEntregaPendente || intacta.Versao != 1 || intacta.Itens[0].MovimentoID != nil {
		t.Errorf("entrega depois do conflito = %+v (%v)", intacta, err)
	}
	reservas, err := c.reservas.AtivasPorOrigem(c.ctx, []uuid.UUID{pedido.Itens[0].ID})
	if err != nil || len(reservas) != 1 || reservas[0].Quantidade != 8 {
		t.Errorf("reservas depois do conflito = %+v (%v), esperado a de 8 intacta", reservas, err)
	}
}

func TestEntregaComVersaoVelhaNaoGrava(t *testing.T) {
	c := novoCenarioPedidos(t)
	c.entrada(t, c.produto, c.galpao, 8)
	pedido := c.novoPedido(t, c.mia, c.dados(domainpedido.ModeloProntaEntrega))
	c.confirmarReservando(t, pedido)
	entrega := c.novaEntrega(t, pedido, c.galpao, 2)

	agora := time.Now().UTC()
	if err := entrega.Cancelar(agora); err != nil {
		t.Fatalf("Cancelar falhou: %v", err)
	}
	pedido = c.obter(t, pedido.ID)
	gravacao := &portspedido.Gravacao{
		Pedido:         pedido,
		VersaoLida:     pedido.Versao,
		StatusAnterior: pedido.Status,
		EntregasAlteradas: []portspedido.EntregaAlterada{
			{Entrega: entrega, VersaoLida: 7, StatusAnterior: domainpedido.StatusEntregaPendente},
		},
	}

	if err := c.pedidos.Gravar(c.ctx, gravacao); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("entrega com versão velha = %v, esperado conflito", err)
	}
	if salvo := c.obter(t, pedido.ID); salvo.Versao != pedido.Versao {
		t.Errorf("pedido avançou para a versão %d mesmo com a entrega recusada", salvo.Versao)
	}
}

func TestPedidoCanceladoGuardaMotivoEDevolucao(t *testing.T) {
	c := novoCenarioPedidos(t)
	pedido := c.novoPedido(t, c.mia, c.dados(domainpedido.ModeloImportacao))
	agora := time.Now().UTC()

	versao := pedido.Versao
	if err := pedido.Confirmar(agora); err != nil {
		t.Fatalf("Confirmar falhou: %v", err)
	}
	if err := pedido.RegistrarEntrada(agora, 7500, agora); err != nil {
		t.Fatalf("RegistrarEntrada falhou: %v", err)
	}
	c.gravar(t, &portspedido.Gravacao{Pedido: pedido, VersaoLida: versao, StatusAnterior: domainpedido.StatusProposta})

	devolver := int64(5000)
	if _, err := pedido.Cancelar(domainpedido.CancelamentoInput{Motivo: "atraso do fornecedor", ValorADevolverCentavos: &devolver}, agora); err != nil {
		t.Fatalf("Cancelar falhou: %v", err)
	}
	c.gravar(t, &portspedido.Gravacao{Pedido: pedido, VersaoLida: 2, StatusAnterior: domainpedido.StatusConfirmado})

	salvo := c.obter(t, pedido.ID)
	hoje := domainpedido.DataDe(agora)
	if salvo.Status != domainpedido.StatusCancelado || salvo.CanceladoMotivo == nil || *salvo.CanceladoMotivo != "atraso do fornecedor" {
		t.Errorf("cancelamento = %q, motivo %v", salvo.Status, salvo.CanceladoMotivo)
	}
	if salvo.ValorADevolverCentavos == nil || *salvo.ValorADevolverCentavos != 5000 || salvo.EntradaPagaCentavos == nil || *salvo.EntradaPagaCentavos != 7500 {
		t.Errorf("valores = devolver %v, entrada %v", salvo.ValorADevolverCentavos, salvo.EntradaPagaCentavos)
	}
	if salvo.EntradaPagaEm == nil || !salvo.EntradaPagaEm.Equal(hoje) || salvo.PrazoLimite == nil || !salvo.PrazoLimite.Equal(hoje.AddDate(0, 0, 120)) {
		t.Errorf("datas = entrada %v, prazo %v", salvo.EntradaPagaEm, salvo.PrazoLimite)
	}

	invalido := c.obter(t, pedido.ID)
	acima := int64(9000)
	invalido.ValorADevolverCentavos = &acima
	if err := c.pedidos.Gravar(c.ctx, &portspedido.Gravacao{Pedido: invalido, VersaoLida: invalido.Versao, StatusAnterior: domainpedido.StatusCancelado}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("devolução acima da entrada no banco = %v, esperado erro de validação", err)
	}
}

func TestListarPedidosFiltra(t *testing.T) {
	c := novoCenarioPedidos(t)
	primeiro := c.novoPedido(t, c.mia, c.dados(domainpedido.ModeloProntaEntrega))
	segundo := c.novoPedido(t, c.mia, c.dados(domainpedido.ModeloImportacao))
	daRed := c.novoPedido(t, c.red, c.dados(domainpedido.ModeloProntaEntrega))
	c.confirmarReservando(t, segundo)

	listar := func(filtro portspedido.PedidoFiltro) []uuid.UUID {
		t.Helper()
		filtro.PaginacaoFiltro = todasAsPaginas
		pedidos, err := c.pedidos.Listar(c.ctx, filtro)
		if err != nil {
			t.Fatalf("Listar falhou: %v", err)
		}
		ids := make([]uuid.UUID, 0, len(pedidos))
		for _, pedido := range pedidos {
			ids = append(ids, pedido.ID)
		}
		return ids
	}
	conferir := func(nome string, obtidos []uuid.UUID, esperados ...uuid.UUID) {
		t.Helper()
		if len(obtidos) != len(esperados) {
			t.Errorf("%s = %v, esperado %v", nome, obtidos, esperados)
			return
		}
		for indice := range esperados {
			if obtidos[indice] != esperados[indice] {
				t.Errorf("%s = %v, esperado %v", nome, obtidos, esperados)
				return
			}
		}
	}

	numeroDois := 2
	conferir("todos", listar(portspedido.PedidoFiltro{}), daRed.ID, segundo.ID, primeiro.ID)
	conferir("empresa", listar(portspedido.PedidoFiltro{Empresa: &c.mia}), segundo.ID, primeiro.ID)
	conferir("acesso à Red", listar(portspedido.PedidoFiltro{EmpresaIDs: []uuid.UUID{c.red}}), daRed.ID)
	conferir("sem acesso", listar(portspedido.PedidoFiltro{EmpresaIDs: []uuid.UUID{}}))
	conferir("status", listar(portspedido.PedidoFiltro{Status: domainpedido.StatusConfirmado}), segundo.ID)
	conferir("modelo", listar(portspedido.PedidoFiltro{Modelo: domainpedido.ModeloImportacao}), segundo.ID)
	conferir("número", listar(portspedido.PedidoFiltro{Empresa: &c.mia, Numero: &numeroDois}), segundo.ID)
	conferir("cliente", listar(portspedido.PedidoFiltro{ClienteID: &c.clienteID, Empresa: &c.red}), daRed.ID)
}

func TestListarEntregasFiltra(t *testing.T) {
	c := novoCenarioPedidos(t)
	c.entrada(t, c.produto, c.galpao, 20)
	daMia := c.novoPedido(t, c.mia, c.dados(domainpedido.ModeloProntaEntrega))
	c.confirmarReservando(t, daMia)
	primeira := c.novaEntrega(t, daMia, c.galpao, 2)
	segunda := c.novaEntrega(t, daMia, c.galpao, 3)

	localDaRed := c.novoLocal(t, c.red, "Galpão da Red")
	representante, err := domaincliente.NovoRepresentante(domaincliente.DadosRepresentante{Nome: "Carlos Representante"})
	if err != nil {
		t.Fatalf("representante de teste inválido: %v", err)
	}
	if _, err := postgres.NewRepresentanteRepository(c.banco).Criar(c.ctx, representante); err != nil {
		t.Fatalf("criação do representante falhou: %v", err)
	}
	dadosDaRed := c.dados(domainpedido.ModeloProntaEntrega)
	dadosDaRed.RepresentanteID = &representante.ID
	daRed := c.novoPedido(t, c.red, dadosDaRed)
	if err := daRed.Confirmar(time.Now().UTC()); err != nil {
		t.Fatalf("Confirmar falhou: %v", err)
	}
	c.gravar(t, &portspedido.Gravacao{Pedido: daRed, VersaoLida: 1, StatusAnterior: domainpedido.StatusProposta})
	daRedEntrega := c.novaEntrega(t, daRed, localDaRed, 1)

	doPedido, err := c.entregas.ListarDoPedido(c.ctx, daMia.ID)
	if err != nil || len(doPedido) != 2 || doPedido[0].ID != primeira.ID || doPedido[1].ID != segunda.ID || len(doPedido[1].Itens) != 1 || doPedido[1].Itens[0].Quantidade != 3 {
		t.Errorf("entregas do pedido = %+v (%v)", doPedido, err)
	}

	listar := func(filtro portspedido.EntregaFiltro) int {
		t.Helper()
		filtro.PaginacaoFiltro = todasAsPaginas
		entregas, err := c.entregas.Listar(c.ctx, filtro)
		if err != nil {
			t.Fatalf("Listar falhou: %v", err)
		}
		return len(entregas)
	}

	ontem := time.Now().UTC().AddDate(0, 0, -1)
	if total := listar(portspedido.EntregaFiltro{}); total != 3 {
		t.Errorf("todas = %d, esperado 3", total)
	}
	if total := listar(portspedido.EntregaFiltro{Empresa: &c.mia}); total != 2 {
		t.Errorf("da MIA = %d, esperado 2", total)
	}
	if total := listar(portspedido.EntregaFiltro{RepresentanteID: &representante.ID}); total != 1 {
		t.Errorf("do representante = %d, esperado 1", total)
	}
	if total := listar(portspedido.EntregaFiltro{EmpresaIDs: []uuid.UUID{c.red}}); total != 1 {
		t.Errorf("acesso à Red = %d, esperado 1", total)
	}
	if total := listar(portspedido.EntregaFiltro{Status: domainpedido.StatusEntregaSaiu}); total != 0 {
		t.Errorf("que saíram = %d, esperado 0", total)
	}
	if total := listar(portspedido.EntregaFiltro{PrevisaoAte: &ontem}); total != 0 {
		t.Errorf("com previsão até ontem = %d, esperado 0", total)
	}
	if ausente, err := c.entregas.Obter(c.ctx, uuid.New()); err != nil || ausente != nil {
		t.Errorf("entrega inexistente = %+v, %v", ausente, err)
	}
	if daRedEntrega.PedidoID != daRed.ID {
		t.Errorf("entrega da Red ligada a %v", daRedEntrega.PedidoID)
	}
}

func TestLocaisAtivosDaEmpresa(t *testing.T) {
	c := novoCenarioPedidos(t)
	inativo := c.novoLocal(t, c.mia, "Antigo depósito")
	if err := inativo.Desativar(false, time.Now().UTC()); err != nil {
		t.Fatalf("Desativar falhou: %v", err)
	}
	if _, err := c.locais.Atualizar(context.Background(), inativo); err != nil {
		t.Fatalf("Atualizar falhou: %v", err)
	}
	c.novoLocal(t, c.red, "Galpão da Red")

	locais, err := c.locais.AtivosDaEmpresa(c.ctx, c.mia)
	if err != nil || len(locais) != 2 || locais[0].ID != c.galpao.ID || locais[1].ID != c.loja.ID {
		t.Errorf("locais ativos da MIA = %+v (%v), esperado galpão e loja", locais, err)
	}
}
