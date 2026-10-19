package cliente_test

import (
	"context"
	"slices"
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainlead "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/cliente"
	"github.com/google/uuid"
)

var (
	_ portsout.UsuarioRepository       = (*repositorioUsuarios)(nil)
	_ portsout.CargoRepository         = (*repositorioCargos)(nil)
	_ portsout.EmpresaRepository       = (*repositorioEmpresas)(nil)
	_ portsout.GrupoRepository         = (*repositorioGrupos)(nil)
	_ portsout.RepresentanteRepository = (*repositorioRepresentantes)(nil)
	_ portsout.ClienteRepository       = (*repositorioClientes)(nil)
	_ portsout.ObraRepository          = (*repositorioObras)(nil)
	_ portsout.LeadRepository          = (*repositorioLeads)(nil)
)

type repositorioLeads struct {
	itens map[uuid.UUID]*domainlead.Lead
}

func (r *repositorioLeads) Obter(_ context.Context, id uuid.UUID) (*domainlead.Lead, error) {
	if lead, ok := r.itens[id]; ok {
		return copiar(lead), nil
	}

	return nil, nil
}

func conflito() error {
	return domain.ErroConflito("já existe um registro com estes dados")
}

func copiar[T any](item *T) *T {
	copia := *item
	return &copia
}

type repositorioUsuarios struct {
	itens map[uuid.UUID]*domainusuarios.Usuario
}

func (r *repositorioUsuarios) Obter(_ context.Context, id uuid.UUID) (*domainusuarios.Usuario, error) {
	if usuario, ok := r.itens[id]; ok {
		return copiar(usuario), nil
	}

	return nil, nil
}

type repositorioCargos struct {
	itens map[uuid.UUID]*domainusuarios.Cargo
}

func (r *repositorioCargos) Obter(_ context.Context, id uuid.UUID) (*domainusuarios.Cargo, error) {
	if cargo, ok := r.itens[id]; ok {
		return copiar(cargo), nil
	}

	return nil, nil
}

type repositorioEmpresas struct {
	itens map[uuid.UUID]*domainempresa.Empresa
}

func (r *repositorioEmpresas) Obter(_ context.Context, id uuid.UUID) (*domainempresa.Empresa, error) {
	if empresa, ok := r.itens[id]; ok {
		return copiar(empresa), nil
	}

	return nil, nil
}

type repositorioGrupos struct {
	itens map[uuid.UUID]*domaincliente.Grupo
}

func (r *repositorioGrupos) repetido(grupo *domaincliente.Grupo) bool {
	for _, existente := range r.itens {
		if existente.ID != grupo.ID && existente.NomeBusca == grupo.NomeBusca {
			return true
		}
	}

	return false
}

func (r *repositorioGrupos) Criar(_ context.Context, grupo *domaincliente.Grupo) (uuid.UUID, error) {
	if r.repetido(grupo) {
		return uuid.Nil, conflito()
	}
	r.itens[grupo.ID] = copiar(grupo)

	return grupo.ID, nil
}

func (r *repositorioGrupos) Obter(_ context.Context, id uuid.UUID) (*domaincliente.Grupo, error) {
	if grupo, ok := r.itens[id]; ok {
		return copiar(grupo), nil
	}

	return nil, nil
}

func (r *repositorioGrupos) ObterPorNomeBusca(_ context.Context, nomeBusca string) (*domaincliente.Grupo, error) {
	for _, grupo := range r.itens {
		if grupo.NomeBusca == nomeBusca {
			return copiar(grupo), nil
		}
	}

	return nil, nil
}

func (r *repositorioGrupos) Listar(_ context.Context, filtro portsout.GrupoFiltro) ([]*domaincliente.Grupo, error) {
	termo := domaincliente.NomeDeBusca(filtro.Busca)
	itens := make([]*domaincliente.Grupo, 0)
	for _, grupo := range r.itens {
		if (termo == "" || strings.Contains(grupo.NomeBusca, termo)) && (filtro.Ativo == nil || grupo.Ativo == *filtro.Ativo) {
			itens = append(itens, copiar(grupo))
		}
	}
	slices.SortFunc(itens, func(a, b *domaincliente.Grupo) int { return strings.Compare(a.NomeBusca, b.NomeBusca) })

	return itens, nil
}

