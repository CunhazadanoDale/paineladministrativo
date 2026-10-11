package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	portsoutempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/empresa"
	"github.com/google/uuid"
)

type cenarioEmpresas struct {
	ctx      context.Context
	empresas *postgres.EmpresaRepository
	usuarios *postgres.UsuarioRepository
	cargos   *postgres.CargoRepository
}

func novoCenarioEmpresas(t *testing.T) *cenarioEmpresas {
	t.Helper()

	usuarios, cargos, banco := cenarioUsuarios(t)

	return &cenarioEmpresas{
		ctx:      context.Background(),
		empresas: postgres.NewEmpresaRepository(banco),
		usuarios: usuarios,
		cargos:   cargos,
	}
}

func (c *cenarioEmpresas) criarEmpresa(t *testing.T, cnpj, nomeFantasia string) *domainempresa.Empresa {
	t.Helper()

	documento, err := domainempresa.NovoCNPJ(cnpj)
	if err != nil {
		t.Fatalf("CNPJ de teste inválido: %v", err)
	}

	empresa, err := domainempresa.NovaEmpresa(documento, domainempresa.DadosEmpresa{
		RazaoSocial:      nomeFantasia + " LTDA",
		NomeFantasia:     nomeFantasia,
		UF:               domainempresa.UFDe("PB"),
		RegimeTributario: domainempresa.RegimePresumido,
	})
	if err != nil {
		t.Fatalf("empresa de teste inválida: %v", err)
	}

	if _, err := c.empresas.Criar(c.ctx, empresa); err != nil {
		t.Fatalf("criação da empresa falhou: %v", err)
	}

	return empresa
}

func (c *cenarioEmpresas) criarUsuario(t *testing.T, email string, administrador bool) uuid.UUID {
	t.Helper()

	cargo := novoCargo("Cargo " + email)
	cargo.Administrador = administrador
	if _, err := c.cargos.Criar(c.ctx, cargo); err != nil {
		t.Fatalf("criação do cargo falhou: %v", err)
	}

	usuario := novoUsuario(cargo.ID, "Usuário "+email, email)
	if _, err := c.usuarios.Criar(c.ctx, usuario); err != nil {
		t.Fatalf("criação do usuário falhou: %v", err)
	}

	return usuario.ID
}

func TestEmpresaCriaObtemEListaComFiltros(t *testing.T) {
	c := novoCenarioEmpresas(t)
	mia := c.criarEmpresa(t, "11.222.333/0001-81", "MIA")
	red := c.criarEmpresa(t, "33.400.004/0001-16", "Red Diamond")

	red.Desativar(time.Now().UTC())
	if alterada, err := c.empresas.Atualizar(c.ctx, red); err != nil || !alterada {
		t.Fatalf("desativação = %v, %v", alterada, err)
	}

	salva, err := c.empresas.Obter(c.ctx, mia.ID)
	if err != nil || salva == nil {
		t.Fatalf("busca por id = %v, %v", salva, err)
	}
	if salva.CNPJ.Digitos() != "11222333000181" || salva.UF.String() != "PB" || salva.RegimeTributario != domainempresa.RegimePresumido {
		t.Errorf("empresa salva %+v diferente da criada", salva)
	}

	porCNPJ, err := c.empresas.ObterPorCNPJ(c.ctx, domainempresa.CNPJDe("11222333000181"))
	if err != nil || porCNPJ == nil || porCNPJ.ID != mia.ID {
		t.Errorf("busca por CNPJ = %v, %v", porCNPJ, err)
	}

	ativa := true
	casos := []struct {
		nome     string
		filtro   portsoutempresa.EmpresaFiltro
		esperado []uuid.UUID
	}{
		{"todas", portsoutempresa.EmpresaFiltro{}, []uuid.UUID{mia.ID, red.ID}},
		{"só ativas", portsoutempresa.EmpresaFiltro{Ativa: &ativa}, []uuid.UUID{mia.ID}},
		{"restrita a uma", portsoutempresa.EmpresaFiltro{EmpresaIDs: []uuid.UUID{red.ID}}, []uuid.UUID{red.ID}},
		{"acesso vazio", portsoutempresa.EmpresaFiltro{EmpresaIDs: []uuid.UUID{}}, nil},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			itens, err := c.empresas.Listar(c.ctx, caso.filtro)
			if err != nil {
				t.Fatalf("listagem falhou: %v", err)
			}
			if len(itens) != len(caso.esperado) {
				t.Fatalf("%d empresas, esperado %d", len(itens), len(caso.esperado))
			}
			for i, item := range itens {
				if item.ID != caso.esperado[i] {
					t.Errorf("posição %d = %s, esperado %s", i, item.NomeFantasia, caso.esperado[i])
				}
			}
		})
	}
}

