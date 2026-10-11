package empresa_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	"github.com/google/uuid"
)

func dadosValidos(t *testing.T) domainempresa.DadosEmpresa {
	t.Helper()

	uf, err := domainempresa.NovaUF("pb")
	if err != nil {
		t.Fatalf("UF de teste inválida: %v", err)
	}
	regime, err := domainempresa.NovoRegimeTributario("Presumido")
	if err != nil {
		t.Fatalf("regime de teste inválido: %v", err)
	}

	return domainempresa.DadosEmpresa{
		RazaoSocial:      "  MIA Construtora LTDA ",
		NomeFantasia:     " MIA ",
		UF:               uf,
		RegimeTributario: regime,
	}
}

func cnpjValido(t *testing.T) domainempresa.CNPJ {
	t.Helper()

	cnpj, err := domainempresa.NovoCNPJ("11.222.333/0001-81")
	if err != nil {
		t.Fatalf("CNPJ de teste inválido: %v", err)
	}

	return cnpj
}

func TestNovaEmpresaNasceAtivaENormalizada(t *testing.T) {
	empresa, err := domainempresa.NovaEmpresa(cnpjValido(t), dadosValidos(t))
	if err != nil {
		t.Fatalf("criação falhou: %v", err)
	}

	if empresa.ID == uuid.Nil || !empresa.Ativa {
		t.Errorf("empresa %+v, esperado id preenchido e ativa", empresa)
	}
	if empresa.RazaoSocial != "MIA Construtora LTDA" || empresa.NomeFantasia != "MIA" {
		t.Errorf("nomes %q / %q, esperado sem espaços nas pontas", empresa.RazaoSocial, empresa.NomeFantasia)
	}
	if empresa.UF.String() != "PB" || empresa.RegimeTributario.String() != "presumido" {
		t.Errorf("uf %q regime %q, esperado PB presumido", empresa.UF, empresa.RegimeTributario)
	}
	if empresa.CriadoEm.IsZero() || !empresa.CriadoEm.Equal(empresa.AtualizadoEm) {
		t.Errorf("datas %v / %v, esperado iguais e preenchidas", empresa.CriadoEm, empresa.AtualizadoEm)
	}
}

func TestNovaEmpresaValidaDados(t *testing.T) {
	casos := []struct {
		nome   string
		ajuste func(*domainempresa.DadosEmpresa)
	}{
		{"razão social vazia", func(d *domainempresa.DadosEmpresa) { d.RazaoSocial = "  " }},
		{"razão social longa", func(d *domainempresa.DadosEmpresa) { d.RazaoSocial = strings.Repeat("a", 201) }},
		{"nome fantasia vazio", func(d *domainempresa.DadosEmpresa) { d.NomeFantasia = "" }},
		{"nome fantasia longo", func(d *domainempresa.DadosEmpresa) { d.NomeFantasia = strings.Repeat("ç", 121) }},
		{"uf ausente", func(d *domainempresa.DadosEmpresa) { d.UF = "" }},
		{"regime ausente", func(d *domainempresa.DadosEmpresa) { d.RegimeTributario = "" }},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			dados := dadosValidos(t)
			caso.ajuste(&dados)

			if _, err := domainempresa.NovaEmpresa(cnpjValido(t), dados); !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("erro %v, esperado erro de validação", err)
			}
		})
	}

	if _, err := domainempresa.NovaEmpresa(domainempresa.CNPJ{}, dadosValidos(t)); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("CNPJ vazio = %v, esperado erro de validação", err)
	}
}

func TestAlterarEmpresaPreservaCNPJ(t *testing.T) {
	empresa, err := domainempresa.NovaEmpresa(cnpjValido(t), dadosValidos(t))
	if err != nil {
		t.Fatalf("criação falhou: %v", err)
	}

	novos := dadosValidos(t)
	novos.NomeFantasia = "Gigante PB"
	depois := empresa.CriadoEm.Add(time.Hour)

	if err := empresa.Alterar(novos, depois); err != nil {
		t.Fatalf("alteração falhou: %v", err)
	}
	if empresa.NomeFantasia != "Gigante PB" || empresa.CNPJ.Digitos() != "11222333000181" {
		t.Errorf("empresa %+v, esperado nome novo e mesmo CNPJ", empresa)
	}
	if !empresa.AtualizadoEm.Equal(depois) {
		t.Errorf("atualizado_em %v, esperado %v", empresa.AtualizadoEm, depois)
	}

	invalidos := dadosValidos(t)
	invalidos.RazaoSocial = ""
	if err := empresa.Alterar(invalidos, depois); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("alteração inválida = %v, esperado erro de validação", err)
	}
	if empresa.RazaoSocial == "" {
		t.Error("alteração inválida mudou a empresa")
	}
}

func TestAtivarEDesativarEmpresa(t *testing.T) {
	empresa, err := domainempresa.NovaEmpresa(cnpjValido(t), dadosValidos(t))
	if err != nil {
		t.Fatalf("criação falhou: %v", err)
	}

	empresa.Desativar(time.Now().UTC())
	if empresa.Ativa {
		t.Error("empresa continuou ativa")
	}

	empresa.Ativar(time.Now().UTC())
	if !empresa.Ativa {
		t.Error("empresa continuou inativa")
	}
}

func TestUFERegimeRejeitamValoresForaDoEnum(t *testing.T) {
	for _, entrada := range []string{"", "XX", "Paraíba", "P"} {
		if _, err := domainempresa.NovaUF(entrada); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("NovaUF(%q) = %v, esperado erro de validação", entrada, err)
		}
	}
	for _, entrada := range []string{"", "lucro real", "mei"} {
		if _, err := domainempresa.NovoRegimeTributario(entrada); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("NovoRegimeTributario(%q) = %v, esperado erro de validação", entrada, err)
		}
	}
	for _, entrada := range []string{"ac", " SP ", "to"} {
		if _, err := domainempresa.NovaUF(entrada); err != nil {
			t.Errorf("NovaUF(%q) falhou: %v", entrada, err)
		}
	}
}
