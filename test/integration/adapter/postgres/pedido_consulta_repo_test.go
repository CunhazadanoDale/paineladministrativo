package postgres_test

import (
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portspedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/pedido"
	"github.com/google/uuid"
)

func (c *cenarioBancoPedidos) cancelar(t *testing.T, pedido *domainpedido.Pedido, devolver *int64) {
	t.Helper()

	versao, status := pedido.Versao, pedido.Status
	if _, err := pedido.Cancelar(domainpedido.CancelamentoInput{Motivo: "cliente desistiu", ValorADevolverCentavos: devolver}, time.Now().UTC()); err != nil {
		t.Fatalf("Cancelar falhou: %v", err)
	}
	c.gravar(t, &portspedido.Gravacao{Pedido: pedido, VersaoLida: versao, StatusAnterior: status})
}

func TestPendenciasSomamProgramadoReservadoEDisponivel(t *testing.T) {
	c := novoCenarioPedidos(t)
	consultas := postgres.NewPedidoConsultaRepository(c.banco)
	c.entrada(t, c.produto, c.galpao, 6)
	pedido := c.novoPedido(t, c.mia, c.dados(domainpedido.ModeloProntaEntrega))
	c.confirmarReservando(t, pedido)
	c.novaEntrega(t, c.obter(t, pedido.ID), c.galpao, 2)
	c.entrada(t, c.produto, c.loja, 3)

	c.novoPedido(t, c.mia, c.dados(domainpedido.ModeloProntaEntrega))
	daRed := c.novoPedido(t, c.red, c.dados(domainpedido.ModeloProntaEntrega))
	if err := daRed.Confirmar(time.Now().UTC()); err != nil {
		t.Fatalf("Confirmar falhou: %v", err)
	}
	c.gravar(t, &portspedido.Gravacao{Pedido: daRed, VersaoLida: 1, StatusAnterior: domainpedido.StatusProposta})

	pendencias, err := consultas.Pendencias(c.ctx, portspedido.PendenciaFiltro{
		PaginacaoFiltro: todasAsPaginas,
		EscopoFiltro:    portspedido.EscopoFiltro{EmpresaIDs: []uuid.UUID{c.mia}},
	})
	if err != nil {
		t.Fatalf("Pendencias falhou: %v", err)
	}
	if len(pendencias) != 2 {
		t.Fatalf("pendências = %d, esperado 2 (os itens do pedido confirmado da MIA)", len(pendencias))
	}

	cimento, porcelanato := pendencias[0], pendencias[1]
	if cimento.ItemID != pedido.Itens[0].ID {
		cimento, porcelanato = porcelanato, cimento
	}
	if cimento.Vendido != 10 || cimento.Entregue != 0 || cimento.Programado != 2 || cimento.Reservado != 6 ||
		cimento.DisponivelNaEmpresa != 3 || cimento.FaltaCobrir() != 4 || cimento.PodeCobrirAgora() != 3 || cimento.ProdutoNome != "Cimento CP II" {
		t.Errorf("pendência do cimento = %+v", cimento)
	}
	if porcelanato.Vendido != 4 || porcelanato.Reservado != 0 || porcelanato.DisponivelNaEmpresa != 0 || porcelanato.Numero != pedido.Numero {
		t.Errorf("pendência do porcelanato = %+v", porcelanato)
	}

	doProduto, err := consultas.Pendencias(c.ctx, portspedido.PendenciaFiltro{PaginacaoFiltro: todasAsPaginas, ProdutoID: &c.segundo.ID})
	if err != nil || len(doProduto) != 2 {
		t.Errorf("pendências do porcelanato nas duas empresas = %d (%v), esperado 2", len(doProduto), err)
	}
	semAcesso, err := consultas.Pendencias(c.ctx, portspedido.PendenciaFiltro{
		PaginacaoFiltro: todasAsPaginas, EscopoFiltro: portspedido.EscopoFiltro{EmpresaIDs: []uuid.UUID{}},
	})
	if err != nil || len(semAcesso) != 0 {
		t.Errorf("pendências sem acesso = %d (%v), esperado 0", len(semAcesso), err)
	}
}

