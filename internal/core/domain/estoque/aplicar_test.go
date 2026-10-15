package estoque_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

func aplicando(t *testing.T, c *cenarioEstoque) func(*domainestoque.OperacaoEstoque, error) *domainestoque.OperacaoEstoque {
	return func(operacao *domainestoque.OperacaoEstoque, err error) *domainestoque.OperacaoEstoque {
		t.Helper()

		if err != nil {
			t.Fatalf("operação falhou: %v", err)
		}
		if err := c.estoque.Aplicar(operacao); err != nil {
			t.Fatalf("aplicar falhou: %v", err)
		}

		return operacao
	}
}

func TestAplicarEncadeiaLiberarSairEReservarDeNovo(t *testing.T) {
	c := novoCenarioEstoque(t)
	aplicar := aplicando(t, c)
	c.comLote(c.galpao, 10, centavos(100), 0)
	reservou := aplicar(c.reservar(c.galpao, 8))

	liberou := aplicar(c.estoque.LiberarReserva(reservou.ReservaCriada, c.inicio.AddDate(0, 1, 0)))
	saiu := aplicar(c.movimentar(domainestoque.TipoMovimentoSaida, c.galpao, 3, nil))
	reservouDeNovo := aplicar(c.reservar(c.galpao, 5))

	versoes := []int64{reservou.VersaoLida, liberou.VersaoLida, saiu.VersaoLida, reservouDeNovo.VersaoLida}
	for indice, versao := range versoes {
		if versao != int64(7+indice) {
			t.Errorf("operação %d leu a versão %d, esperado %d", indice, versao, 7+indice)
		}
	}
	if c.estoque.Produto.VersaoEstoque != 11 {
		t.Errorf("versão final = %d, esperado 11", c.estoque.Produto.VersaoEstoque)
	}
	if c.estoque.Produto.Saldo.Quantidade() != 7 || c.estoque.Produto.Reservado.Quantidade() != 5 {
		t.Errorf("produto com saldo %d e reservado %d, esperado 7 e 5",
			c.estoque.Produto.Saldo.Quantidade(), c.estoque.Produto.Reservado.Quantidade())
	}
	if saldo := c.estoque.SaldoNoLocal(c.galpao.ID); saldo.Quantidade != 7 || saldo.Reservado != 5 {
		t.Errorf("saldo do galpão = %+v, esperado 7 com 5 reservados", saldo)
	}
	if c.estoque.TotalAlocado != 7 {
		t.Errorf("total alocado = %d, esperado 7", c.estoque.TotalAlocado)
	}
}

func TestAplicarConsomeLotesSemAlterarOOriginal(t *testing.T) {
	c := novoCenarioEstoque(t)
	antigo := c.comLote(c.galpao, 4, centavos(100), 0)
	recente := c.comLote(c.galpao, 6, centavos(200), 1)
	aplicar := aplicando(t, c)

	aplicar(c.movimentar(domainestoque.TipoMovimentoSaida, c.galpao, 5, nil))

	if antigo.QuantidadeAtual != 4 {
		t.Errorf("lote original alterado para %d", antigo.QuantidadeAtual)
	}

	segunda, err := c.movimentar(domainestoque.TipoMovimentoSaida, c.galpao, 2, nil)
	if err != nil {
		t.Fatalf("segunda saída falhou: %v", err)
	}
	consumos := segunda.Movimento.Consumos
	if len(consumos) != 1 || consumos[0].LoteID != recente.ID || consumos[0].Quantidade != 2 {
		t.Errorf("segunda saída consumiu %+v, esperado 2 do lote recente", consumos)
	}
}

func TestAplicarIncluiLotesCriados(t *testing.T) {
	c := novoCenarioEstoque(t)
	aplicar := aplicando(t, c)

	entrada := aplicar(c.movimentar(domainestoque.TipoMovimentoEntrada, c.galpao, 5, centavos(300)))

	if len(c.estoque.Lotes) != 1 || c.estoque.Lotes[0].ID != entrada.LotesCriados[0].ID {
		t.Fatalf("lotes do estoque = %+v, esperado o lote da entrada", c.estoque.Lotes)
	}
	if c.estoque.TotalAlocado != 5 || c.estoque.Produto.Saldo.Quantidade() != 5 {
		t.Errorf("alocado %d e saldo %d, esperado 5 e 5", c.estoque.TotalAlocado, c.estoque.Produto.Saldo.Quantidade())
	}
}

func TestAplicarRecusaOperacaoDeOutraVersaoOuProduto(t *testing.T) {
	c := novoCenarioEstoque(t)
	c.comLote(c.galpao, 10, centavos(100), 0)

	primeira, err := c.reservar(c.galpao, 2)
	if err != nil {
		t.Fatalf("reservar falhou: %v", err)
	}
	segunda, err := c.reservar(c.galpao, 3)
	if err != nil {
		t.Fatalf("reservar falhou: %v", err)
	}
	if err := c.estoque.Aplicar(primeira); err != nil {
		t.Fatalf("aplicar falhou: %v", err)
	}

	if err := c.estoque.Aplicar(segunda); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("operação lida antes da anterior = %v, esperado conflito", err)
	}

	deOutroProduto := *primeira
	deOutroProduto.ProdutoID = uuid.New()
	deOutroProduto.VersaoLida = c.estoque.Produto.VersaoEstoque
	if err := c.estoque.Aplicar(&deOutroProduto); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("operação de outro produto = %v, esperado erro de validação", err)
	}
	if err := c.estoque.Aplicar(nil); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("operação nula = %v, esperado erro de validação", err)
	}
}

func TestReservarDeItemDePedidoExigeOrigem(t *testing.T) {
	c := novoCenarioEstoque(t)
	c.comLote(c.galpao, 10, centavos(100), 0)
	itemID := uuid.New()
	nulo := uuid.Nil

	casos := []struct {
		nome      string
		origem    domainestoque.OrigemReserva
		origemID  *uuid.UUID
		sentinela error
	}{
		{"pedido com item", domainestoque.OrigemReservaPedidoItem, &itemID, nil},
		{"pedido sem item", domainestoque.OrigemReservaPedidoItem, nil, domain.ErrValidacao},
		{"pedido com item nulo", domainestoque.OrigemReservaPedidoItem, &nulo, domain.ErrValidacao},
		{"origem desconhecida", domainestoque.OrigemReservaDe("compra"), nil, domain.ErrValidacao},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			operacao, err := c.estoque.Reservar(domainestoque.ReservaInput{
				Local:      c.galpao,
				Quantidade: domainestoque.QuantidadeDe(2),
				Origem:     caso.origem,
				OrigemID:   caso.origemID,
				UsuarioID:  c.usuario,
				Agora:      c.inicio,
			})
			if !errors.Is(err, caso.sentinela) {
				t.Fatalf("erro %v, esperado %v", err, caso.sentinela)
			}
			if caso.sentinela != nil {
				return
			}

			reserva := operacao.ReservaCriada
			if reserva.Origem != domainestoque.OrigemReservaPedidoItem || reserva.OrigemID == nil || *reserva.OrigemID != itemID {
				t.Errorf("reserva criada com origem %q e item %v", reserva.Origem, reserva.OrigemID)
			}
		})
	}
}
