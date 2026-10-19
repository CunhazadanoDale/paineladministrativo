package pedido_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/pedido"
)

func esperarErroDaProposta(t *testing.T, err error, sentinela error, trecho string) {
	t.Helper()

	if !errors.Is(err, sentinela) || !strings.Contains(err.Error(), trecho) {
		t.Fatalf("erro = %v; esperado %v com %q", err, sentinela, trecho)
	}
}

func TestPropostaGravaPrecoDeTabelaCondicoesEFrete(t *testing.T) {
	c := novoCenario(t)
	c.m.precosTabela[c.cimento.ID.String()+"pronta_entrega"] = 1800
	entrada, prazo, frete := 2000, 45, int64(9000)
	validade := domainpedido.Hoje(time.Now().UTC()).AddDate(0, 0, 7)
	dados := c.dados()
	dados.EntradaBP, dados.PrazoBoletoDias = &entrada, &prazo
	dados.FreteTipo, dados.FreteEstimadoCentavos = "CIF", &frete
	dados.Validade = &validade

	pedido := c.criar(t, c.comercial, c.mia, dados)
	if pedido.Revisao != 1 || !pedido.Validade.Equal(validade) || pedido.Frete == nil || *pedido.Frete.Centavos() != 9000 {
		t.Fatalf("proposta = revisão %d, validade %v, frete %+v", pedido.Revisao, pedido.Validade, pedido.Frete)
	}
	if pedido.Condicao.Entrada().PontosBase() != 2000 || *pedido.Condicao.PrazoBoletoDias() != 45 {
		t.Fatalf("condição = %+v", pedido.Condicao)
	}
	if pedido.Itens[0].PrecoTabela == nil || pedido.Itens[0].PrecoTabela.Centavos() != 1800 || pedido.Itens[1].PrecoTabela != nil {
		t.Fatalf("preços de tabela = %+v / %+v", pedido.Itens[0].PrecoTabela, pedido.Itens[1].PrecoTabela)
	}

	c.m.precosTabela[c.cimento.ID.String()+"pronta_entrega"] = 1900
	alterado, err := c.pedidos.Alterar(c.ctx, portsin.AlterarPedidoInput{UsuarioID: c.comercial, PedidoID: pedido.ID, Versao: pedido.Versao, Dados: c.dados()})
	if err != nil {
		t.Fatalf("Alterar: %v", err)
	}
	if alterado.Revisao != 2 || alterado.Itens[0].PrecoTabela.Centavos() != 1900 || alterado.Frete != nil ||
		alterado.Condicao.Entrada().PontosBase() != 0 || *alterado.Condicao.PrazoBoletoDias() != 30 {
		t.Fatalf("alterado = revisão %d, tabela %d, frete %+v, condição %+v",
			alterado.Revisao, alterado.Itens[0].PrecoTabela.Centavos(), alterado.Frete, alterado.Condicao)
	}
}

func TestPropostaRecusaFreteSemTipoECondicaoInvalida(t *testing.T) {
	c := novoCenario(t)
	frete := int64(100)

	dados := c.dados()
	dados.FreteEstimadoCentavos = &frete
	_, err := c.pedidos.Criar(c.ctx, portsin.CriarPedidoInput{UsuarioID: c.comercial, EmpresaID: c.mia, Dados: dados})
	esperarErroDaProposta(t, err, domain.ErrValidacao, "tipo do frete estimado")

	dados = c.dados()
	dados.FreteTipo = "cif"
	_, err = c.pedidos.Criar(c.ctx, portsin.CriarPedidoInput{UsuarioID: c.comercial, EmpresaID: c.mia, Dados: dados})
	esperarErroDaProposta(t, err, domain.ErrValidacao, "valor do frete é obrigatório")

	prazo := 10
	dados = c.dados()
	dados.PrazoEntregaDias = &prazo
	_, err = c.pedidos.Criar(c.ctx, portsin.CriarPedidoInput{UsuarioID: c.comercial, EmpresaID: c.mia, Dados: dados})
	esperarErroDaProposta(t, err, domain.ErrValidacao, "só vale na importação")
}

func TestConfirmarRecusaPropostaVencida(t *testing.T) {
	c := novoCenario(t)
	pedido := c.criar(t, c.comercial, c.mia, c.dados())
	vencida := domainpedido.Hoje(time.Now().UTC()).AddDate(0, 0, -1)
	c.pedido(pedido.ID).Validade = &vencida

	_, err := c.pedidos.Confirmar(c.ctx, portsin.TransicaoPedidoInput{UsuarioID: c.comercial, PedidoID: pedido.ID, Versao: pedido.Versao})
	esperarErroDaProposta(t, err, domain.ErrConflito, "proposta vencida em")
}
