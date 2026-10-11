package estoque_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	"github.com/google/uuid"
)

func (c *cenario) reservarNoLocal(t *testing.T, produto *domainestoque.Produto, local *domainestoque.Local, quantidade int) *domainestoque.Reserva {
	t.Helper()

	reserva, err := c.reserva.Reservar(c.ctx, portsin.ReservarEstoqueInput{
		UsuarioID:  c.usuarioID,
		ProdutoID:  produto.ID,
		LocalID:    local.ID,
		Quantidade: quantidade,
		Observacao: "pedido do cliente",
	})
	if err != nil {
		t.Fatalf("reservar falhou: %v", err)
	}

	return reserva
}

func TestReservarValidaPerfilLocalEDisponivel(t *testing.T) {
	c := novoCenario(t)
	mia := c.novaEmpresa(true)
	red := c.novaEmpresa(true)
	galpao := c.novoLocal(t, mia, "Galpão")
	produto := c.produtoDeTeste(t)
	c.entradaNoLocal(t, produto, galpao, 5, 100)
	administradorDaRed := c.novoUsuarioComPerfil("Diretoria Red", true, false, red)

	casos := []struct {
		nome      string
		usuario   uuid.UUID
		produtoID uuid.UUID
		localID   uuid.UUID
		qtd       int
		sentinela error
	}{
		{"sem perfil de administrador", c.visitanteID, produto.ID, galpao.ID, 1, domain.ErrPermissao},
		{"local inexistente", c.usuarioID, produto.ID, uuid.New(), 1, domain.ErrValidacao},
		{"local fora do acesso", administradorDaRed, produto.ID, galpao.ID, 1, domain.ErrPermissao},
		{"produto inexistente", c.usuarioID, uuid.New(), galpao.ID, 1, domain.ErrNotFound},
		{"quantidade zero", c.usuarioID, produto.ID, galpao.ID, 0, domain.ErrValidacao},
		{"acima do disponível", c.usuarioID, produto.ID, galpao.ID, 6, domain.ErrValidacao},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			_, err := c.reserva.Reservar(c.ctx, portsin.ReservarEstoqueInput{
				UsuarioID:  caso.usuario,
				ProdutoID:  caso.produtoID,
				LocalID:    caso.localID,
				Quantidade: caso.qtd,
			})
			if !errors.Is(err, caso.sentinela) {
				t.Errorf("erro %v, esperado %v", err, caso.sentinela)
			}
		})
	}

	reserva := c.reservarNoLocal(t, produto, galpao, 3)
	if reserva.Status != domainestoque.StatusReservaAtiva || reserva.Origem != domainestoque.OrigemReservaManual || reserva.UsuarioID != c.usuarioID {
		t.Errorf("reserva criada = %+v", reserva)
	}

	saldo, err := c.movimento.Saldo(c.ctx, produto.ID, c.usuarioID)
	if err != nil || saldo.Produto.Reservado.Quantidade() != 3 || saldo.Produto.Disponivel() != 2 || saldo.Locais[0].Reservado != 3 {
		t.Errorf("saldo depois de reservar = %+v (%v), esperado 3 reservados e 2 disponíveis", saldo, err)
	}
}

