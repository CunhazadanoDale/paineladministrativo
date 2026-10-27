package postgres_test

import (
	"context"
	"testing"
)

func TestLeadContatosCadastradosIgnoraInativoEComparaNormalizado(t *testing.T) {
	repo, banco, _, etapas := cenario(t)
	ctx := context.Background()

	ativo := novoLead(etapas[0], "Ativo")
	ativo.Email = "Contato@Aguia.com.br"
	ativo.Telefone = "(83) 99999-0000"
	inativo := novoLead(etapas[0], "Inativo")
	inativo.Email = "inativo@exemplo.com"
	inativo.Telefone = "83 3222-1111"
	for _, lead := range []any{ativo, inativo} {
		if _, err := banco.NamedExecContext(ctx, `
			INSERT INTO lead (id, nome, email, telefone, ativo, origem, criado_em, atualizado_em, etapa_id)
			VALUES (:id, :nome, :email, :telefone, :ativo, :origem, :criado_em, :atualizado_em, :etapa_id)
		`, lead); err != nil {
			t.Fatalf("inserção do lead falhou: %v", err)
		}
	}
	if _, err := banco.ExecContext(ctx, `UPDATE lead SET ativo = FALSE WHERE id = $1`, inativo.ID); err != nil {
		t.Fatalf("desativação falhou: %v", err)
	}

	emails, err := repo.EmailsCadastrados(ctx, []string{"contato@aguia.com.br", "inativo@exemplo.com", "novo@exemplo.com"})
	if err != nil {
		t.Fatalf("busca de emails falhou: %v", err)
	}
	if len(emails) != 1 || !emails["contato@aguia.com.br"] {
		t.Errorf("emails cadastrados = %v", emails)
	}

	telefones, err := repo.TelefonesCadastrados(ctx, []string{"83999990000", "8332221111", "11999990000"})
	if err != nil {
		t.Fatalf("busca de telefones falhou: %v", err)
	}
	if len(telefones) != 1 || !telefones["83999990000"] {
		t.Errorf("telefones cadastrados = %v", telefones)
	}

	vazio, err := repo.EmailsCadastrados(ctx, nil)
	if err != nil || len(vazio) != 0 {
		t.Errorf("lista vazia = %v, %v", vazio, err)
	}
}
