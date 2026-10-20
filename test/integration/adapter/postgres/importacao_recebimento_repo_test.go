package postgres_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/importacao"
	portspedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/pedido"
	"github.com/google/uuid"
)

var (
	_ portsimportacao.EstoqueRepository = (*postgres.EstoqueRepository)(nil)
	_ portsimportacao.LocalRepository   = (*postgres.LocalRepository)(nil)
)

type cenarioBancoRecebimento struct {
	*cenarioBancoPedidos
	importacoes  *postgres.ImportacaoRepository
	fornecedorID uuid.UUID
}

func novoCenarioRecebimento(t *testing.T) *cenarioBancoRecebimento {
	t.Helper()

	base := novoCenarioPedidos(t)
	identificacao := "91440600MA4UX"
	fornecedor, err := domainfornecedor.NovoFornecedor(domainfornecedor.PaisDe("CN"), domaincliente.Documento{}, domainfornecedor.DadosFornecedor{
		Nome: "Foshan Ceramics", MoedaPadrao: domainfornecedor.MoedaDolar, IdentificacaoEstrangeira: &identificacao,
	})
	if err != nil {
		t.Fatalf("fornecedor de teste inválido: %v", err)
	}
	if _, err := postgres.NewFornecedorRepository(base.banco).Criar(base.ctx, fornecedor); err != nil {
		t.Fatalf("criação do fornecedor falhou: %v", err)
	}

	return &cenarioBancoRecebimento{
		cenarioBancoPedidos: base,
		importacoes:         postgres.NewImportacaoRepository(base.banco),
		fornecedorID:        fornecedor.ID,
	}
}

func (c *cenarioBancoRecebimento) pedidoDeImportacao(t *testing.T) *domainpedido.Pedido {
	t.Helper()

	pedido := c.novoPedido(t, c.mia, c.dados(domainpedido.ModeloImportacao))
	versaoLida := pedido.Versao
	agora := time.Now().UTC()
	if err := pedido.Confirmar(agora); err != nil {
		t.Fatalf("Confirmar: %v", err)
	}
	anterior := domainpedido.StatusProposta
	c.gravar(t, &portspedido.Gravacao{
		Pedido: pedido, VersaoLida: versaoLida, StatusAnterior: anterior,
		Historicos: []*domainpedido.Historico{
			domainpedido.NovoHistorico(pedido.ID, c.usuarioID, domainpedido.AcaoConfirmado, &anterior, pedido.Status, "", agora),
		},
	})

	return c.obter(t, pedido.ID)
}

func (c *cenarioBancoRecebimento) importacaoDesembaracada(t *testing.T) *domainimportacao.Importacao {
	t.Helper()

	agora := time.Now().UTC()
	hoje := domainimportacao.Hoje(agora)
	importacao, err := domainimportacao.NovaImportacao(c.mia, c.usuarioID, domainimportacao.DadosImportacao{
		FornecedorID: c.fornecedorID,
		Moeda:        domainfornecedor.MoedaDolar,
		Itens: []domainimportacao.ItemInput{
			{ProdutoID: c.produto.ID, Quantidade: 12, ValorMoedaCentesimos: 120000},
			{ProdutoID: c.segundo.ID, Quantidade: 4, ValorMoedaCentesimos: 40000},
		},
	}, agora)
	if err != nil {
		t.Fatalf("NovaImportacao: %v", err)
	}
	c.gravarImportacao(t, &portsimportacao.Gravacao{Importacao: importacao, Novo: true, FornecedorTrocado: true})

	if err := importacao.Embarcar(domainimportacao.EmbarqueInput{EmbarcadaEm: hoje.AddDate(0, 0, -10), ChegadaPrevista: hoje}, agora); err != nil {
		t.Fatalf("Embarcar: %v", err)
	}
	c.gravarImportacao(t, &portsimportacao.Gravacao{Importacao: importacao, VersaoLida: 1, StatusAnterior: domainimportacao.StatusAberta})
	if err := importacao.Desembaracar(domainimportacao.DesembaracoInput{ChegadaEm: hoje, DesembaracadaEm: hoje, Declaracao: "26/1"}, agora); err != nil {
		t.Fatalf("Desembaracar: %v", err)
	}
	c.gravarImportacao(t, &portsimportacao.Gravacao{Importacao: importacao, VersaoLida: 2, StatusAnterior: domainimportacao.StatusEmbarcada})

	return importacao
}

