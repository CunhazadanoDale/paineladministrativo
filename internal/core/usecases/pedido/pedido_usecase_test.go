package pedido_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/pedido"
	"github.com/google/uuid"
)

func TestCriarNumeraPorEmpresaEUsaORepresentanteDoCliente(t *testing.T) {
	c := novoCenario(t)

	primeiro := c.criar(t, c.comercial, c.mia, c.dados())
	segundo := c.criar(t, c.administrador, c.mia, c.dados())
	daRed := c.criar(t, c.comercialRed, c.red, c.dados())

	if primeiro.Numero != 1 || segundo.Numero != 2 || daRed.Numero != 1 {
		t.Errorf("números = %d, %d e %d; esperado 1, 2 e 1", primeiro.Numero, segundo.Numero, daRed.Numero)
	}
	if primeiro.RepresentanteID == nil || *primeiro.RepresentanteID != c.outroRepresentante.ID {
		t.Errorf("representante = %v, esperado o do cliente", primeiro.RepresentanteID)
	}
	if primeiro.Status != domainpedido.StatusProposta || primeiro.UsuarioID != c.comercial {
		t.Errorf("pedido criado com status %q e usuário %v", primeiro.Status, primeiro.UsuarioID)
	}

	historico, err := c.pedidos.ListarHistorico(c.ctx, primeiro.ID, c.comercial)
	if err != nil || len(historico) != 1 || historico[0].Acao != domainpedido.AcaoCriado {
		t.Errorf("histórico = %+v (%v)", historico, err)
	}
}

func TestCriarValida(t *testing.T) {
	c := novoCenario(t)
	obraDeOutro := uuid.New()
	inativa := c.obra.ID

	casos := []struct {
		nome      string
		usuario   uuid.UUID
		empresa   uuid.UUID
		alterar   func(*portsin.DadosPedidoInput)
		sentinela error
	}{
		{"operador sem perfil comercial", c.operador, c.mia, nil, domain.ErrPermissao},
		{"financeiro sem perfil comercial", c.financeiro, c.mia, nil, domain.ErrPermissao},
		{"empresa fora do acesso", c.comercial, c.red, nil, domain.ErrPermissao},
		{"empresa inativa", c.comercial, c.inativa, nil, domain.ErrValidacao},
		{"sem empresa", c.comercial, uuid.Nil, nil, domain.ErrValidacao},
		{"cliente inativo", c.comercial, c.mia, func(d *portsin.DadosPedidoInput) { d.ClienteID = c.clienteInativo.ID }, domain.ErrValidacao},
		{"cliente inexistente", c.comercial, c.mia, func(d *portsin.DadosPedidoInput) { d.ClienteID = uuid.New() }, domain.ErrValidacao},
		{"obra de outro cliente", c.comercial, c.mia, func(d *portsin.DadosPedidoInput) { d.ObraID = &obraDeOutro }, domain.ErrValidacao},
		{"produto inativo", c.comercial, c.mia, func(d *portsin.DadosPedidoInput) { d.Itens[0].ProdutoID = c.produtoInativo.ID }, domain.ErrValidacao},
		{"produto inexistente", c.comercial, c.mia, func(d *portsin.DadosPedidoInput) { d.Itens[0].ProdutoID = uuid.New() }, domain.ErrValidacao},
		{"modelo inválido", c.comercial, c.mia, func(d *portsin.DadosPedidoInput) { d.Modelo = "consignado" }, domain.ErrValidacao},
		{"obra inativa", c.comercial, c.mia, func(d *portsin.DadosPedidoInput) {
			c.obra.Ativo = false
			d.ObraID = &inativa
		}, domain.ErrValidacao},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			dados := c.dados()
			if caso.alterar != nil {
				caso.alterar(&dados)
			}

			_, err := c.pedidos.Criar(c.ctx, portsin.CriarPedidoInput{UsuarioID: caso.usuario, EmpresaID: caso.empresa, Dados: dados})
			if !errors.Is(err, caso.sentinela) {
				t.Errorf("erro %v, esperado %v", err, caso.sentinela)
			}
		})
	}
	if c.m.gravacoes != 0 {
		t.Errorf("criação recusada gravou %d vezes", c.m.gravacoes)
	}
}

