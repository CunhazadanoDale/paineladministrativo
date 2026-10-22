package dre_test

import (
	"context"
	"errors"
	"testing"

	domaindre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/dre"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/dre"
	"github.com/google/uuid"
)

var (
	_ portsout.UsuarioRepository = (*repositorioUsuarios)(nil)
	_ portsout.CargoRepository   = (*repositorioCargos)(nil)
	_ portsout.EmpresaRepository = (*repositorioEmpresas)(nil)
	_ portsout.ResumoRepository  = (*repositorioResumos)(nil)
)

var errBanco = errors.New("banco fora do ar")

type repositorioUsuarios struct {
	itens map[uuid.UUID]*domainusuarios.Usuario
}

func (r *repositorioUsuarios) Obter(_ context.Context, id uuid.UUID) (*domainusuarios.Usuario, error) {
	usuario, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *usuario
	return &copia, nil
}

type repositorioCargos struct {
	itens map[uuid.UUID]*domainusuarios.Cargo
}

func (r *repositorioCargos) Obter(_ context.Context, id uuid.UUID) (*domainusuarios.Cargo, error) {
	cargo, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *cargo
	return &copia, nil
}

type repositorioEmpresas struct {
	acessos map[uuid.UUID]domainempresa.Acesso
}

func (r *repositorioEmpresas) AcessoDoUsuario(_ context.Context, usuarioID uuid.UUID) (domainempresa.Acesso, error) {
	if acesso, ok := r.acessos[usuarioID]; ok {
		return acesso, nil
	}

	return domainempresa.AcessoRestrito(nil), nil
}

type repositorioResumos struct {
	lancamentos  []domaindre.Lancamento
	semCusto     []domaindre.ConsumoSemCusto
	faturamentos []domaindre.Faturamento
	erro         error
	filtros      []portsout.FiltroResumo
}

func (r *repositorioResumos) Lancamentos(_ context.Context, filtro portsout.FiltroResumo) ([]domaindre.Lancamento, error) {
	r.filtros = append(r.filtros, filtro)

	return r.lancamentos, r.erro
}

func (r *repositorioResumos) ConsumosSemCusto(_ context.Context, filtro portsout.FiltroResumo) ([]domaindre.ConsumoSemCusto, error) {
	r.filtros = append(r.filtros, filtro)

	return r.semCusto, r.erro
}

func (r *repositorioResumos) Faturamento(_ context.Context, filtro portsout.FiltroResumo) ([]domaindre.Faturamento, error) {
	r.filtros = append(r.filtros, filtro)

	return r.faturamentos, r.erro
}

type cenario struct {
	usuarios      *repositorioUsuarios
	cargos        *repositorioCargos
	empresas      *repositorioEmpresas
	resumos       *repositorioResumos
	financeiro    uuid.UUID
	administrador uuid.UUID
	comercial     uuid.UUID
	restrito      uuid.UUID
	empresaA      uuid.UUID
	empresaB      uuid.UUID
}

func novoCenario(t *testing.T) *cenario {
	t.Helper()

	c := &cenario{
		usuarios:      &repositorioUsuarios{itens: map[uuid.UUID]*domainusuarios.Usuario{}},
		cargos:        &repositorioCargos{itens: map[uuid.UUID]*domainusuarios.Cargo{}},
		empresas:      &repositorioEmpresas{acessos: map[uuid.UUID]domainempresa.Acesso{}},
		resumos:       &repositorioResumos{},
		financeiro:    uuid.New(),
		administrador: uuid.New(),
		comercial:     uuid.New(),
		restrito:      uuid.New(),
		empresaA:      uuid.New(),
		empresaB:      uuid.New(),
	}

	c.usuarioComCargo(c.financeiro, domainusuarios.Cargo{Financeiro: true}, domainempresa.AcessoTotal())
	c.usuarioComCargo(c.administrador, domainusuarios.Cargo{Administrador: true}, domainempresa.AcessoTotal())
	c.usuarioComCargo(c.comercial, domainusuarios.Cargo{Comercial: true}, domainempresa.AcessoTotal())
	c.usuarioComCargo(c.restrito, domainusuarios.Cargo{Financeiro: true}, domainempresa.AcessoRestrito([]uuid.UUID{c.empresaA}))

	return c
}

func (c *cenario) usuarioComCargo(usuarioID uuid.UUID, cargo domainusuarios.Cargo, acesso domainempresa.Acesso) {
	cargo.ID = uuid.New()
	c.cargos.itens[cargo.ID] = &cargo
	c.usuarios.itens[usuarioID] = &domainusuarios.Usuario{ID: usuarioID, CargoID: cargo.ID, Ativo: true}
	c.empresas.acessos[usuarioID] = acesso
}
