package postgres_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/estoque"
	"github.com/google/uuid"
)

func (c *cenarioBancoLocais) reservar(t *testing.T, local *domainestoque.Local, quantidade int) *domainestoque.Reserva {
	t.Helper()

	operacao, err := c.carregar(t, local).Reservar(domainestoque.ReservaInput{
		Local:      local,
		Quantidade: domainestoque.QuantidadeDe(quantidade),
		UsuarioID:  c.usuarioID,
		Agora:      time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("reserva de domínio falhou: %v", err)
	}
	if err := c.estoque.Gravar(c.ctx, operacao); err != nil {
		t.Fatalf("gravar a reserva falhou: %v", err)
	}

	return operacao.ReservaCriada
}

func TestReservaGravaReservadoEBaixaComASaida(t *testing.T) {
	c := novoCenarioLocais(t)
	reservas := postgres.NewReservaRepository(c.banco)
	galpao := c.novoLocal(t, c.mia, "Galpão")
	c.gravar(t, domainestoque.TipoMovimentoEntrada, galpao, 10, custoEm(100))

	reserva := c.reservar(t, galpao, 4)
	salva, err := reservas.Obter(c.ctx, reserva.ID)
	if err != nil || salva == nil || salva.Status != domainestoque.StatusReservaAtiva || salva.Quantidade != 4 || salva.EncerradaEm != nil {
		t.Fatalf("reserva salva = %+v (%v)", salva, err)
	}
	if ausente, err := reservas.Obter(c.ctx, uuid.New()); err != nil || ausente != nil {
		t.Errorf("reserva inexistente = %+v, %v; esperado nil sem erro", ausente, err)
	}

	estoque := c.carregar(t, galpao)
	if estoque.Produto.Reservado.Quantidade() != 4 || estoque.SaldoNoLocal(galpao.ID).Reservado != 4 || estoque.Produto.VersaoEstoque != 2 {
		t.Errorf("depois de reservar: produto reservado %d, local %d, versão %d",
			estoque.Produto.Reservado.Quantidade(), estoque.SaldoNoLocal(galpao.ID).Reservado, estoque.Produto.VersaoEstoque)
	}

	baixa, err := estoque.Movimentar(domainestoque.MovimentoLocalInput{
		Tipo:       domainestoque.TipoMovimentoSaida,
		Local:      galpao,
		Quantidade: domainestoque.QuantidadeDe(4),
		Reserva:    salva,
		UsuarioID:  c.usuarioID,
		Agora:      time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("baixa de domínio falhou: %v", err)
	}
	if err := c.estoque.Gravar(c.ctx, baixa); err != nil {
		t.Fatalf("gravar a baixa falhou: %v", err)
	}

	baixada, err := reservas.Obter(c.ctx, reserva.ID)
	if err != nil || baixada.Status != domainestoque.StatusReservaBaixada || baixada.EncerradaEm == nil {
		t.Errorf("reserva depois da baixa = %+v (%v)", baixada, err)
	}
	depois := c.carregar(t, galpao)
	if depois.Produto.Saldo.Quantidade() != 6 || depois.Produto.Reservado.Quantidade() != 0 || depois.SaldoNoLocal(galpao.ID).Reservado != 0 {
		t.Errorf("depois da baixa: saldo %d, reservado %d", depois.Produto.Saldo.Quantidade(), depois.Produto.Reservado.Quantidade())
	}

	movimentos, err := c.movimentos.Listar(c.ctx, portsout.MovimentoFiltro{PaginacaoFiltro: todasAsPaginas, ProdutoID: &c.produto.ID})
	if err != nil || len(movimentos) == 0 || movimentos[0].ReservaID == nil || *movimentos[0].ReservaID != reserva.ID {
		t.Errorf("movimento da baixa sem a reserva: %+v (%v)", movimentos, err)
	}

	liberacao := &domainestoque.OperacaoEstoque{
		ProdutoID:        c.produto.ID,
		VersaoLida:       depois.Produto.VersaoEstoque,
		Saldo:            depois.Produto.Saldo,
		Reservado:        depois.Produto.Reservado,
		Saldos:           []domainestoque.SaldoLocal{depois.SaldoNoLocal(galpao.ID)},
		ReservaEncerrada: &domainestoque.Reserva{ID: reserva.ID, Status: domainestoque.StatusReservaLiberada, EncerradaEm: &salva.CriadoEm},
		AtualizadoEm:     time.Now().UTC(),
	}
	if err := c.estoque.Gravar(c.ctx, liberacao); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("encerrar reserva já baixada = %v, esperado conflito", err)
	}
	if versao := c.carregar(t, galpao).Produto.VersaoEstoque; versao != depois.Produto.VersaoEstoque {
		t.Errorf("versão depois do conflito = %d, esperado %d: a operação recusada não pode deixar rastro", versao, depois.Produto.VersaoEstoque)
	}
}

func TestListarReservasFiltraStatusLocalEEmpresa(t *testing.T) {
	c := novoCenarioLocais(t)
	reservas := postgres.NewReservaRepository(c.banco)
	galpao := c.novoLocal(t, c.mia, "Galpão")
	daRed := c.novoLocal(t, c.red, "Depósito")
	c.gravar(t, domainestoque.TipoMovimentoEntrada, galpao, 5, custoEm(100))
	c.gravar(t, domainestoque.TipoMovimentoEntrada, daRed, 5, custoEm(100))
	primeira := c.reservar(t, galpao, 1)
	c.reservar(t, galpao, 2)
	c.reservar(t, daRed, 3)

	liberacao, err := c.carregar(t, galpao).LiberarReserva(primeira, time.Now().UTC())
	if err != nil {
		t.Fatalf("liberação de domínio falhou: %v", err)
	}
	if err := c.estoque.Gravar(c.ctx, liberacao); err != nil {
		t.Fatalf("gravar a liberação falhou: %v", err)
	}

	casos := []struct {
		nome     string
		filtro   portsout.ReservaFiltro
		esperado int
	}{
		{"todas", portsout.ReservaFiltro{}, 3},
		{"ativas", portsout.ReservaFiltro{Status: domainestoque.StatusReservaAtiva}, 2},
		{"do galpão", portsout.ReservaFiltro{LocalID: &galpao.ID}, 2},
		{"da Red", portsout.ReservaFiltro{EmpresaIDs: []uuid.UUID{c.red}}, 1},
		{"acesso vazio", portsout.ReservaFiltro{EmpresaIDs: []uuid.UUID{}}, 0},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			caso.filtro.PaginacaoFiltro = todasAsPaginas
			caso.filtro.ProdutoID = c.produto.ID
			itens, err := reservas.Listar(c.ctx, caso.filtro)
			if err != nil || len(itens) != caso.esperado {
				t.Errorf("%d reservas (%v), esperado %d", len(itens), err, caso.esperado)
			}
		})
	}

	if estoque := c.carregar(t, galpao, daRed); estoque.Produto.Reservado.Quantidade() != 5 {
		t.Errorf("reservado do produto = %d, esperado 5", estoque.Produto.Reservado.Quantidade())
	}
}

func TestBancoRecusaReservadoAcimaDoSaldo(t *testing.T) {
	c := novoCenarioLocais(t)
	galpao := c.novoLocal(t, c.mia, "Galpão")
	c.gravar(t, domainestoque.TipoMovimentoEntrada, galpao, 2, custoEm(100))

	if _, err := c.banco.ExecContext(c.ctx,
		`UPDATE produto_saldo_local SET reservado = 3 WHERE produto_id = $1 AND local_id = $2`, c.produto.ID, galpao.ID,
	); err == nil {
		t.Error("reservado acima da quantidade do local foi aceito")
	}
	if _, err := c.banco.ExecContext(c.ctx, `UPDATE produto SET quantidade_reservada = 3 WHERE id = $1`, c.produto.ID); err == nil {
		t.Error("reservado acima do saldo do produto foi aceito")
	}
}
