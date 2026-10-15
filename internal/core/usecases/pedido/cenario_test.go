package pedido_test

import (
	"context"
	"testing"
	"time"

	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/pedido"
	pedidousecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/pedido"
	"github.com/google/uuid"
)

type cenario struct {
	ctx      context.Context
	m        *memoria
	pedidos  portsin.PedidoUseCase
	entregas portsin.EntregaUseCase

	mia       uuid.UUID
	red       uuid.UUID
	inativa   uuid.UUID
	galpao    *domainestoque.Local
	loja      *domainestoque.Local
	galpaoRed *domainestoque.Local

	administrador      uuid.UUID
	comercial          uuid.UUID
	comercialRed       uuid.UUID
	financeiro         uuid.UUID
	operador           uuid.UUID
	vendedor           uuid.UUID
	representante      *domaincliente.Representante
	outroRepresentante *domaincliente.Representante

	cliente           *domaincliente.Cliente
	clienteInativo    *domaincliente.Cliente
	clienteDoVendedor *domaincliente.Cliente
	obra              *domaincliente.Obra
	cimento           *domainestoque.Produto
	porcelanato       *domainestoque.Produto
	produtoInativo    *domainestoque.Produto
}

func novoCenario(t *testing.T) *cenario {
	t.Helper()

	m := novaMemoria()
	c := &cenario{
		ctx: context.Background(),
		m:   m,
		pedidos: pedidousecases.NewPedidoUsecase(
			repoPedidos{m}, repoEntregas{m}, repoClientes{m}, repoObras{m}, repoRepresentantes{m}, repoProdutos{m},
			repoEmpresas{m}, repoEstoque{m}, repoReservas{m}, repoLocais{m}, repoUsuarios{m}, repoCargos{m},
		),
		entregas: pedidousecases.NewEntregaUsecase(
			repoPedidos{m}, repoEntregas{m}, repoRepresentantes{m}, repoEmpresas{m},
			repoEstoque{m}, repoReservas{m}, repoLocais{m}, repoUsuarios{m}, repoCargos{m},
		),
	}

	c.mia = c.novaEmpresa(true)
	c.red = c.novaEmpresa(true)
	c.inativa = c.novaEmpresa(false)
	c.galpao = c.novoLocal(t, c.mia, "Galpão")
	c.loja = c.novoLocal(t, c.mia, "Loja")
	c.galpaoRed = c.novoLocal(t, c.red, "Galpão da Red")

	acessoMia := domainempresa.AcessoRestrito([]uuid.UUID{c.mia, c.inativa})
	c.administrador = c.novoUsuario(true, false, false, domainempresa.AcessoTotal())
	c.comercial = c.novoUsuario(false, true, false, acessoMia)
	c.comercialRed = c.novoUsuario(false, true, false, domainempresa.AcessoRestrito([]uuid.UUID{c.red}))
	c.financeiro = c.novoUsuario(false, false, true, acessoMia)
	c.operador = c.novoUsuario(false, false, false, acessoMia)
	c.vendedor = c.novoUsuario(false, true, false, acessoMia)
	c.representante = c.novoRepresentante(&c.vendedor)
	c.outroRepresentante = c.novoRepresentante(nil)

	c.cliente = c.novoCliente(true, &c.outroRepresentante.ID)
	c.clienteInativo = c.novoCliente(false, nil)
	c.clienteDoVendedor = c.novoCliente(true, &c.representante.ID)
	c.obra = &domaincliente.Obra{ID: uuid.New(), ClienteID: c.cliente.ID, Nome: "Residencial Alfa", Ativo: true}
	m.obras[c.obra.ID] = c.obra

	c.cimento = c.novoProduto("Cimento CP II", true)
	c.porcelanato = c.novoProduto("Porcelanato 60x60", true)
	c.produtoInativo = c.novoProduto("Telha antiga", false)

	return c
}

func (c *cenario) novaEmpresa(ativa bool) uuid.UUID {
	empresa := &domainempresa.Empresa{ID: uuid.New(), NomeFantasia: "Empresa", Ativa: ativa}
	c.m.empresas[empresa.ID] = empresa

	return empresa.ID
}

func (c *cenario) novoLocal(t *testing.T, empresaID uuid.UUID, nome string) *domainestoque.Local {
	t.Helper()

	local, err := domainestoque.NovoLocal(empresaID, nome)
	if err != nil {
		t.Fatalf("NovoLocal falhou: %v", err)
	}
	c.m.locais[local.ID] = local

	return local
}

func (c *cenario) novoUsuario(administrador, comercial, financeiro bool, acesso domainempresa.Acesso) uuid.UUID {
	cargo := &domainusuarios.Cargo{
		ID: uuid.New(), Nome: "Cargo", Ativo: true,
		Administrador: administrador, Comercial: comercial, Financeiro: financeiro,
	}
	usuario := &domainusuarios.Usuario{ID: uuid.New(), Nome: "Usuário", CargoID: cargo.ID, Ativo: true}
	c.m.cargos[cargo.ID] = cargo
	c.m.usuarios[usuario.ID] = usuario
	c.m.acessos[usuario.ID] = acesso

	return usuario.ID
}

