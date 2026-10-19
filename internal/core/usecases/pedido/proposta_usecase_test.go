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
	c.m.precosTabela[c.cimento.ID.String()+"importacao"] = 1800
	entrada, prazo, frete := 2000, 45, int64(9000)
	validade := domainpedido.Hoje(time.Now().UTC()).AddDate(0, 0, 7)
	dados := c.dados()
	dados.Modelo = "importacao"
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

func TestDocumentoDaPropostaJuntaAsPartesDoPedido(t *testing.T) {
	c := novoCenario(t)
	c.m.empresas[c.mia].RazaoSocial = "MIA Materiais Ltda"
	numero := "120"
	c.obra.Endereco.Logradouro, c.obra.Endereco.Numero, c.obra.Endereco.Cidade = "Rua das Flores", &numero, "João Pessoa"
	dados := c.dados()
	dados.ObraID = &c.obra.ID
	pedido := c.criar(t, c.comercial, c.mia, dados)

	documento, err := c.pedidos.Documento(c.ctx, pedido.ID, c.operador)
	if err != nil {
		t.Fatalf("Documento: %v", err)
	}
	if documento.Empresa.RazaoSocial != "MIA Materiais Ltda" || documento.Cliente.Nome != "Construtora Alfa" || documento.Numero != pedido.Numero {
		t.Fatalf("documento = %+v", documento)
	}
	if documento.Obra == nil || documento.Obra.Endereco != "Rua das Flores, 120 — João Pessoa" {
		t.Fatalf("obra = %+v", documento.Obra)
	}
	if documento.Representante == nil || documento.Representante.ID != c.outroRepresentante.ID {
		t.Fatalf("representante = %+v", documento.Representante)
	}
	if len(documento.Itens) != 2 || documento.Itens[0].Nome != "Cimento CP II" || len(documento.Condicoes) == 0 {
		t.Fatalf("itens e condições = %+v / %v", documento.Itens, documento.Condicoes)
	}

	_, err = c.pedidos.Documento(c.ctx, pedido.ID, c.comercialRed)
	esperarErroDaProposta(t, err, domain.ErrPermissao, "sem acesso a esta empresa")

	if modelos := c.pedidos.ModelosDeVenda(c.ctx); len(modelos) != 2 {
		t.Fatalf("modelos = %+v", modelos)
	}
}

func TestEntradaSoNaImportacaoEPrazoSemEntradaContaDaConfirmacao(t *testing.T) {
	c := novoCenario(t)
	entrada := 1000
	dados := c.dados()
	dados.EntradaBP = &entrada
	_, err := c.pedidos.Criar(c.ctx, portsin.CriarPedidoInput{UsuarioID: c.comercial, EmpresaID: c.mia, Dados: dados})
	esperarErroDaProposta(t, err, domain.ErrValidacao, "entrada só vale na importação")

	semEntrada := 0
	importacao := c.dados()
	importacao.Modelo = "importacao"
	importacao.EntradaBP = &semEntrada
	pedido := c.criar(t, c.comercial, c.mia, importacao)
	confirmado, err := c.pedidos.Confirmar(c.ctx, portsin.TransicaoPedidoInput{UsuarioID: c.comercial, PedidoID: pedido.ID, Versao: pedido.Versao})
	if err != nil {
		t.Fatalf("Confirmar: %v", err)
	}
	esperado := domainpedido.Hoje(time.Now().UTC()).AddDate(0, 0, 120)
	if confirmado.PrazoLimite == nil || !confirmado.PrazoLimite.Equal(esperado) {
		t.Fatalf("prazo-limite = %v, esperado %v", confirmado.PrazoLimite, esperado)
	}

	_, err = c.pedidos.RegistrarEntrada(c.ctx, portsin.RegistrarEntradaInput{
		UsuarioID: c.financeiro, PedidoID: confirmado.ID, Versao: confirmado.Versao, PagaEm: time.Now().UTC(), ValorCentavos: 100,
	})
	esperarErroDaProposta(t, err, domain.ErrValidacao, "não tem entrada")
}
