package receber_test

import (
	"context"
	"slices"
	"sort"

	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/receber"
	"github.com/google/uuid"
)

type memoria struct {
	usuarios   map[uuid.UUID]*domainusuarios.Usuario
	cargos     map[uuid.UUID]*domainusuarios.Cargo
	empresas   map[uuid.UUID]*domainempresa.Empresa
	acessos    map[uuid.UUID]domainempresa.Acesso
	clientes   map[uuid.UUID]*domaincliente.Cliente
	contas     map[uuid.UUID]*domainfinanceiro.ContaBancaria
	categorias map[uuid.UUID]*domainfinanceiro.CategoriaFinanceira
	titulos    map[uuid.UUID]*domainreceber.Titulo
	historicos []*domainreceber.Historico
	alteracoes []*portsout.Alteracao
	filtros    []portsout.TituloFiltro
}

func novaMemoria() *memoria {
	return &memoria{
		usuarios:   map[uuid.UUID]*domainusuarios.Usuario{},
		cargos:     map[uuid.UUID]*domainusuarios.Cargo{},
		empresas:   map[uuid.UUID]*domainempresa.Empresa{},
		acessos:    map[uuid.UUID]domainempresa.Acesso{},
		clientes:   map[uuid.UUID]*domaincliente.Cliente{},
		contas:     map[uuid.UUID]*domainfinanceiro.ContaBancaria{},
		categorias: map[uuid.UUID]*domainfinanceiro.CategoriaFinanceira{},
		titulos:    map[uuid.UUID]*domainreceber.Titulo{},
	}
}

func copiar[T any](item *T) *T {
	if item == nil {
		return nil
	}
	copia := *item

	return &copia
}

type repoUsuarios struct{ *memoria }

func (r repoUsuarios) Obter(_ context.Context, id uuid.UUID) (*domainusuarios.Usuario, error) {
	return copiar(r.usuarios[id]), nil
}

type repoCargos struct{ *memoria }

func (r repoCargos) Obter(_ context.Context, id uuid.UUID) (*domainusuarios.Cargo, error) {
	return copiar(r.cargos[id]), nil
}

type repoEmpresas struct{ *memoria }

func (r repoEmpresas) Obter(_ context.Context, id uuid.UUID) (*domainempresa.Empresa, error) {
	return copiar(r.empresas[id]), nil
}

func (r repoEmpresas) AcessoDoUsuario(_ context.Context, usuarioID uuid.UUID) (domainempresa.Acesso, error) {
	return r.acessos[usuarioID], nil
}

type repoClientes struct{ *memoria }

func (r repoClientes) Obter(_ context.Context, id uuid.UUID) (*domaincliente.Cliente, error) {
	return copiar(r.clientes[id]), nil
}

type repoContas struct{ *memoria }

func (r repoContas) Obter(_ context.Context, id uuid.UUID) (*domainfinanceiro.ContaBancaria, error) {
	return copiar(r.contas[id]), nil
}

type repoCategorias struct{ *memoria }

func (r repoCategorias) Obter(_ context.Context, id uuid.UUID) (*domainfinanceiro.CategoriaFinanceira, error) {
	return copiar(r.categorias[id]), nil
}

type repoTitulos struct{ *memoria }

func (r repoTitulos) Criar(_ context.Context, titulo *domainreceber.Titulo, historico *domainreceber.Historico) error {
	r.titulos[titulo.ID] = copiar(titulo)
	r.historicos = append(r.historicos, historico)

	return nil
}

func (r repoTitulos) Atualizar(_ context.Context, alteracao *portsout.Alteracao) error {
	atual, ok := r.titulos[alteracao.Titulo.ID]
	if !ok {
		return domainreceber.ErroTituloNaoEncontrado()
	}
	if atual.Versao != alteracao.VersaoLida || atual.Status != alteracao.StatusAnterior {
		return domainreceber.ErroVersaoDivergente()
	}

	alteracao.Titulo.Versao = atual.Versao + 1
	r.titulos[alteracao.Titulo.ID] = copiar(alteracao.Titulo)
	r.alteracoes = append(r.alteracoes, alteracao)
	if alteracao.Historico != nil {
		r.historicos = append(r.historicos, alteracao.Historico)
	}

	return nil
}

func (r repoTitulos) Obter(_ context.Context, id uuid.UUID) (*domainreceber.Titulo, error) {
	return copiar(r.titulos[id]), nil
}

func (r repoTitulos) Listar(_ context.Context, filtro portsout.TituloFiltro) ([]*domainreceber.Titulo, error) {
	r.memoria.filtros = append(r.memoria.filtros, filtro)

	titulos := make([]*domainreceber.Titulo, 0)
	for _, titulo := range r.titulos {
		if filtro.Empresa != nil && titulo.EmpresaID != *filtro.Empresa {
			continue
		}
		if filtro.EmpresaIDs != nil && !slices.Contains(filtro.EmpresaIDs, titulo.EmpresaID) {
			continue
		}
		if filtro.Status != "" && titulo.Status != filtro.Status {
			continue
		}
		if filtro.VencidosEm != nil && !titulo.Vencido(*filtro.VencidosEm) {
			continue
		}
		titulos = append(titulos, copiar(titulo))
	}
	sort.Slice(titulos, func(i, j int) bool { return titulos[i].Vencimento.Before(titulos[j].Vencimento) })

	return titulos, nil
}

func (r repoTitulos) ListarHistorico(_ context.Context, tituloID uuid.UUID) ([]*domainreceber.Historico, error) {
	historicos := make([]*domainreceber.Historico, 0)
	for _, historico := range r.historicos {
		if historico.TituloID == tituloID {
			historicos = append(historicos, historico)
		}
	}

	return historicos, nil
}