func TestEmpresaComCNPJRepetidoRecebeConflito(t *testing.T) {
	c := novoCenarioEmpresas(t)
	c.criarEmpresa(t, "11.222.333/0001-81", "MIA")

	documento, err := domainempresa.NovoCNPJ("11222333000181")
	if err != nil {
		t.Fatalf("CNPJ inválido: %v", err)
	}
	repetida, err := domainempresa.NovaEmpresa(documento, domainempresa.DadosEmpresa{
		RazaoSocial:      "Outra",
		NomeFantasia:     "Outra",
		UF:               domainempresa.UFDe("SP"),
		RegimeTributario: domainempresa.RegimeReal,
	})
	if err != nil {
		t.Fatalf("empresa inválida: %v", err)
	}

	if _, err := c.empresas.Criar(c.ctx, repetida); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("CNPJ repetido = %v, esperado conflito", err)
	}
}

func TestAcessoDoUsuarioConsideraAdministradorAcessoTotalEVinculos(t *testing.T) {
	c := novoCenarioEmpresas(t)
	mia := c.criarEmpresa(t, "11.222.333/0001-81", "MIA")
	red := c.criarEmpresa(t, "33.400.004/0001-16", "Red Diamond")

	administrador := c.criarUsuario(t, "diretoria@exemplo.com", true)
	total := c.criarUsuario(t, "total@exemplo.com", false)
	restrito := c.criarUsuario(t, "restrito@exemplo.com", false)
	semVinculo := c.criarUsuario(t, "sem@exemplo.com", false)

	if err := c.empresas.DefinirVinculos(c.ctx, total, domainempresa.Vinculos{AcessoTodas: true}); err != nil {
		t.Fatalf("vínculo total falhou: %v", err)
	}
	if err := c.empresas.DefinirVinculos(c.ctx, restrito, domainempresa.Vinculos{EmpresaIDs: []uuid.UUID{mia.ID}}); err != nil {
		t.Fatalf("vínculo restrito falhou: %v", err)
	}

	for _, usuario := range []uuid.UUID{administrador, total} {
		acesso, err := c.empresas.AcessoDoUsuario(c.ctx, usuario)
		if err != nil || !acesso.Todas() {
			t.Errorf("acesso de %s = %+v, %v; esperado total", usuario, acesso, err)
		}
	}

	acesso, err := c.empresas.AcessoDoUsuario(c.ctx, restrito)
	if err != nil {
		t.Fatalf("acesso restrito falhou: %v", err)
	}
	if acesso.Todas() || !acesso.Permite(mia.ID) || acesso.Permite(red.ID) {
		t.Errorf("acesso restrito %+v, esperado só a MIA", acesso)
	}

	vazio, err := c.empresas.AcessoDoUsuario(c.ctx, semVinculo)
	if err != nil {
		t.Fatalf("acesso sem vínculo falhou: %v", err)
	}
	if vazio.Todas() || len(vazio.EmpresaIDs()) != 0 {
		t.Errorf("acesso sem vínculo %+v, esperado nenhuma empresa", vazio)
	}

	if _, err := c.empresas.AcessoDoUsuario(c.ctx, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("usuário inexistente = %v, esperado não encontrado", err)
	}
}