func (r *repositorioGrupos) Atualizar(_ context.Context, grupo *domaincliente.Grupo) (bool, error) {
	if _, ok := r.itens[grupo.ID]; !ok {
		return false, nil
	}
	if r.repetido(grupo) {
		return false, conflito()
	}
	r.itens[grupo.ID] = copiar(grupo)

	return true, nil
}

type repositorioRepresentantes struct {
	itens map[uuid.UUID]*domaincliente.Representante
}

func (r *repositorioRepresentantes) repetido(representante *domaincliente.Representante) bool {
	for _, existente := range r.itens {
		if existente.ID == representante.ID {
			continue
		}
		if representante.Documento != nil && existente.Documento != nil && *existente.Documento == *representante.Documento {
			return true
		}
		if representante.UsuarioID != nil && existente.UsuarioID != nil && *existente.UsuarioID == *representante.UsuarioID {
			return true
		}
	}

	return false
}

func (r *repositorioRepresentantes) Criar(_ context.Context, representante *domaincliente.Representante) (uuid.UUID, error) {
	if r.repetido(representante) {
		return uuid.Nil, conflito()
	}
	r.itens[representante.ID] = copiar(representante)

	return representante.ID, nil
}

func (r *repositorioRepresentantes) Obter(_ context.Context, id uuid.UUID) (*domaincliente.Representante, error) {
	if representante, ok := r.itens[id]; ok {
		return copiar(representante), nil
	}

	return nil, nil
}

func (r *repositorioRepresentantes) ObterPorDocumento(_ context.Context, documento string) (*domaincliente.Representante, error) {
	for _, representante := range r.itens {
		if representante.Documento != nil && representante.Documento.Valor() == documento {
			return copiar(representante), nil
		}
	}

	return nil, nil
}

func (r *repositorioRepresentantes) ObterPorUsuario(_ context.Context, usuarioID uuid.UUID) (*domaincliente.Representante, error) {
	for _, representante := range r.itens {
		if representante.UsuarioID != nil && *representante.UsuarioID == usuarioID {
			return copiar(representante), nil
		}
	}

	return nil, nil
}

func (r *repositorioRepresentantes) Listar(_ context.Context, filtro portsout.RepresentanteFiltro) ([]*domaincliente.Representante, error) {
	termo := domaincliente.NomeDeBusca(filtro.Busca)
	itens := make([]*domaincliente.Representante, 0)
	for _, representante := range r.itens {
		if (termo == "" || strings.Contains(representante.NomeBusca, termo)) && (filtro.Ativo == nil || representante.Ativo == *filtro.Ativo) {
			itens = append(itens, copiar(representante))
		}
	}
	slices.SortFunc(itens, func(a, b *domaincliente.Representante) int { return strings.Compare(a.NomeBusca, b.NomeBusca) })

	return itens, nil
}

func (r *repositorioRepresentantes) Atualizar(_ context.Context, representante *domaincliente.Representante) (bool, error) {
	if _, ok := r.itens[representante.ID]; !ok {
		return false, nil
	}
	if r.repetido(representante) {
		return false, conflito()
	}
	r.itens[representante.ID] = copiar(representante)

	return true, nil
}

func (r *repositorioRepresentantes) DefinirFornecedor(_ context.Context, representante *domaincliente.Representante) (bool, error) {
	atual, ok := r.itens[representante.ID]
	if !ok {
		return false, nil
	}
	atual.FornecedorID = representante.FornecedorID
	atual.AtualizadoEm = representante.AtualizadoEm

	return true, nil
}

type repositorioClientes struct {
	itens map[uuid.UUID]*domaincliente.Cliente
	obras *repositorioObras
}

func (r *repositorioClientes) repetido(cliente *domaincliente.Cliente) bool {
	for _, existente := range r.itens {
		if existente.ID == cliente.ID {
			continue
		}
		if existente.Documento == cliente.Documento {
			return true
		}
		if cliente.LeadID != nil && existente.LeadID != nil && *existente.LeadID == *cliente.LeadID {
			return true
		}
	}

	return false
}

func (r *repositorioClientes) Criar(_ context.Context, cliente *domaincliente.Cliente) (uuid.UUID, error) {
	if r.repetido(cliente) {
		return uuid.Nil, conflito()
	}
	r.itens[cliente.ID] = copiar(cliente)

	return cliente.ID, nil
}

func (r *repositorioClientes) Obter(_ context.Context, id uuid.UUID) (*domaincliente.Cliente, error) {
	if cliente, ok := r.itens[id]; ok {
		return copiar(cliente), nil
	}

	return nil, nil
}