func TestBaixaELiberacaoDeReserva(t *testing.T) {
	c := novoCenario(t)
	galpao := c.localPadrao(t)
	produto := c.produtoDeTeste(t)
	c.entradaNoLocal(t, produto, galpao, 10, 100)
	paraBaixar := c.reservarNoLocal(t, produto, galpao, 4)
	paraLiberar := c.reservarNoLocal(t, produto, galpao, 5)

	saida := func(quantidade int, reservaID *uuid.UUID) (*domainestoque.Movimento, error) {
		return c.movimento.Movimentar(c.ctx, portsin.MovimentarEstoqueInput{
			UsuarioID:  c.usuarioID,
			ProdutoID:  produto.ID,
			LocalID:    galpao.ID,
			Tipo:       "saida",
			Quantidade: quantidade,
			ReservaID:  reservaID,
		})
	}

	if _, err := saida(2, nil); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("saída sem reserva acima do disponível = %v, esperado erro de validação", err)
	}
	inexistente := uuid.New()
	if _, err := saida(4, &inexistente); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("saída com reserva inexistente = %v, esperado erro de validação", err)
	}

	movimento, err := saida(4, &paraBaixar.ID)
	if err != nil {
		t.Fatalf("baixa da reserva falhou: %v", err)
	}
	if movimento.ReservaID == nil || *movimento.ReservaID != paraBaixar.ID || movimento.SaldoApos.Quantidade() != 6 {
		t.Errorf("movimento da baixa = %+v", movimento)
	}
	if _, err := saida(4, &paraBaixar.ID); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("baixar reserva já baixada = %v, esperado conflito", err)
	}

	if _, err := c.reserva.Liberar(c.ctx, paraLiberar.ID, c.visitanteID); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("liberar sem perfil = %v, esperado erro de permissão", err)
	}
	if _, err := c.reserva.Liberar(c.ctx, uuid.New(), c.usuarioID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("liberar reserva inexistente = %v, esperado não encontrado", err)
	}
	liberada, err := c.reserva.Liberar(c.ctx, paraLiberar.ID, c.usuarioID)
	if err != nil || liberada.Status != domainestoque.StatusReservaLiberada || liberada.EncerradaEm == nil {
		t.Fatalf("liberar = %+v (%v)", liberada, err)
	}
	if _, err := c.reserva.Liberar(c.ctx, paraLiberar.ID, c.usuarioID); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("liberar duas vezes = %v, esperado conflito", err)
	}

	saldo, err := c.movimento.Saldo(c.ctx, produto.ID, c.usuarioID)
	if err != nil || saldo.Produto.Saldo.Quantidade() != 6 || saldo.Produto.Reservado.Quantidade() != 0 || saldo.Locais[0].Disponivel() != 6 {
		t.Errorf("saldo final = %+v (%v), esperado 6 disponíveis", saldo, err)
	}

	ativas, err := c.reserva.Listar(c.ctx, portsin.ListarReservasInput{UsuarioID: c.usuarioID, ProdutoID: produto.ID, Status: "ativa"})
	if err != nil || len(ativas) != 0 {
		t.Errorf("reservas ativas = %d (%v), esperado nenhuma", len(ativas), err)
	}
	todas, err := c.reserva.Listar(c.ctx, portsin.ListarReservasInput{UsuarioID: c.usuarioID, ProdutoID: produto.ID})
	if err != nil || len(todas) != 2 {
		t.Errorf("reservas = %d (%v), esperado 2", len(todas), err)
	}
	if _, err := c.reserva.Listar(c.ctx, portsin.ListarReservasInput{UsuarioID: c.usuarioID, ProdutoID: produto.ID, Status: "cancelada"}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("status desconhecido = %v, esperado erro de validação", err)
	}
}

func TestReservasFicamIsoladasPorEmpresa(t *testing.T) {
	c := novoCenario(t)
	mia := c.novaEmpresa(true)
	red := c.novaEmpresa(true)
	galpaoMia := c.novoLocal(t, mia, "Galpão MIA")
	depositoRed := c.novoLocal(t, red, "Depósito Red")
	produto := c.produtoDeTeste(t)
	c.entradaNoLocal(t, produto, galpaoMia, 5, 100)
	c.entradaNoLocal(t, produto, depositoRed, 5, 100)
	daMia := c.reservarNoLocal(t, produto, galpaoMia, 1)
	c.reservarNoLocal(t, produto, depositoRed, 2)
	ana := c.novoUsuarioComPerfil("Ana", false, false, mia)

	reservas, err := c.reserva.Listar(c.ctx, portsin.ListarReservasInput{UsuarioID: ana, ProdutoID: produto.ID})
	if err != nil || len(reservas) != 1 || reservas[0].ID != daMia.ID {
		t.Errorf("reservas da Ana = %+v (%v), esperado só a da MIA", reservas, err)
	}
	if _, err := c.reserva.Listar(c.ctx, portsin.ListarReservasInput{UsuarioID: ana, ProdutoID: produto.ID, LocalID: &depositoRed.ID}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("reservas do local alheio = %v, esperado erro de permissão", err)
	}

	semAcesso := c.novoUsuarioComPerfil("Sem acesso", false, false)
	if reservas, err := c.reserva.Listar(c.ctx, portsin.ListarReservasInput{UsuarioID: semAcesso, ProdutoID: produto.ID}); err != nil || len(reservas) != 0 {
		t.Errorf("reservas para acesso vazio = %d (%v), esperado nenhuma", len(reservas), err)
	}
}

func TestVitrineEscondeProdutoTodoReservado(t *testing.T) {
	c := novoCenario(t)
	galpao := c.localPadrao(t)
	produto := c.produtoDeTeste(t)
	c.entradaNoLocal(t, produto, galpao, 3, 100)

	if _, err := c.publico.ObterProdutoPorSlug(c.ctx, produto.Slug.Valor()); err != nil {
		t.Fatalf("produto com disponível fora da vitrine: %v", err)
	}

	c.reservarNoLocal(t, produto, galpao, 3)

	if _, err := c.publico.ObterProdutoPorSlug(c.ctx, produto.Slug.Valor()); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("produto todo reservado no detalhe público = %v, esperado não encontrado", err)
	}
	lista, err := c.publico.ListarProdutos(c.ctx, portsin.ListarProdutosPublicosInput{})
	if err != nil || len(lista) != 0 {
		t.Errorf("vitrine com produto todo reservado = %d (%v), esperado vazia", len(lista), err)
	}
}