func TestRepresentanteComLoginSoVeEMexeNosProprios(t *testing.T) {
	c := novoCenario(t)

	proprio := c.criar(t, c.vendedor, c.mia, c.dadosDoVendedor())
	if proprio.RepresentanteID == nil || *proprio.RepresentanteID != c.representante.ID {
		t.Fatalf("pedido do vendedor com representante %v, esperado ele mesmo", proprio.RepresentanteID)
	}

	if _, err := c.pedidos.Criar(c.ctx, portsin.CriarPedidoInput{UsuarioID: c.vendedor, EmpresaID: c.mia, Dados: c.dados()}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("vendedor com cliente de outro representante = %v, esperado permissão", err)
	}
	if _, err := c.pedidos.Alterar(c.ctx, portsin.AlterarPedidoInput{UsuarioID: c.vendedor, PedidoID: proprio.ID, Versao: 1, Dados: c.dados()}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("vendedor trocando para cliente de outro representante = %v, esperado permissão", err)
	}

	outroDados := c.dadosDoVendedor()
	outroDados.RepresentanteID = &c.outroRepresentante.ID
	if _, err := c.pedidos.Criar(c.ctx, portsin.CriarPedidoInput{UsuarioID: c.vendedor, EmpresaID: c.mia, Dados: outroDados}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("vendedor em nome de outro = %v, esperado permissão", err)
	}

	alheio := c.criar(t, c.comercial, c.mia, c.dados())
	if _, err := c.pedidos.Obter(c.ctx, alheio.ID, c.vendedor); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("vendedor vendo pedido alheio = %v, esperado permissão", err)
	}
	if _, err := c.pedidos.ListarHistorico(c.ctx, alheio.ID, c.vendedor); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("vendedor vendo histórico alheio = %v, esperado permissão", err)
	}

	lista, err := c.pedidos.Listar(c.ctx, portsin.ListarPedidosInput{UsuarioID: c.vendedor})
	if err != nil || len(lista) != 1 || lista[0].ID != proprio.ID {
		t.Errorf("lista do vendedor = %d pedidos (%v), esperado só o próprio", len(lista), err)
	}
	if _, err := c.pedidos.Listar(c.ctx, portsin.ListarPedidosInput{UsuarioID: c.vendedor, RepresentanteID: &c.outroRepresentante.ID}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("vendedor filtrando outro representante = %v, esperado permissão", err)
	}

	if _, err := c.pedidos.Cancelar(c.ctx, portsin.CancelarPedidoInput{UsuarioID: c.vendedor, PedidoID: proprio.ID, Versao: 1, Motivo: "cliente desistiu"}); err != nil {
		t.Errorf("vendedor cancelando a própria proposta = %v", err)
	}

	confirmado := c.criar(t, c.vendedor, c.mia, c.dadosDoVendedor())
	if _, err := c.pedidos.Confirmar(c.ctx, portsin.TransicaoPedidoInput{UsuarioID: c.vendedor, PedidoID: confirmado.ID, Versao: 1}); err != nil {
		t.Fatalf("Confirmar falhou: %v", err)
	}
	if _, err := c.pedidos.Cancelar(c.ctx, portsin.CancelarPedidoInput{UsuarioID: c.vendedor, PedidoID: confirmado.ID, Versao: 2, Motivo: "x"}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("vendedor cancelando pedido confirmado = %v, esperado permissão", err)
	}
}

