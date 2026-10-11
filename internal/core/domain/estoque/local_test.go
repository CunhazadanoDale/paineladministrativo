package estoque_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

func TestNovoLocalNormalizaENasceAtivo(t *testing.T) {
	empresaID := uuid.New()

	local, err := domainestoque.NovoLocal(empresaID, "  Galpão Cabedelo  ")
	if err != nil {
		t.Fatalf("NovoLocal falhou: %v", err)
	}
	if local.ID == uuid.Nil || local.EmpresaID != empresaID {
		t.Errorf("local criado com id %s e empresa %s", local.ID, local.EmpresaID)
	}
	if local.Nome != "Galpão Cabedelo" {
		t.Errorf("nome = %q, esperado sem espaços nas pontas", local.Nome)
	}
	if !local.Ativo {
		t.Error("local novo deveria nascer ativo")
	}
}

func TestNovoLocalValidaEntrada(t *testing.T) {
	casos := []struct {
		nome      string
		empresaID uuid.UUID
		nomeLocal string
	}{
		{"sem empresa", uuid.Nil, "Galpão"},
		{"sem nome", uuid.New(), "   "},
		{"nome longo", uuid.New(), strings.Repeat("a", 121)},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			if _, err := domainestoque.NovoLocal(caso.empresaID, caso.nomeLocal); !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("NovoLocal = %v, esperado erro de validação", err)
			}
		})
	}

	if _, err := domainestoque.NovoLocal(uuid.New(), strings.Repeat("ã", 120)); err != nil {
		t.Errorf("nome com 120 caracteres acentuados = %v, esperado sucesso", err)
	}
}

func TestRenomearLocalPreservaEmpresa(t *testing.T) {
	local, err := domainestoque.NovoLocal(uuid.New(), "Galpão")
	if err != nil {
		t.Fatalf("NovoLocal falhou: %v", err)
	}
	empresaID := local.EmpresaID
	agora := time.Now().UTC().Add(time.Minute)

	if err := local.Renomear("", agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("renomear para vazio = %v, esperado erro de validação", err)
	}
	if err := local.Renomear("Galpão Norte", agora); err != nil {
		t.Fatalf("renomear falhou: %v", err)
	}
	if local.Nome != "Galpão Norte" || local.EmpresaID != empresaID || !local.AtualizadoEm.Equal(agora) {
		t.Errorf("local depois de renomear = %+v", local)
	}
}

func TestDesativarLocalExigeSaldoZerado(t *testing.T) {
	local, err := domainestoque.NovoLocal(uuid.New(), "Galpão")
	if err != nil {
		t.Fatalf("NovoLocal falhou: %v", err)
	}
	agora := time.Now().UTC()

	if err := local.Desativar(true, agora); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("desativar com saldo = %v, esperado erro de validação", err)
	}
	if !local.Ativo {
		t.Error("local com saldo não deveria ser desativado")
	}

	if err := local.Desativar(false, agora); err != nil {
		t.Fatalf("desativar sem saldo falhou: %v", err)
	}
	if local.Ativo {
		t.Error("local deveria estar inativo")
	}

	local.Ativar(agora)
	if !local.Ativo {
		t.Error("local deveria voltar a ficar ativo")
	}
}

func TestNovoCustoAceitaZeroERejeitaNegativo(t *testing.T) {
	custo, err := domainestoque.NovoCusto(0)
	if err != nil || custo.Centavos() != 0 {
		t.Errorf("NovoCusto(0) = %v, %v; esperado custo zero", custo, err)
	}
	if _, err := domainestoque.NovoCusto(-1); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("NovoCusto(-1) = %v, esperado erro de validação", err)
	}
}
