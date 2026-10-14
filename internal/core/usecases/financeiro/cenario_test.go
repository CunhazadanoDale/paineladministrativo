package financeiro_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	financeirousecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/financeiro"
	"github.com/google/uuid"
)

type cenario struct {
	ctx            context.Context
	contas         *financeirousecases.ContaBancariaUsecaseImpl
	categorias     *financeirousecases.CategoriaFinanceiraUsecaseImpl
	contasRepo     *repositorioContas
	categoriasRepo *repositorioCategorias
	empresasRepo   *repositorioEmpresas
	usuariosRepo   *repositorioUsuarios
	cargosRepo     *repositorioCargos
	mia            uuid.UUID
	red            uuid.UUID
	inativa        uuid.UUID
	administrador  uuid.UUID
	financeiroMIA  uuid.UUID
	financeiroRed  uuid.UUID
	semEmpresa     uuid.UUID
	comum          uuid.UUID
}

func novoCenario(t *testing.T) *cenario {
	t.Helper()

	empresas := novoRepositorioEmpresas()
	usuarios := &repositorioUsuarios{itens: map[uuid.UUID]*domainusuarios.Usuario{}}
	cargos := &repositorioCargos{itens: map[uuid.UUID]*domainusuarios.Cargo{}}
	contas := &repositorioContas{itens: map[uuid.UUID]*domainfinanceiro.ContaBancaria{}, empresas: empresas}
	categorias := &repositorioCategorias{itens: map[uuid.UUID]*domainfinanceiro.CategoriaFinanceira{}}

	c := &cenario{
		ctx:            context.Background(),
		contas:         financeirousecases.NewContaBancariaUsecase(contas, empresas, usuarios, cargos),
		categorias:     financeirousecases.NewCategoriaFinanceiraUsecase(categorias, usuarios, cargos),
		contasRepo:     contas,
		categoriasRepo: categorias,
		empresasRepo:   empresas,
		usuariosRepo:   usuarios,
		cargosRepo:     cargos,
	}

	c.mia = c.novaEmpresa("MIA", true)
	c.red = c.novaEmpresa("Red Diamond", true)
	c.inativa = c.novaEmpresa("Gigante SP", false)
	c.administrador = c.novoUsuario("Ana Admin", true, false, domainempresa.AcessoTotal())
	c.financeiroMIA = c.novoUsuario("Fábio MIA", false, true, domainempresa.AcessoRestrito([]uuid.UUID{c.mia, c.inativa}))
	c.financeiroRed = c.novoUsuario("Rita Red", false, true, domainempresa.AcessoRestrito([]uuid.UUID{c.red}))
	c.semEmpresa = c.novoUsuario("Sérgio Sem Empresa", false, true, domainempresa.AcessoRestrito(nil))
	c.comum = c.novoUsuario("Carla Comum", false, false, domainempresa.AcessoTotal())

	return c
}

func (c *cenario) novaEmpresa(nome string, ativa bool) uuid.UUID {
	id := uuid.New()
	c.empresasRepo.itens[id] = &domainempresa.Empresa{ID: id, NomeFantasia: nome, Ativa: ativa}

	return id
}

func (c *cenario) novoUsuario(nome string, administrador, financeiro bool, acesso domainempresa.Acesso) uuid.UUID {
	cargoID := uuid.New()
	c.cargosRepo.itens[cargoID] = &domainusuarios.Cargo{
		ID:            cargoID,
		Nome:          "Cargo de " + nome,
		Ativo:         true,
		Administrador: administrador,
		Financeiro:    financeiro,
	}

	usuarioID := uuid.New()
	c.usuariosRepo.itens[usuarioID] = &domainusuarios.Usuario{ID: usuarioID, Nome: nome, CargoID: cargoID, Ativo: true}
	c.empresasRepo.acessos[usuarioID] = acesso

	return usuarioID
}

func esperarErro(t *testing.T, err error, sentinela error, trecho string) {
	t.Helper()

	if !errors.Is(err, sentinela) || !strings.Contains(err.Error(), trecho) {
		t.Fatalf("erro = %v; esperado %v com %q", err, sentinela, trecho)
	}
}

func texto(valor string) *string {
	return &valor
}
