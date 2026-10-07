package solicitacao_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

func TestDesignarUsuarioInexistenteRetornaNaoEncontrado(t *testing.T) {
	c := novoCenario(t)

	if _, err := c.aprovador.Designar(c.ctx, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("designação inexistente = %v, esperado não encontrado", err)
	}
}

func TestDesignarUsuarioInativoRecebeErroDeValidacao(t *testing.T) {
	c := novoCenario(t)
	inativo := c.novoUsuario("Ana", false, false)
	c.usuariosRepo.itens[inativo].Ativo = false

	if _, err := c.aprovador.Designar(c.ctx, inativo); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("designação de inativo = %v, esperado erro de validação", err)
	}
}

func TestDesignarUsuarioJaDesignadoRecebeConflito(t *testing.T) {
	c := novoCenario(t)
	usuario := c.novoUsuario("Ana", false, false)
	c.designar(t, usuario)

	if _, err := c.aprovador.Designar(c.ctx, usuario); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("designação duplicada = %v, esperado erro de conflito", err)
	}
}

func TestDesignarEConsultarAprovador(t *testing.T) {
	c := novoCenario(t)
	usuario := c.novoUsuario("Ana", false, false)

	designado, err := c.aprovador.EhDesignado(c.ctx, usuario)
	if err != nil {
		t.Fatalf("consulta falhou: %v", err)
	}
	if designado {
		t.Fatal("usuário saiu designado antes da designação")
	}

	id, err := c.aprovador.Designar(c.ctx, usuario)
	if err != nil {
		t.Fatalf("designação falhou: %v", err)
	}
	if id == uuid.Nil {
		t.Fatal("designação devolveu id zero")
	}

	salvo, err := c.aprovador.Obter(c.ctx, id)
	if err != nil {
		t.Fatalf("consulta por id falhou: %v", err)
	}
	if salvo.UsuarioID != usuario {
		t.Errorf("usuário = %s, esperado %s", salvo.UsuarioID, usuario)
	}

	designado, err = c.aprovador.EhDesignado(c.ctx, usuario)
	if err != nil {
		t.Fatalf("consulta falhou: %v", err)
	}
	if !designado {
		t.Error("usuário designado não foi reconhecido")
	}
}

func TestListarAprovadores(t *testing.T) {
	c := novoCenario(t)
	ana := c.novoUsuario("Ana", false, false)
	bruno := c.novoUsuario("Bruno", false, false)
	c.novoUsuario("Carla", false, false)

	c.designar(t, ana)
	c.designar(t, bruno)

	itens, err := c.aprovador.Listar(c.ctx, domain.PaginacaoFiltro{Page: 1, Size: 20})
	if err != nil {
		t.Fatalf("listagem falhou: %v", err)
	}
	if len(itens) != 2 {
		t.Errorf("%d aprovadores, esperado 2", len(itens))
	}
}

func TestRemoverAprovador(t *testing.T) {
	c := novoCenario(t)
	usuario := c.novoUsuario("Ana", false, false)
	id := c.designar(t, usuario)

	if err := c.aprovador.Remover(c.ctx, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("remoção inexistente = %v, esperado não encontrado", err)
	}

	if err := c.aprovador.Remover(c.ctx, id); err != nil {
		t.Fatalf("remoção falhou: %v", err)
	}

	designado, err := c.aprovador.EhDesignado(c.ctx, usuario)
	if err != nil {
		t.Fatalf("consulta falhou: %v", err)
	}
	if designado {
		t.Error("aprovador removido continuou designado")
	}
}

func TestDesignarComIdZeroRecebeErroDeValidacao(t *testing.T) {
	c := novoCenario(t)

	if _, err := c.aprovador.Designar(c.ctx, uuid.Nil); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("designação sem id = %v, esperado erro de validação", err)
	}
}