func (c *cenario) novoRepresentante(usuarioID *uuid.UUID) *domaincliente.Representante {
	representante := &domaincliente.Representante{ID: uuid.New(), Nome: "Representante", UsuarioID: usuarioID, Ativo: true}
	c.m.representantes[representante.ID] = representante

	return representante
}

func (c *cenario) novoCliente(ativo bool, representanteID *uuid.UUID) *domaincliente.Cliente {
	cliente := &domaincliente.Cliente{ID: uuid.New(), Nome: "Construtora Alfa", RepresentanteID: representanteID, Ativo: ativo}
	c.m.clientes[cliente.ID] = cliente

	return cliente
}

func (c *cenario) novoProduto(nome string, ativo bool) *domainestoque.Produto {
	produto := &domainestoque.Produto{ID: uuid.New(), Nome: nome, Ativo: ativo, VersaoEstoque: 1}
	c.m.produtos[produto.ID] = produto
	c.m.estoques[produto.ID] = &domainestoque.EstoqueProduto{
		Produto: copiar(produto),
		Saldos:  map[uuid.UUID]domainestoque.SaldoLocal{},
	}

	return produto
}

func (c *cenario) entrada(t *testing.T, produto *domainestoque.Produto, local *domainestoque.Local, quantidade int) {
	t.Helper()

	estoque := c.m.estoques[produto.ID]
	custo := domainestoque.CustoDe(1000)
	operacao, err := estoque.Movimentar(domainestoque.MovimentoLocalInput{
		Tipo:       domainestoque.TipoMovimentoEntrada,
		Local:      local,
		Quantidade: domainestoque.QuantidadeDe(quantidade),
		Custo:      &custo,
		UsuarioID:  c.administrador,
		Agora:      time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("entrada de domínio falhou: %v", err)
	}
	if err := estoque.Aplicar(operacao); err != nil {
		t.Fatalf("aplicar a entrada falhou: %v", err)
	}
}

func (c *cenario) saldo(produto *domainestoque.Produto, local *domainestoque.Local) domainestoque.SaldoLocal {
	return c.m.estoques[produto.ID].SaldoNoLocal(local.ID)
}

func (c *cenario) reservasDoItem(itemID uuid.UUID) map[uuid.UUID]int {
	porLocal := map[uuid.UUID]int{}
	for _, reserva := range c.m.reservas {
		if reserva.Ativa() && reserva.OrigemID != nil && *reserva.OrigemID == itemID {
			porLocal[reserva.LocalID] += reserva.Quantidade
		}
	}

	return porLocal
}

func (c *cenario) dados() portsin.DadosPedidoInput {
	return portsin.DadosPedidoInput{
		ClienteID: c.cliente.ID,
		Modelo:    "pronta_entrega",
		Itens: []portsin.ItemInput{
			{ProdutoID: c.cimento.ID, Quantidade: 10, PrecoUnitarioCentavos: 1500},
			{ProdutoID: c.porcelanato.ID, Quantidade: 4, PrecoUnitarioCentavos: 2500},
		},
	}
}

func (c *cenario) dadosDoVendedor() portsin.DadosPedidoInput {
	dados := c.dados()
	dados.ClienteID = c.clienteDoVendedor.ID

	return dados
}

func (c *cenario) criar(t *testing.T, usuarioID, empresaID uuid.UUID, dados portsin.DadosPedidoInput) *domainpedido.Pedido {
	t.Helper()

	pedido, err := c.pedidos.Criar(c.ctx, portsin.CriarPedidoInput{UsuarioID: usuarioID, EmpresaID: empresaID, Dados: dados})
	if err != nil {
		t.Fatalf("Criar falhou: %v", err)
	}

	return pedido
}

func (c *cenario) confirmado(t *testing.T, dados portsin.DadosPedidoInput) *domainpedido.Pedido {
	t.Helper()

	pedido := c.criar(t, c.comercial, c.mia, dados)
	confirmado, err := c.pedidos.Confirmar(c.ctx, portsin.TransicaoPedidoInput{UsuarioID: c.comercial, PedidoID: pedido.ID, Versao: pedido.Versao})
	if err != nil {
		t.Fatalf("Confirmar falhou: %v", err)
	}

	return confirmado
}

func (c *cenario) novaEntrega(t *testing.T, pedido *domainpedido.Pedido, local *domainestoque.Local, quantidade int) *domainpedido.Entrega {
	t.Helper()

	entrega, err := c.entregas.Criar(c.ctx, portsin.CriarEntregaInput{
		UsuarioID: c.comercial,
		PedidoID:  pedido.ID,
		FreteTipo: "fob",
		Itens:     []portsin.EntregaItemInput{{PedidoItemID: pedido.Itens[0].ID, LocalID: local.ID, Quantidade: quantidade}},
	})
	if err != nil {
		t.Fatalf("Criar entrega falhou: %v", err)
	}

	return entrega
}

func (c *cenario) pedido(id uuid.UUID) *domainpedido.Pedido {
	return c.m.pedidos[id]
}