func (c *cenarioBancoRecebimento) gravarImportacao(t *testing.T, gravacao *portsimportacao.Gravacao) {
	t.Helper()

	if err := c.importacoes.Gravar(c.ctx, gravacao); err != nil {
		t.Fatalf("gravar a importação falhou: %v", err)
	}
}

func (c *cenarioBancoRecebimento) alocar(t *testing.T, importacao *domainimportacao.Importacao, entradas ...domainimportacao.AlocacaoInput) error {
	t.Helper()

	ids := make([]uuid.UUID, 0, len(entradas))
	for _, entrada := range entradas {
		ids = append(ids, entrada.PedidoItemID)
	}
	pedidos := c.itensDePedido(t, importacao.ID, ids...)
	if err := importacao.DefinirAlocacoes(c.usuarioID, entradas, pedidos, time.Now().UTC()); err != nil {
		t.Fatalf("DefinirAlocacoes: %v", err)
	}

	return c.importacoes.Gravar(c.ctx, &portsimportacao.Gravacao{
		Importacao: importacao, VersaoLida: importacao.Versao, StatusAnterior: importacao.Status, SubstituirAlocacoes: true,
	})
}

func (c *cenarioBancoRecebimento) itensDePedido(t *testing.T, importacaoID uuid.UUID, ids ...uuid.UUID) map[uuid.UUID]*domainimportacao.PedidoItemAlocavel {
	t.Helper()

	itens, err := c.importacoes.ItensDePedido(c.ctx, ids, importacaoID)
	if err != nil {
		t.Fatalf("ItensDePedido: %v", err)
	}
	mapa := make(map[uuid.UUID]*domainimportacao.PedidoItemAlocavel, len(itens))
	for _, item := range itens {
		mapa[item.PedidoItemID] = item
	}

	return mapa
}

func (c *cenarioBancoRecebimento) gravacaoDoRecebimento(t *testing.T, importacao *domainimportacao.Importacao) *portsimportacao.Gravacao {
	t.Helper()

	ids := make([]uuid.UUID, 0)
	for _, alocacao := range importacao.Alocacoes {
		ids = append(ids, alocacao.PedidoItemID)
	}
	pedidos := c.itensDePedido(t, importacao.ID, ids...)

	agora := time.Now().UTC()
	versaoLida := importacao.Versao
	cambio := int64(5_000_000)
	if _, err := importacao.Receber(domainimportacao.RecebimentoInput{RecebidaEm: domainimportacao.Hoje(agora), LocalID: c.galpao.ID, CambioMicro: &cambio}, agora); err != nil {
		t.Fatalf("Receber: %v", err)
	}

	operacoes := make([]*domainestoque.OperacaoEstoque, 0)
	estoques := make(map[uuid.UUID]*domainestoque.EstoqueProduto)
	for _, item := range importacao.Itens {
		estoque, err := c.estoque.Carregar(c.ctx, item.ProdutoID, []uuid.UUID{c.galpao.ID})
		if err != nil {
			t.Fatalf("Carregar: %v", err)
		}
		custo := domainestoque.CustoDe(*item.CustoUnitarioCentavos)
		operacao, err := estoque.Movimentar(domainestoque.MovimentoLocalInput{
			Tipo: domainestoque.TipoMovimentoEntrada, Local: c.galpao, Quantidade: domainestoque.QuantidadeDe(*item.QuantidadeRecebida),
			Custo: &custo, UsuarioID: c.usuarioID, Agora: agora,
		})
		if err != nil {
			t.Fatalf("Movimentar: %v", err)
		}
		if err := estoque.Aplicar(operacao); err != nil {
			t.Fatalf("Aplicar: %v", err)
		}
		movimentoID := operacao.Movimento.ID
		item.MovimentoID = &movimentoID
		estoques[item.ProdutoID] = estoque
		operacoes = append(operacoes, operacao)
	}

	atendidos := make([]uuid.UUID, 0)
	for _, planejada := range importacao.PlanejarReservas(pedidos) {
		estoque := estoques[planejada.ProdutoID]
		pedidoItemID := planejada.Alocacao.PedidoItemID
		operacao, err := estoque.Reservar(domainestoque.ReservaInput{
			Local: c.galpao, Quantidade: domainestoque.QuantidadeDe(planejada.Quantidade),
			Origem: domainestoque.OrigemReservaPedidoItem, OrigemID: &pedidoItemID, UsuarioID: c.usuarioID, Agora: agora,
		})
		if err != nil {
			t.Fatalf("Reservar: %v", err)
		}
		if err := estoque.Aplicar(operacao); err != nil {
			t.Fatalf("Aplicar reserva: %v", err)
		}
		reservaID := operacao.ReservaCriada.ID
		planejada.Alocacao.ReservaID = &reservaID
		operacoes = append(operacoes, operacao)
		atendidos = append(atendidos, planejada.PedidoID)
	}

	historicos := make([]*domainpedido.Historico, 0, len(atendidos))
	for _, pedidoID := range atendidos {
		historicos = append(historicos, domainpedido.NovoHistorico(pedidoID, c.usuarioID, domainpedido.AcaoPendenciasReservadas, nil,
			domainpedido.StatusConfirmado, "importação", agora))
	}

	return &portsimportacao.Gravacao{
		Importacao: importacao, VersaoLida: versaoLida, StatusAnterior: domainimportacao.StatusDesembaracada, Recebimento: true,
		OperacoesEstoque: operacoes, PedidosAtendidos: atendidos[:min(len(atendidos), 1)], HistoricosDePedido: historicos[:min(len(historicos), 1)],
	}
}

