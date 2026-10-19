package postgres_test

import (
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	domaintabelapreco "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/tabelapreco"
	portstabelapreco "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/tabelapreco"
	"github.com/google/uuid"
)

var _ portstabelapreco.ProdutoRepository = (*postgres.ProdutoRepository)(nil)

type cenarioBancoTabela struct {
	*cenarioBancoLocais
	tabela *postgres.TabelaPrecoRepository
}

func novoCenarioTabela(t *testing.T) *cenarioBancoTabela {
	t.Helper()

	base := novoCenarioLocais(t)

	return &cenarioBancoTabela{cenarioBancoLocais: base, tabela: postgres.NewTabelaPrecoRepository(base.banco)}
}

func dia(ano int, mes time.Month, d int) time.Time {
	return time.Date(ano, mes, d, 0, 0, 0, 0, time.UTC)
}

func (c *cenarioBancoTabela) lancar(t *testing.T, produtoID uuid.UUID, modelo string, preco int64, vigenteDe time.Time) *domaintabelapreco.Item {
	t.Helper()

	item, err := domaintabelapreco.NovoItem(domaintabelapreco.DadosItem{
		ProdutoID: produtoID, Modelo: modelo, PrecoCentavos: preco, VigenteDe: vigenteDe,
	}, c.usuarioID, time.Now().UTC())
	if err != nil {
		t.Fatalf("NovoItem: %v", err)
	}
	if err := c.tabela.Criar(c.ctx, item); err != nil {
		t.Fatalf("Criar: %v", err)
	}

	return item
}

func (c *cenarioBancoTabela) vigente(t *testing.T, modelo domainpedido.Modelo, data time.Time) *domaintabelapreco.PrecoVigente {
	t.Helper()

	vigentes, err := c.tabela.Vigentes(c.ctx, portstabelapreco.VigenteFiltro{Modelo: modelo, Data: data, ProdutoID: &c.produto.ID})
	if err != nil || len(vigentes) != 1 {
		t.Fatalf("Vigentes = %+v, %v", vigentes, err)
	}

	return vigentes[0]
}

func TestTabelaPrecoRepositoryResolveAVigenciaPorDia(t *testing.T) {
	c := novoCenarioTabela(t)
	setembro := c.lancar(t, c.produto.ID, "pronta_entrega", 10000, dia(2026, 9, 1))
	novembro := c.lancar(t, c.produto.ID, "pronta_entrega", 11000, dia(2026, 11, 1))
	c.lancar(t, c.produto.ID, "importacao", 8000, dia(2026, 9, 1))

	if antes := c.vigente(t, domainpedido.ModeloProntaEntrega, dia(2026, 8, 31)); antes.Origem != domaintabelapreco.OrigemPrecoSemPreco || antes.PrecoCentavos != nil {
		t.Fatalf("antes da primeira vigência = %+v", antes)
	}
	outubro := c.vigente(t, domainpedido.ModeloProntaEntrega, dia(2026, 10, 31))
	if outubro.Origem != domaintabelapreco.OrigemPrecoTabela || *outubro.PrecoCentavos != 10000 || *outubro.ItemID != setembro.ID ||
		outubro.ProdutoNome != c.produto.Nome || outubro.Unidade == "" {
		t.Fatalf("outubro = %+v", outubro)
	}
	if primeiro := c.vigente(t, domainpedido.ModeloProntaEntrega, dia(2026, 11, 1)); *primeiro.PrecoCentavos != 11000 || *primeiro.ItemID != novembro.ID {
		t.Fatalf("1º de novembro = %+v", primeiro)
	}
	if importacao := c.vigente(t, domainpedido.ModeloImportacao, dia(2026, 12, 1)); *importacao.PrecoCentavos != 8000 {
		t.Fatalf("importação = %+v", importacao)
	}

	historico, err := c.tabela.Listar(c.ctx, portstabelapreco.ItemFiltro{ProdutoID: &c.produto.ID, Modelo: domainpedido.ModeloProntaEntrega})
	if err != nil || len(historico) != 2 || historico[0].ID != novembro.ID || historico[1].ID != setembro.ID {
		t.Fatalf("histórico = %+v, %v", historico, err)
	}
}