func TestDefinirVinculosSubstituiOConjunto(t *testing.T) {
	c := novoCenarioEmpresas(t)
	mia := c.criarEmpresa(t, "11.222.333/0001-81", "MIA")
	red := c.criarEmpresa(t, "33.400.004/0001-16", "Red Diamond")
	ce := c.criarEmpresa(t, "44.555.666/0001-81", "Gigante CE")
	usuario := c.criarUsuario(t, "ana@exemplo.com", false)

	if err := c.empresas.DefinirVinculos(c.ctx, usuario, domainempresa.Vinculos{EmpresaIDs: []uuid.UUID{mia.ID, red.ID}}); err != nil {
		t.Fatalf("primeiro vínculo falhou: %v", err)
	}
	if err := c.empresas.DefinirVinculos(c.ctx, usuario, domainempresa.Vinculos{EmpresaIDs: []uuid.UUID{ce.ID, ce.ID}}); err != nil {
		t.Fatalf("substituição falhou: %v", err)
	}

	vinculos, err := c.empresas.VinculosDoUsuario(c.ctx, usuario)
	if err != nil || vinculos == nil {
		t.Fatalf("consulta de vínculos = %v, %v", vinculos, err)
	}
	if vinculos.AcessoTodas || len(vinculos.EmpresaIDs) != 1 || vinculos.EmpresaIDs[0] != ce.ID {
		t.Errorf("vínculos %+v, esperado só a Gigante CE", vinculos)
	}

	if err := c.empresas.DefinirVinculos(c.ctx, usuario, domainempresa.Vinculos{EmpresaIDs: []uuid.UUID{uuid.New()}}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("empresa inexistente = %v, esperado erro de validação", err)
	}
	if depois, err := c.empresas.VinculosDoUsuario(c.ctx, usuario); err != nil || len(depois.EmpresaIDs) != 1 {
		t.Errorf("falha na gravação alterou os vínculos: %+v, %v", depois, err)
	}

	if err := c.empresas.DefinirVinculos(c.ctx, uuid.New(), domainempresa.Vinculos{}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("usuário inexistente = %v, esperado não encontrado", err)
	}
	if inexistente, err := c.empresas.VinculosDoUsuario(c.ctx, uuid.New()); err != nil || inexistente != nil {
		t.Errorf("vínculos de usuário inexistente = %v, %v; esperado nil", inexistente, err)
	}
}

func TestRemoverEmpresaRespeitaVinculos(t *testing.T) {
	c := novoCenarioEmpresas(t)
	livre := c.criarEmpresa(t, "11.222.333/0001-81", "MIA")
	vinculada := c.criarEmpresa(t, "33.400.004/0001-16", "Red Diamond")
	usuario := c.criarUsuario(t, "ana@exemplo.com", false)

	if err := c.empresas.DefinirVinculos(c.ctx, usuario, domainempresa.Vinculos{EmpresaIDs: []uuid.UUID{vinculada.ID}}); err != nil {
		t.Fatalf("vínculo falhou: %v", err)
	}

	if removida, err := c.empresas.Remover(c.ctx, vinculada.ID); !errors.Is(err, domain.ErrValidacao) || removida {
		t.Errorf("remoção de empresa vinculada = %v, %v; esperado erro de validação", removida, err)
	}
	if removida, err := c.empresas.Remover(c.ctx, livre.ID); err != nil || !removida {
		t.Errorf("remoção de empresa livre = %v, %v", removida, err)
	}
	if removida, err := c.empresas.Remover(c.ctx, uuid.New()); err != nil || removida {
		t.Errorf("remoção de empresa inexistente = %v, %v; esperado false sem erro", removida, err)
	}
}