func TestImportacaoAlocaPedidosEConfereORestanteNoBanco(t *testing.T) {
	c := novoCenarioRecebimento(t)
	pedido := c.pedidoDeImportacao(t)
	itemDoPedido := pedido.Itens[0]
	primeira := c.importacaoDesembaracada(t)

	if err := c.alocar(t, primeira, domainimportacao.AlocacaoInput{
		ImportacaoItemID: primeira.Itens[0].ID, PedidoItemID: itemDoPedido.ID, Quantidade: 6,
	}); err != nil {
		t.Fatalf("alocar: %v", err)
	}
	lida, err := c.importacoes.Obter(c.ctx, primeira.ID)
	if err != nil || len(lida.Alocacoes) != 1 || lida.Alocacoes[0].Quantidade != 6 || lida.Alocacoes[0].ReservaID != nil {
		t.Fatalf("alocações lidas = %+v (%v)", lida.Alocacoes, err)
	}

	segunda := c.importacaoDesembaracada(t)
	itens := c.itensDePedido(t, segunda.ID, itemDoPedido.ID)
	if info := itens[itemDoPedido.ID]; info.AlocadoEmOutras != 6 || info.Restante() != 10 || info.PedidoNumero != pedido.Numero ||
		info.Modelo != domainpedido.ModeloImportacao || info.StatusPedido != domainpedido.StatusConfirmado {
		t.Errorf("item do pedido visto pela segunda importação = %+v", info)
	}

	segunda.Alocacoes = []*domainimportacao.Alocacao{{
		ImportacaoItemID: segunda.Itens[0].ID, PedidoItemID: itemDoPedido.ID, Quantidade: 5, UsuarioID: c.usuarioID,
		CriadoEm: time.Now().UTC(), AtualizadoEm: time.Now().UTC(),
	}}
	if err := c.importacoes.Gravar(c.ctx, &portsimportacao.Gravacao{
		Importacao: segunda, VersaoLida: segunda.Versao, StatusAnterior: segunda.Status, SubstituirAlocacoes: true,
	}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("alocação acima do restante passou pelo banco: %v", err)
	}

	if err := c.alocar(t, primeira); err != nil {
		t.Fatalf("tirar as alocações: %v", err)
	}
	if lida, _ := c.importacoes.Obter(c.ctx, primeira.ID); len(lida.Alocacoes) != 0 {
		t.Errorf("alocações depois de limpar = %+v", lida.Alocacoes)
	}
}