func TestIsolamentoEntreEmpresas(t *testing.T) {
	c := novoCenario(t)
	daMia := c.criar(t, c.comercial, c.mia, c.dados())
	daRed := c.criar(t, c.comercialRed, c.red, c.dados())

	listaMia, err := c.pedidos.Listar(c.ctx, portsin.ListarPedidosInput{UsuarioID: c.comercial})
	if err != nil || len(listaMia) != 1 || listaMia[0].ID != daMia.ID {
		t.Errorf("lista da MIA = %d (%v)", len(listaMia), err)
	}
	listaRed, err := c.pedidos.Listar(c.ctx, portsin.ListarPedidosInput{UsuarioID: c.comercialRed})
	if err != nil || len(listaRed) != 1 || listaRed[0].ID != daRed.ID {
		t.Errorf("lista da Red = %d (%v)", len(listaRed), err)
	}
	todos, err := c.pedidos.Listar(c.ctx, portsin.ListarPedidosInput{UsuarioID: c.administrador})
	if err != nil || len(todos) != 2 {
		t.Errorf("lista do administrador = %d (%v), esperado 2", len(todos), err)
	}

	if _, err := c.pedidos.Listar(c.ctx, portsin.ListarPedidosInput{UsuarioID: c.comercial, Empresa: &c.red}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("filtrar empresa sem acesso = %v, esperado permissão", err)
	}
	if _, err := c.pedidos.Obter(c.ctx, daRed.ID, c.comercial); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("ver pedido da outra empresa = %v, esperado permissão", err)
	}
	if _, err := c.pedidos.Alterar(c.ctx, portsin.AlterarPedidoInput{UsuarioID: c.comercial, PedidoID: daRed.ID, Versao: 1, Dados: c.dados()}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("alterar pedido da outra empresa = %v, esperado permissão", err)
	}
	if _, err := c.pedidos.Confirmar(c.ctx, portsin.TransicaoPedidoInput{UsuarioID: c.comercial, PedidoID: daRed.ID, Versao: 1}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("confirmar pedido da outra empresa = %v, esperado permissão", err)
	}
	if _, err := c.pedidos.Obter(c.ctx, uuid.New(), c.comercial); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("pedido inexistente = %v, esperado não encontrado", err)
	}
}

func TestAlterarExigeVersaoEProposta(t *testing.T) {
	c := novoCenario(t)
	pedido := c.criar(t, c.comercial, c.mia, c.dados())
	c.m.produtos[c.porcelanato.ID].Ativo = false

	dados := c.dados()
	dados.Itens[0].Quantidade = 20
	alterado, err := c.pedidos.Alterar(c.ctx, portsin.AlterarPedidoInput{UsuarioID: c.comercial, PedidoID: pedido.ID, Versao: 1, Dados: dados})
	if err != nil {
		t.Fatalf("Alterar com produto já no pedido inativado falhou: %v", err)
	}
	if alterado.Versao != 2 || alterado.ValorTotalCentavos != 40000 {
		t.Errorf("alterado = versão %d, total %d", alterado.Versao, alterado.ValorTotalCentavos)
	}

	if _, err := c.pedidos.Alterar(c.ctx, portsin.AlterarPedidoInput{UsuarioID: c.comercial, PedidoID: pedido.ID, Versao: 1, Dados: dados}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("versão velha = %v, esperado conflito", err)
	}

	if _, err := c.pedidos.Confirmar(c.ctx, portsin.TransicaoPedidoInput{UsuarioID: c.comercial, PedidoID: pedido.ID, Versao: 2}); err != nil {
		t.Fatalf("Confirmar falhou: %v", err)
	}
	if _, err := c.pedidos.Alterar(c.ctx, portsin.AlterarPedidoInput{UsuarioID: c.comercial, PedidoID: pedido.ID, Versao: 3, Dados: dados}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("alterar confirmado = %v, esperado conflito", err)
	}
}

func TestConfirmarReservaODisponivelComecandoPeloMaiorLocal(t *testing.T) {
	c := novoCenario(t)
	c.entrada(t, c.cimento, c.galpao, 3)
	c.entrada(t, c.cimento, c.loja, 6)
	c.entrada(t, c.cimento, c.galpaoRed, 50)

	pedido := c.confirmado(t, c.dados())

	if pedido.Status != domainpedido.StatusConfirmado || pedido.Versao != 2 {
		t.Errorf("pedido = %q, versão %d", pedido.Status, pedido.Versao)
	}
	cimento := c.reservasDoItem(pedido.Itens[0].ID)
	if cimento[c.loja.ID] != 6 || cimento[c.galpao.ID] != 3 || cimento[c.galpaoRed.ID] != 0 {
		t.Errorf("reservas do cimento = %v, esperado 6 na loja e 3 no galpão", cimento)
	}
	if porcelanato := c.reservasDoItem(pedido.Itens[1].ID); len(porcelanato) != 0 {
		t.Errorf("porcelanato sem estoque reservou %v", porcelanato)
	}
	if saldo := c.saldo(c.cimento, c.loja); saldo.Reservado != 6 || saldo.Disponivel() != 0 {
		t.Errorf("loja = %+v, esperado 6 reservados", saldo)
	}

	_, reservas, err := c.pedidos.ObterComReservas(c.ctx, pedido.ID, c.comercial)
	if err != nil || len(reservas) != 2 {
		t.Errorf("reservas do pedido = %d (%v), esperado 2", len(reservas), err)
	}

	historico, _ := c.pedidos.ListarHistorico(c.ctx, pedido.ID, c.comercial)
	if len(historico) != 2 || historico[1].Acao != domainpedido.AcaoConfirmado || historico[1].Detalhe == "" {
		t.Errorf("histórico = %+v", historico)
	}
}