func (r *repositorioClientes) ObterPorDocumento(_ context.Context, documento string) (*domaincliente.Cliente, error) {
	for _, cliente := range r.itens {
		if cliente.Documento.Valor() == documento {
			return copiar(cliente), nil
		}
	}

	return nil, nil
}

func (r *repositorioClientes) Listar(_ context.Context, filtro portsout.ClienteFiltro) ([]*domaincliente.Cliente, error) {
	termo := domaincliente.NomeDeBusca(filtro.Busca)
	documento := domaincliente.TermoDeDocumento(filtro.Busca)
	itens := make([]*domaincliente.Cliente, 0)
	for _, cliente := range r.itens {
		if termo != "" && !strings.Contains(cliente.NomeBusca, termo) && (documento == "" || !strings.HasPrefix(cliente.Documento.Valor(), documento)) {
			continue
		}
		if filtro.GrupoID != nil && !iguais(cliente.GrupoID, filtro.GrupoID) {
			continue
		}
		if filtro.RepresentanteID != nil && !iguais(cliente.RepresentanteID, filtro.RepresentanteID) {
			continue
		}
		if filtro.Tipo != "" && cliente.Tipo != filtro.Tipo {
			continue
		}
		if filtro.UF != "" && cliente.UF.String() != filtro.UF {
			continue
		}
		if filtro.Ativo != nil && cliente.Ativo != *filtro.Ativo {
			continue
		}
		if filtro.LeadID != nil && !iguais(cliente.LeadID, filtro.LeadID) {
			continue
		}
		itens = append(itens, copiar(cliente))
	}
	slices.SortFunc(itens, func(a, b *domaincliente.Cliente) int { return strings.Compare(a.NomeBusca, b.NomeBusca) })

	return itens, nil
}

func iguais(a, b *uuid.UUID) bool {
	return a != nil && b != nil && *a == *b
}

func (r *repositorioClientes) Atualizar(_ context.Context, cliente *domaincliente.Cliente) (bool, error) {
	if _, ok := r.itens[cliente.ID]; !ok {
		return false, nil
	}
	if r.repetido(cliente) {
		return false, conflito()
	}
	r.itens[cliente.ID] = copiar(cliente)

	return true, nil
}

func (r *repositorioClientes) Remover(_ context.Context, id uuid.UUID) (bool, error) {
	if _, ok := r.itens[id]; !ok {
		return false, nil
	}
	for _, obra := range r.obras.itens {
		if obra.ClienteID == id {
			return false, domain.ErroValidacao("registro em uso por outros dados e não pode ser excluído")
		}
	}
	delete(r.itens, id)

	return true, nil
}

type repositorioObras struct {
	itens map[uuid.UUID]*domaincliente.Obra
}

func (r *repositorioObras) repetida(obra *domaincliente.Obra) bool {
	for _, existente := range r.itens {
		if existente.ID != obra.ID && existente.ClienteID == obra.ClienteID && strings.EqualFold(existente.Nome, obra.Nome) {
			return true
		}
	}

	return false
}

func (r *repositorioObras) Criar(_ context.Context, obra *domaincliente.Obra) (uuid.UUID, error) {
	if r.repetida(obra) {
		return uuid.Nil, conflito()
	}
	r.itens[obra.ID] = copiar(obra)

	return obra.ID, nil
}

func (r *repositorioObras) Obter(_ context.Context, id uuid.UUID) (*domaincliente.Obra, error) {
	if obra, ok := r.itens[id]; ok {
		return copiar(obra), nil
	}

	return nil, nil
}

func (r *repositorioObras) Listar(_ context.Context, filtro portsout.ObraFiltro) ([]*domaincliente.Obra, error) {
	itens := make([]*domaincliente.Obra, 0)
	for _, obra := range r.itens {
		if obra.ClienteID == filtro.ClienteID && (filtro.Ativo == nil || obra.Ativo == *filtro.Ativo) {
			itens = append(itens, copiar(obra))
		}
	}
	slices.SortFunc(itens, func(a, b *domaincliente.Obra) int { return strings.Compare(a.Nome, b.Nome) })

	return itens, nil
}

func (r *repositorioObras) Atualizar(_ context.Context, obra *domaincliente.Obra) (bool, error) {
	if _, ok := r.itens[obra.ID]; !ok {
		return false, nil
	}
	if r.repetida(obra) {
		return false, conflito()
	}
	r.itens[obra.ID] = copiar(obra)

	return true, nil
}
