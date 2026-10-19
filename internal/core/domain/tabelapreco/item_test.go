package tabelapreco_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	domaintabelapreco "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/tabelapreco"
	"github.com/google/uuid"
)

var agora = time.Date(2026, 10, 15, 15, 0, 0, 0, time.UTC)

func esperarErro(t *testing.T, err error, sentinela error, trecho string) {
	t.Helper()

	if !errors.Is(err, sentinela) || !strings.Contains(err.Error(), trecho) {
		t.Fatalf("erro = %v; esperado %v com %q", err, sentinela, trecho)
	}
}

func dadosDeTeste() domaintabelapreco.DadosItem {
	return domaintabelapreco.DadosItem{
		ProdutoID:     uuid.New(),
		Modelo:        " Importacao ",
		PrecoCentavos: 12990,
		VigenteDe:     time.Date(2026, 11, 1, 10, 30, 0, 0, time.UTC),
		Observacao:    "  reajuste do fornecedor  ",
	}
}

func TestNovoItemNormalizaModeloDataEObservacao(t *testing.T) {
	item, err := domaintabelapreco.NovoItem(dadosDeTeste(), uuid.New(), agora)
	if err != nil {
		t.Fatalf("NovoItem: %v", err)
	}
	if item.Modelo != domainpedido.ModeloImportacao || item.Observacao != "reajuste do fornecedor" ||
		!item.VigenteDe.Equal(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)) || item.ID == uuid.Nil {
		t.Fatalf("item = %+v", item)
	}
	if !item.Futuro(agora) {
		t.Fatalf("vigência de novembro é futura em outubro")
	}
	if item.Futuro(time.Date(2026, 11, 1, 4, 0, 0, 0, time.UTC)) {
		t.Fatalf("a vigência começa no próprio dia em Brasília")
	}
}

func TestNovoItemRecusaDadosInvalidos(t *testing.T) {
	casos := []struct {
		nome    string
		ajustar func(*domaintabelapreco.DadosItem)
		trecho  string
	}{
		{"sem produto", func(d *domaintabelapreco.DadosItem) { d.ProdutoID = uuid.Nil }, "produto do preço"},
		{"preço zero", func(d *domaintabelapreco.DadosItem) { d.PrecoCentavos = 0 }, "maior que zero"},
		{"sem vigência", func(d *domaintabelapreco.DadosItem) { d.VigenteDe = time.Time{} }, "início da vigência"},
		{"modelo inválido", func(d *domaintabelapreco.DadosItem) { d.Modelo = "consignado" }, "modelo de venda"},
		{"observação longa", func(d *domaintabelapreco.DadosItem) { d.Observacao = strings.Repeat("a", 501) }, "500 caracteres"},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			dados := dadosDeTeste()
			caso.ajustar(&dados)

			_, err := domaintabelapreco.NovoItem(dados, uuid.New(), agora)
			esperarErro(t, err, domain.ErrValidacao, caso.trecho)
		})
	}
}

func TestPrecoVigentePrefereATabelaECaiNoProduto(t *testing.T) {
	tabela, produto := int64(9000), int64(10000)
	itemID := uuid.New()
	inicio := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

	var vigente domaintabelapreco.PrecoVigente
	vigente.Resolver(&tabela, &itemID, &inicio, &produto)
	if vigente.Origem != domaintabelapreco.OrigemPrecoTabela || *vigente.PrecoCentavos != 9000 || *vigente.ItemID != itemID {
		t.Fatalf("com tabela = %+v", vigente)
	}

	vigente.Resolver(nil, nil, nil, &produto)
	if vigente.Origem != domaintabelapreco.OrigemPrecoProduto || *vigente.PrecoCentavos != 10000 || vigente.ItemID != nil {
		t.Fatalf("sem tabela = %+v", vigente)
	}

	vigente.Resolver(nil, nil, nil, nil)
	if vigente.Origem != domaintabelapreco.OrigemPrecoSemPreco || vigente.PrecoCentavos != nil {
		t.Fatalf("sem preço = %+v", vigente)
	}
}