func TestPedidosPorPrazo(t *testing.T) {
	c := novoCenarioPedidos(t)
	consultas := postgres.NewPedidoConsultaRepository(c.banco)
	hoje := domainpedido.Hoje(time.Now())

	comPrazo := func(dias int, confirmar bool) *domainpedido.Pedido {
		dados := c.dados(domainpedido.ModeloProntaEntrega)
		prazo := hoje.AddDate(0, 0, dias)
		dados.PrazoLimite = &prazo
		pedido := c.novoPedido(t, c.mia, dados)
		if confirmar {
			c.confirmarReservando(t, pedido)
		}
		return pedido
	}
	vencido := comPrazo(-1, true)
	proximo := comPrazo(5, true)
	distante := comPrazo(60, true)
	comPrazo(2, false)
	cancelado := comPrazo(3, true)
	c.cancelar(t, c.obter(t, cancelado.ID), nil)

	listar := func(situacao domainpedido.SituacaoPrazo) []uuid.UUID {
		t.Helper()
		janela, err := domainpedido.NovaJanelaDePrazo(situacao, nil, time.Now().UTC())
		if err != nil {
			t.Fatalf("NovaJanelaDePrazo falhou: %v", err)
		}
		pedidos, err := consultas.PorPrazo(c.ctx, portspedido.PrazoFiltro{PaginacaoFiltro: todasAsPaginas, Janela: janela})
		if err != nil {
			t.Fatalf("PorPrazo falhou: %v", err)
		}
		ids := make([]uuid.UUID, 0, len(pedidos))
		for _, pedido := range pedidos {
			ids = append(ids, pedido.ID)
		}
		return ids
	}

	if ids := listar(domainpedido.SituacaoPrazoVencido); len(ids) != 1 || ids[0] != vencido.ID {
		t.Errorf("vencidos = %v, esperado só %v", ids, vencido.ID)
	}
	if ids := listar(domainpedido.SituacaoPrazoAVencer); len(ids) != 1 || ids[0] != proximo.ID {
		t.Errorf("a vencer em 30 dias = %v, esperado só %v", ids, proximo.ID)
	}
	if ids := listar(domainpedido.SituacaoPrazoTodos); len(ids) != 2 || ids[0] != vencido.ID || ids[1] != proximo.ID {
		t.Errorf("vencidos e a vencer = %v, esperado %v e %v", ids, vencido.ID, proximo.ID)
	}
	if distante.PrazoLimite == nil {
		t.Error("pedido distante sem prazo")
	}
}

func TestCancelamentosComDevolucao(t *testing.T) {
	c := novoCenarioPedidos(t)
	consultas := postgres.NewPedidoConsultaRepository(c.banco)

	semEntrada := c.novoPedido(t, c.mia, c.dados(domainpedido.ModeloProntaEntrega))
	c.cancelar(t, semEntrada, nil)

	comEntrada := c.novoPedido(t, c.mia, c.dados(domainpedido.ModeloImportacao))
	agora := time.Now().UTC()
	versao := comEntrada.Versao
	if err := comEntrada.Confirmar(agora); err != nil {
		t.Fatalf("Confirmar falhou: %v", err)
	}
	if err := comEntrada.RegistrarEntrada(domainpedido.Hoje(agora), 7500, agora); err != nil {
		t.Fatalf("RegistrarEntrada falhou: %v", err)
	}
	c.gravar(t, &portspedido.Gravacao{Pedido: comEntrada, VersaoLida: versao, StatusAnterior: domainpedido.StatusProposta})
	devolver := int64(5000)
	c.cancelar(t, comEntrada, &devolver)
	c.novoPedido(t, c.mia, c.dados(domainpedido.ModeloProntaEntrega))

	contar := func(filtro portspedido.CancelamentoFiltro) int {
		t.Helper()
		filtro.PaginacaoFiltro = todasAsPaginas
		pedidos, err := consultas.Cancelamentos(c.ctx, filtro)
		if err != nil {
			t.Fatalf("Cancelamentos falhou: %v", err)
		}
		return len(pedidos)
	}
	hoje := domainpedido.Hoje(agora)
	amanha := hoje.AddDate(0, 0, 1)

	if total := contar(portspedido.CancelamentoFiltro{}); total != 2 {
		t.Errorf("cancelados = %d, esperado 2", total)
	}
	if total := contar(portspedido.CancelamentoFiltro{ComDevolucao: true}); total != 1 {
		t.Errorf("com devolução = %d, esperado 1", total)
	}
	if total := contar(portspedido.CancelamentoFiltro{De: &hoje, Ate: &hoje}); total != 2 {
		t.Errorf("cancelados hoje = %d, esperado 2", total)
	}
	if total := contar(portspedido.CancelamentoFiltro{De: &amanha}); total != 0 {
		t.Errorf("cancelados a partir de amanhã = %d, esperado 0", total)
	}
	if total := contar(portspedido.CancelamentoFiltro{EscopoFiltro: portspedido.EscopoFiltro{Empresa: &c.red}}); total != 0 {
		t.Errorf("cancelados da Red = %d, esperado 0", total)
	}
}
