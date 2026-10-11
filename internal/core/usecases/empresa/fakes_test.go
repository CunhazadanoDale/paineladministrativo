package empresa_test

import (
	"context"
	"sort"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/empresa"
	"github.com/google/uuid"
)

var (
	_ portsout.EmpresaRepository = (*repositorioEmpresas)(nil)
	_ portsout.UsuarioRepository = (*repositorioUsuarios)(nil)
	_ portsout.CargoRepository   = (*repositorioCargos)(nil)
)

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
	itens             map[uuid.UUID]*domainempresa.Empresa
	vinculos          map[uuid.UUID]domainempresa.Vinculos
	usuarios          *repositorioUsuarios
	cargos            *repositorioCargos
	conflitoNaCriacao bool
}

func (r *repositorioEmpresas) Criar(_ context.Context, empresa *domainempresa.Empresa) (uuid.UUID, error) {
	if r.conflitoNaCriacao {
		return uuid.Nil, domain.ErroConflito("já existe um registro com estes dados")
	}

	copia := *empresa
	r.itens[empresa.ID] = &copia

	return empresa.ID, nil
}

func (r *repositorioEmpresas) Obter(_ context.Context, id uuid.UUID) (*domainempresa.Empresa, error) {
	empresa, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *empresa
	return &copia, nil
}

func (r *repositorioEmpresas) ObterPorCNPJ(_ context.Context, cnpj domainempresa.CNPJ) (*domainempresa.Empresa, error) {
	for _, empresa := range r.itens {
		if empresa.CNPJ.Digitos() == cnpj.Digitos() {
			copia := *empresa
			return &copia, nil
		}
	}

	return nil, nil
}

func (r *repositorioEmpresas) Listar(_ context.Context, filtro portsout.EmpresaFiltro) ([]*domainempresa.Empresa, error) {
	permitidas := map[uuid.UUID]bool{}
	for _, id := range filtro.EmpresaIDs {
		permitidas[id] = true
	}

	itens := make([]*domainempresa.Empresa, 0, len(r.itens))
	for _, empresa := range r.itens {
		if filtro.Ativa != nil && empresa.Ativa != *filtro.Ativa {
			continue
		}
		if filtro.EmpresaIDs != nil && !permitidas[empresa.ID] {
			continue
		}
		copia := *empresa
		itens = append(itens, &copia)
	}

	sort.Slice(itens, func(i, j int) bool { return itens[i].NomeFantasia < itens[j].NomeFantasia })

	return itens, nil
}

func (r *repositorioEmpresas) Atualizar(_ context.Context, empresa *domainempresa.Empresa) (bool, error) {
	if _, ok := r.itens[empresa.ID]; !ok {
		return false, nil
	}

	copia := *empresa
	r.itens[empresa.ID] = &copia

	return true, nil
}

func (r *repositorioEmpresas) Remover(_ context.Context, id uuid.UUID) (bool, error) {
	if _, ok := r.itens[id]; !ok {
		return false, nil
	}
	for _, vinculos := range r.vinculos {
		for _, empresaID := range vinculos.EmpresaIDs {
			if empresaID == id {
				return false, domain.ErroValidacao("registro em uso por outros dados e não pode ser excluído")
			}
		}
	}

	delete(r.itens, id)

	return true, nil
}

func (r *repositorioEmpresas) AcessoDoUsuario(_ context.Context, usuarioID uuid.UUID) (domainempresa.Acesso, error) {
	usuario, ok := r.usuarios.itens[usuarioID]
	if !ok {
		return domainempresa.Acesso{}, domain.ErroNaoEncontrado("usuário não encontrado")
	}

	vinculos := r.vinculos[usuarioID]
	if cargo, ok := r.cargos.itens[usuario.CargoID]; (ok && cargo.Administrador) || vinculos.AcessoTodas {
		return domainempresa.AcessoTotal(), nil
	}

	return domainempresa.AcessoRestrito(vinculos.EmpresaIDs), nil
}

func (r *repositorioEmpresas) VinculosDoUsuario(_ context.Context, usuarioID uuid.UUID) (*domainempresa.Vinculos, error) {
	if _, ok := r.usuarios.itens[usuarioID]; !ok {
		return nil, nil
	}

	vinculos := r.vinculos[usuarioID]

	return &vinculos, nil
}

func (r *repositorioEmpresas) DefinirVinculos(_ context.Context, usuarioID uuid.UUID, vinculos domainempresa.Vinculos) error {
	if _, ok := r.usuarios.itens[usuarioID]; !ok {
		return domain.ErroNaoEncontrado("usuário não encontrado")
	}

	r.vinculos[usuarioID] = vinculos

	return nil
}
