package receber_test

import (
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/receber"
	"github.com/google/uuid"
)

func (c *cenario) novoPedido(empresaID uuid.UUID, modelo domainpedido.Modelo, status domainpedido.Status) *domainpedido.Pedido {
	pedido := &domainpedido.Pedido{
		ID:                 uuid.New(),
		EmpresaID:          empresaID,
		Numero:             57,
		ClienteID:          c.cliente,
		Modelo:             modelo,
		Condicao:           domainpedido.CondicaoDoModelo(modelo),
		Status:             status,
		ValorTotalCentavos: 100001,
		Versao:             3,
	}
	c.m.pedidos[pedido.ID] = pedido

	return pedido
}

func (c *cenario) entradaInput(usuarioID, pedidoID uuid.UUID) portsin.GerarTituloDeEntradaInput {
	return portsin.GerarTituloDeEntradaInput{
		UsuarioID:  usuarioID,
		PedidoID:   pedidoID,
		Vencimento: domainreceber.Hoje(time.Now().UTC()).AddDate(0, 0, 5),
	}
}

func TestGerarTituloDeEntradaDoPedido(t *testing.T) {
	c := novoCenario(t)
	pedido := c.novoPedido(c.mia, domainpedido.ModeloImportacao, domainpedido.StatusConfirmado)
	input := c.entradaInput(c.financeiro, pedido.ID)
	input.ContaBancariaID = &c.contaMia

	titulo, criado, err := c.entradas.GerarTituloDeEntrada(c.ctx, input)
	if err != nil {
		t.Fatalf("GerarTituloDeEntrada falhou: %v", err)
	}
	if !criado || titulo.Origem != domainreceber.OrigemPedido || *titulo.OrigemID != pedido.ID || titulo.Numero != "PED-57" {
		t.Fatalf("título = %+v, criado = %v", titulo, criado)
	}
	if titulo.ValorCentavos != 30000 || titulo.Status != domainreceber.StatusEmBanco || titulo.ClienteID != c.cliente || titulo.EmpresaID != c.mia {
		t.Fatalf("título = %+v", titulo)
	}

	repetido, criado, err := c.entradas.GerarTituloDeEntrada(c.ctx, c.entradaInput(c.financeiro, pedido.ID))
	if err != nil || criado || repetido.ID != titulo.ID || len(c.m.geracoes) != 1 {
		t.Fatalf("repetir devolve o existente sem gerar: %+v, %v, %v", repetido, criado, err)
	}
}

func TestGerarTituloDeEntradaValidaOPedido(t *testing.T) {
	c := novoCenario(t)
	prontaEntrega := c.novoPedido(c.mia, domainpedido.ModeloProntaEntrega, domainpedido.StatusConfirmado)
	proposta := c.novoPedido(c.mia, domainpedido.ModeloImportacao, domainpedido.StatusProposta)
	comEntrada := c.novoPedido(c.mia, domainpedido.ModeloImportacao, domainpedido.StatusParcialmenteEntregue)
	pagaEm := time.Now().UTC()
	comEntrada.EntradaPagaEm = &pagaEm
	daRed := c.novoPedido(c.red, domainpedido.ModeloImportacao, domainpedido.StatusConfirmado)
	valido := c.novoPedido(c.mia, domainpedido.ModeloImportacao, domainpedido.StatusConfirmado)

	_, _, err := c.entradas.GerarTituloDeEntrada(c.ctx, c.entradaInput(c.financeiro, prontaEntrega.ID))
	esperarErro(t, err, domain.ErrValidacao, "só o pedido de importação")
	_, _, err = c.entradas.GerarTituloDeEntrada(c.ctx, c.entradaInput(c.financeiro, proposta.ID))
	esperarErro(t, err, domain.ErrConflito, "confirmado e não entregue")
	_, _, err = c.entradas.GerarTituloDeEntrada(c.ctx, c.entradaInput(c.financeiro, comEntrada.ID))
	esperarErro(t, err, domain.ErrConflito, "já foi registrada")
	_, _, err = c.entradas.GerarTituloDeEntrada(c.ctx, c.entradaInput(c.financeiro, daRed.ID))
	esperarErro(t, err, domain.ErrPermissao, "sem acesso")
	_, _, err = c.entradas.GerarTituloDeEntrada(c.ctx, c.entradaInput(c.financeiro, uuid.New()))
	esperarErro(t, err, domain.ErrNotFound, "pedido não encontrado")
	_, _, err = c.entradas.GerarTituloDeEntrada(c.ctx, c.entradaInput(c.comercial, valido.ID))
	esperarErro(t, err, domain.ErrPermissao, "contas a receber")

	semVencimento := c.entradaInput(c.financeiro, valido.ID)
	semVencimento.Vencimento = time.Time{}
	_, _, err = c.entradas.GerarTituloDeEntrada(c.ctx, semVencimento)
	esperarErro(t, err, domain.ErrValidacao, "vencimento é obrigatório")

	contaDaRed := c.entradaInput(c.financeiro, valido.ID)
	contaDaRed.ContaBancariaID = &c.contaRed
	_, _, err = c.entradas.GerarTituloDeEntrada(c.ctx, contaDaRed)
	esperarErro(t, err, domain.ErrValidacao, "mesma empresa")

	if len(c.m.geracoes) != 0 {
		t.Fatalf("nenhuma geração deveria chegar ao repositório: %d", len(c.m.geracoes))
	}
}