func TestReservarPendenciasCobreSoOQueFalta(t *testing.T) {
	c := novoCenario(t)
	c.entrada(t, c.cimento, c.galpao, 4)
	pedido := c.confirmado(t, c.dados())

	semNovidade, err := c.pedidos.ReservarPendencias(c.ctx, portsin.TransicaoPedidoInput{UsuarioID: c.comercial, PedidoID: pedido.ID, Versao: 2})
	if err != nil || semNovidade.Versao != 2 {
		t.Fatalf("sem estoque novo = versão %d (%v), esperado nada gravado", semNovidade.Versao, err)
	}

	c.entrada(t, c.cimento, c.galpao, 20)
	c.entrada(t, c.porcelanato, c.loja, 1)
	atualizado, err := c.pedidos.ReservarPendencias(c.ctx, portsin.TransicaoPedidoInput{UsuarioID: c.comercial, PedidoID: pedido.ID, Versao: 2})
	if err != nil {
		t.Fatalf("ReservarPendencias falhou: %v", err)
	}

	if atualizado.Versao != 3 {
		t.Errorf("versão = %d, esperado 3", atualizado.Versao)
	}
	if cimento := c.reservasDoItem(pedido.Itens[0].ID); cimento[c.galpao.ID] != 10 {
		t.Errorf("reservado de cimento = %v, esperado 10 no galpão", cimento)
	}
	if porcelanato := c.reservasDoItem(pedido.Itens[1].ID); porcelanato[c.loja.ID] != 1 {
		t.Errorf("reservado de porcelanato = %v, esperado 1 na loja", porcelanato)
	}
	if saldo := c.saldo(c.cimento, c.galpao); saldo.Disponivel() != 14 {
		t.Errorf("disponível no galpão = %d, esperado 14", saldo.Disponivel())
	}

	proposta := c.criar(t, c.comercial, c.mia, c.dados())
	if _, err := c.pedidos.ReservarPendencias(c.ctx, portsin.TransicaoPedidoInput{UsuarioID: c.comercial, PedidoID: proposta.ID, Versao: 1}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("pendências de proposta = %v, esperado conflito", err)
	}
}

func TestRegistrarEntradaEhDoFinanceiro(t *testing.T) {
	c := novoCenario(t)
	dados := c.dados()
	dados.Modelo = "importacao"
	pedido := c.confirmado(t, dados)
	ontem := time.Now().UTC().AddDate(0, 0, -1)

	if _, err := c.pedidos.RegistrarEntrada(c.ctx, portsin.RegistrarEntradaInput{
		UsuarioID: c.comercial, PedidoID: pedido.ID, Versao: 2, PagaEm: ontem, ValorCentavos: 7500,
	}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("comercial registrando entrada = %v, esperado permissão", err)
	}

	registrado, err := c.pedidos.RegistrarEntrada(c.ctx, portsin.RegistrarEntradaInput{
		UsuarioID: c.financeiro, PedidoID: pedido.ID, Versao: 2, PagaEm: ontem, ValorCentavos: 7500,
	})
	if err != nil {
		t.Fatalf("RegistrarEntrada falhou: %v", err)
	}
	esperado := domainpedido.DataDe(ontem).AddDate(0, 0, 120)
	if registrado.PrazoLimite == nil || !registrado.PrazoLimite.Equal(esperado) || registrado.Versao != 3 {
		t.Errorf("prazo-limite = %v, versão %d", registrado.PrazoLimite, registrado.Versao)
	}
}