func TestTabelaPrecoRepositoryCaiNoPrecoDoProdutoEFiltraPorBusca(t *testing.T) {
	c := novoCenarioTabela(t)
	if _, err := c.banco.ExecContext(c.ctx, `UPDATE produto SET preco = 9900 WHERE id = $1`, c.produto.ID); err != nil {
		t.Fatalf("definir preço do produto: %v", err)
	}

	vigente := c.vigente(t, domainpedido.ModeloProntaEntrega, dia(2026, 10, 1))
	if vigente.Origem != domaintabelapreco.OrigemPrecoProduto || *vigente.PrecoCentavos != 9900 || vigente.ItemID != nil {
		t.Fatalf("vigente pelo produto = %+v", vigente)
	}

	encontrados, err := c.tabela.Vigentes(c.ctx, portstabelapreco.VigenteFiltro{Modelo: domainpedido.ModeloProntaEntrega, Data: dia(2026, 10, 1), Busca: c.produto.Nome[:4]})
	if err != nil || len(encontrados) == 0 {
		t.Fatalf("busca = %+v, %v", encontrados, err)
	}
	nenhum, err := c.tabela.Vigentes(c.ctx, portstabelapreco.VigenteFiltro{Modelo: domainpedido.ModeloProntaEntrega, Data: dia(2026, 10, 1), Busca: "zzz_inexistente"})
	if err != nil || len(nenhum) != 0 {
		t.Fatalf("busca sem resultado = %+v, %v", nenhum, err)
	}
}

func TestTabelaPrecoRepositoryRecusaRepetidaProdutoInativoERemoveSoFuturo(t *testing.T) {
	c := novoCenarioTabela(t)
	hoje := domainpedido.Hoje(time.Now().UTC())
	vigorando := c.lancar(t, c.produto.ID, "pronta_entrega", 10000, hoje)
	futuro := c.lancar(t, c.produto.ID, "pronta_entrega", 12000, hoje.AddDate(0, 1, 0))

	repetido, err := domaintabelapreco.NovoItem(domaintabelapreco.DadosItem{
		ProdutoID: c.produto.ID, Modelo: "pronta_entrega", PrecoCentavos: 1, VigenteDe: hoje,
	}, c.usuarioID, time.Now().UTC())
	if err != nil {
		t.Fatalf("NovoItem: %v", err)
	}
	esperarErro(t, c.tabela.Criar(c.ctx, repetido), domain.ErrConflito, "já existe preço")

	esperarErro(t, c.tabela.RemoverFuturo(c.ctx, vigorando.ID, hoje), domain.ErrConflito, "vigência ainda não começou")
	esperarErro(t, c.tabela.RemoverFuturo(c.ctx, uuid.New(), hoje), domain.ErrNotFound, "preço da tabela não encontrado")
	if err := c.tabela.RemoverFuturo(c.ctx, futuro.ID, hoje); err != nil {
		t.Fatalf("remover o futuro: %v", err)
	}

	if _, err := c.banco.ExecContext(c.ctx, `UPDATE produto SET ativo = false WHERE id = $1`, c.produto.ID); err != nil {
		t.Fatalf("desativar o produto: %v", err)
	}
	inativo, err := domaintabelapreco.NovoItem(domaintabelapreco.DadosItem{
		ProdutoID: c.produto.ID, Modelo: "importacao", PrecoCentavos: 500, VigenteDe: hoje,
	}, c.usuarioID, time.Now().UTC())
	if err != nil {
		t.Fatalf("NovoItem: %v", err)
	}
	esperarErro(t, c.tabela.Criar(c.ctx, inativo), domain.ErrValidacao, "produto inativo")
}