func TestImportacaoRecebeComEntradaLoteEReservaNaMesmaTransacao(t *testing.T) {
	c := novoCenarioRecebimento(t)
	pedido := c.pedidoDeImportacao(t)
	importacao := c.importacaoDesembaracada(t)
	if err := c.alocar(t, importacao, domainimportacao.AlocacaoInput{
		ImportacaoItemID: importacao.Itens[0].ID, PedidoItemID: pedido.Itens[0].ID, Quantidade: 10,
	}); err != nil {
		t.Fatalf("alocar: %v", err)
	}
	importacao, _ = c.importacoes.Obter(c.ctx, importacao.ID)

	if err := c.importacoes.Gravar(c.ctx, c.gravacaoDoRecebimento(t, importacao)); err != nil {
		t.Fatalf("gravar recebimento: %v", err)
	}

	lida, err := c.importacoes.Obter(c.ctx, importacao.ID)
	if err != nil || lida.Status != domainimportacao.StatusRecebida || *lida.LocalRecebimentoID != c.galpao.ID || lida.Cambio.Micro() != 5_000_000 {
		t.Fatalf("importação recebida = %+v (%v)", lida, err)
	}
	porcelanato := lida.Itens[0]
	if *porcelanato.QuantidadeRecebida != 12 || *porcelanato.CustoUnitarioCentavos != 50000 || porcelanato.MovimentoID == nil {
		t.Errorf("item recebido = %+v", porcelanato)
	}
	if alocacao := lida.Alocacoes[0]; *alocacao.Reservada != 10 || alocacao.ReservaID == nil {
		t.Errorf("alocação depois do recebimento = %+v", alocacao)
	}

	estoque := c.estoqueDe(t, c.produto)
	if saldo := estoque.SaldoNoLocal(c.galpao.ID); saldo.Quantidade != 12 || saldo.Reservado != 10 {
		t.Errorf("saldo no galpão = %+v", saldo)
	}
	if len(estoque.Lotes) != 1 || estoque.Lotes[0].Custo == nil || estoque.Lotes[0].Custo.Centavos() != 50000 ||
		estoque.Lotes[0].MovimentoID != *porcelanato.MovimentoID {
		t.Errorf("lotes = %+v", estoque.Lotes)
	}

	reservas, err := c.reservas.AtivasPorOrigem(c.ctx, []uuid.UUID{pedido.Itens[0].ID})
	if err != nil || len(reservas) != 1 || reservas[0].Quantidade != 10 || reservas[0].ID != *lida.Alocacoes[0].ReservaID {
		t.Errorf("reservas do pedido = %+v (%v)", reservas, err)
	}
	historico, err := c.pedidos.ListarHistorico(c.ctx, pedido.ID)
	if err != nil || historico[len(historico)-1].Acao != domainpedido.AcaoPendenciasReservadas {
		t.Errorf("histórico do pedido = %+v (%v)", historico, err)
	}
}

func TestImportacaoRecebimentoDesfazTudoComPedidoForaDeAtendimento(t *testing.T) {
	c := novoCenarioRecebimento(t)
	pedido := c.pedidoDeImportacao(t)
	importacao := c.importacaoDesembaracada(t)
	if err := c.alocar(t, importacao, domainimportacao.AlocacaoInput{
		ImportacaoItemID: importacao.Itens[0].ID, PedidoItemID: pedido.Itens[0].ID, Quantidade: 10,
	}); err != nil {
		t.Fatalf("alocar: %v", err)
	}
	importacao, _ = c.importacoes.Obter(c.ctx, importacao.ID)
	gravacao := c.gravacaoDoRecebimento(t, importacao)

	if _, err := c.banco.Exec(`UPDATE pedido SET status = 'cancelado', cancelado_motivo = 'teste' WHERE id = $1`, pedido.ID); err != nil {
		t.Fatalf("cancelar pedido: %v", err)
	}
	if err := c.importacoes.Gravar(c.ctx, gravacao); !errors.Is(err, domain.ErrConflito) {
		t.Fatalf("recebimento com pedido cancelado: erro = %v", err)
	}

	lida, _ := c.importacoes.Obter(c.ctx, importacao.ID)
	if lida.Status != domainimportacao.StatusDesembaracada || lida.Itens[0].MovimentoID != nil {
		t.Errorf("importação depois da recusa = %+v", lida)
	}
	if saldo := c.estoqueDe(t, c.produto).SaldoNoLocal(c.galpao.ID); saldo.Quantidade != 0 {
		t.Errorf("estoque entrou mesmo com a recusa: %+v", saldo)
	}
}