func TestCancelarLiberaReservasECancelaEntregasPendentes(t *testing.T) {
	c := novoCenario(t)
	c.entrada(t, c.cimento, c.galpao, 10)
	c.entrada(t, c.porcelanato, c.galpao, 4)
	pedido := c.confirmado(t, c.dados())
	entrega := c.novaEntrega(t, pedido, c.galpao, 3)

	cancelado, err := c.pedidos.Cancelar(c.ctx, portsin.CancelarPedidoInput{
		UsuarioID: c.comercial, PedidoID: pedido.ID, Versao: c.pedido(pedido.ID).Versao, Motivo: "obra embargada",
	})
	if err != nil {
		t.Fatalf("Cancelar falhou: %v", err)
	}

	if cancelado.Status != domainpedido.StatusCancelado || cancelado.ValorADevolverCentavos != nil {
		t.Errorf("cancelado = %q, devolução %v", cancelado.Status, cancelado.ValorADevolverCentavos)
	}
	if reservas := c.reservasDoItem(pedido.Itens[0].ID); len(reservas) != 0 {
		t.Errorf("reservas ativas depois do cancelamento = %v", reservas)
	}
	if saldo := c.saldo(c.cimento, c.galpao); saldo.Reservado != 0 || saldo.Quantidade != 10 {
		t.Errorf("galpão = %+v, esperado 10 livres", saldo)
	}
	if c.m.entregas[entrega.ID].Status != domainpedido.StatusEntregaCancelada {
		t.Errorf("entrega pendente ficou %q", c.m.entregas[entrega.ID].Status)
	}

	historico, _ := c.pedidos.ListarHistorico(c.ctx, pedido.ID, c.comercial)
	acoes := make([]domainpedido.Acao, 0, len(historico))
	for _, item := range historico {
		acoes = append(acoes, item.Acao)
	}
	if len(acoes) != 5 || acoes[3] != domainpedido.AcaoCancelado || acoes[4] != domainpedido.AcaoEntregaCancelada {
		t.Errorf("ações do histórico = %v", acoes)
	}
}

func TestCancelarComEntradaDevolveAEntradaPorPadrao(t *testing.T) {
	c := novoCenario(t)
	dados := c.dados()
	dados.Modelo = "importacao"
	pedido := c.confirmado(t, dados)
	if _, err := c.pedidos.RegistrarEntrada(c.ctx, portsin.RegistrarEntradaInput{
		UsuarioID: c.financeiro, PedidoID: pedido.ID, Versao: 2, PagaEm: domainpedido.Hoje(time.Now()), ValorCentavos: 7500,
	}); err != nil {
		t.Fatalf("RegistrarEntrada falhou: %v", err)
	}

	cancelado, err := c.pedidos.Cancelar(c.ctx, portsin.CancelarPedidoInput{UsuarioID: c.comercial, PedidoID: pedido.ID, Versao: 3, Motivo: "atraso"})
	if err != nil {
		t.Fatalf("Cancelar falhou: %v", err)
	}
	if cancelado.ValorADevolverCentavos == nil || *cancelado.ValorADevolverCentavos != 7500 {
		t.Errorf("valor a devolver = %v, esperado 7500", cancelado.ValorADevolverCentavos)
	}
}

func TestListarValidaFiltros(t *testing.T) {
	c := novoCenario(t)
	c.criar(t, c.comercial, c.mia, c.dados())

	if _, err := c.pedidos.Listar(c.ctx, portsin.ListarPedidosInput{UsuarioID: c.comercial, Status: "aberto"}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("status inválido = %v, esperado erro de validação", err)
	}
	if _, err := c.pedidos.Listar(c.ctx, portsin.ListarPedidosInput{UsuarioID: c.comercial, Modelo: "x"}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("modelo inválido = %v, esperado erro de validação", err)
	}
	propostas, err := c.pedidos.Listar(c.ctx, portsin.ListarPedidosInput{UsuarioID: c.operador, Status: "proposta"})
	if err != nil || len(propostas) != 1 {
		t.Errorf("operador lendo propostas = %d (%v), esperado 1", len(propostas), err)
	}
}
