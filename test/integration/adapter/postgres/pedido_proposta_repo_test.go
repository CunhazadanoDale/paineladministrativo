package postgres_test

import (
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	domaintabelapreco "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/tabelapreco"
	portspedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/pedido"
)

var _ portspedido.PrecoTabelaRepository = (*postgres.TabelaPrecoRepository)(nil)

func TestPedidoRepositoryGravaOsCamposDaProposta(t *testing.T) {
	c := novoCenarioPedidos(t)
	hoje := domainpedido.Hoje(time.Now().UTC())
	validade := hoje.AddDate(0, 0, 9)
	entrada, prazo := 2500, 60
	tabela := int64(1700)
	frete, err := domainpedido.NovoFrete(domainpedido.TipoFreteCIF, &tabela)
	if err != nil {
		t.Fatalf("NovoFrete: %v", err)
	}

	dados := c.dados(domainpedido.ModeloImportacao)
	dados.Validade = &validade
	dados.Condicao = &domainpedido.CondicaoInput{EntradaBP: &entrada, PrazoEntregaDias: &prazo}
	dados.Frete = &frete
	dados.Itens[0].PrecoTabelaCentavos = &tabela
	pedido := c.novoPedido(t, c.mia, dados)

	lido := c.obter(t, pedido.ID)
	if lido.Validade == nil || !lido.Validade.Equal(validade) || lido.Revisao != 1 || lido.Frete == nil ||
		lido.Frete.Tipo() != domainpedido.TipoFreteCIF || *lido.Frete.Centavos() != 1700 {
		t.Fatalf("proposta lida = validade %v, revisão %d, frete %+v", lido.Validade, lido.Revisao, lido.Frete)
	}
	if lido.Condicao.Entrada().PontosBase() != 2500 || *lido.Condicao.PrazoEntregaDias() != 60 {
		t.Fatalf("condição lida = %+v", lido.Condicao)
	}
	if lido.Itens[0].PrecoTabela == nil || lido.Itens[0].PrecoTabela.Centavos() != 1700 || lido.Itens[1].PrecoTabela != nil {
		t.Fatalf("itens lidos = %+v / %+v", lido.Itens[0].PrecoTabela, lido.Itens[1].PrecoTabela)
	}

	versao := lido.Versao
	alteracao := c.dados(domainpedido.ModeloImportacao)
	if err := lido.Alterar(alteracao, time.Now().UTC()); err != nil {
		t.Fatalf("Alterar: %v", err)
	}
	c.gravar(t, &portspedido.Gravacao{
		Pedido: lido, VersaoLida: versao, StatusAnterior: domainpedido.StatusProposta, SubstituirItens: true,
		Historicos: []*domainpedido.Historico{
			domainpedido.NovoHistorico(lido.ID, c.usuarioID, domainpedido.AcaoAlterado, nil, lido.Status, "", time.Now().UTC()),
		},
	})

	alterado := c.obter(t, pedido.ID)
	if alterado.Revisao != 2 || alterado.Frete != nil || !alterado.Validade.Equal(hoje.AddDate(0, 0, 15)) ||
		alterado.Condicao.Entrada().PontosBase() != 3000 || alterado.Itens[0].PrecoTabela != nil {
		t.Fatalf("proposta alterada = revisão %d, frete %+v, validade %v, condição %+v",
			alterado.Revisao, alterado.Frete, alterado.Validade, alterado.Condicao)
	}
}

func TestTabelaPrecoRepositoryPrecoVigenteDoProduto(t *testing.T) {
	c := novoCenarioTabela(t)
	hoje := domainpedido.Hoje(time.Now().UTC())

	if preco, err := c.tabela.PrecoVigente(c.ctx, c.produto.ID, domainpedido.ModeloProntaEntrega, hoje); err != nil || preco != nil {
		t.Fatalf("sem tabela nem preço do produto = %v, %v", preco, err)
	}

	if _, err := c.banco.ExecContext(c.ctx, `UPDATE produto SET preco = 9900 WHERE id = $1`, c.produto.ID); err != nil {
		t.Fatalf("definir preço do produto: %v", err)
	}
	if preco, err := c.tabela.PrecoVigente(c.ctx, c.produto.ID, domainpedido.ModeloProntaEntrega, hoje); err != nil || preco == nil || *preco != 9900 {
		t.Fatalf("preço do produto = %v, %v", preco, err)
	}

	item, err := domaintabelapreco.NovoItem(domaintabelapreco.DadosItem{
		ProdutoID: c.produto.ID, Modelo: "pronta_entrega", PrecoCentavos: 12000, VigenteDe: hoje,
	}, c.usuarioID, time.Now().UTC())
	if err != nil {
		t.Fatalf("NovoItem: %v", err)
	}
	if err := c.tabela.Criar(c.ctx, item); err != nil {
		t.Fatalf("Criar: %v", err)
	}
	if preco, err := c.tabela.PrecoVigente(c.ctx, c.produto.ID, domainpedido.ModeloProntaEntrega, hoje); err != nil || *preco != 12000 {
		t.Fatalf("preço de tabela = %v, %v", preco, err)
	}
	if preco, err := c.tabela.PrecoVigente(c.ctx, c.produto.ID, domainpedido.ModeloProntaEntrega, hoje.AddDate(0, 0, -1)); err != nil || *preco != 9900 {
		t.Fatalf("antes da vigência = %v, %v", preco, err)
	}
	if preco, err := c.tabela.PrecoVigente(c.ctx, c.produto.ID, domainpedido.ModeloImportacao, hoje); err != nil || *preco != 9900 {
		t.Fatalf("outro modelo = %v, %v", preco, err)
	}
}
