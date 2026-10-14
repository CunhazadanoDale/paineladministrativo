package fornecedor_test

import (
	"context"
	"errors"
	"sort"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/fornecedor"
	"github.com/google/uuid"
)

var (
	_ portsout.FornecedorRepository = (*repositorioFornecedores)(nil)
	_ portsout.UsuarioRepository    = (*repositorioUsuarios)(nil)
	_ portsout.CargoRepository      = (*repositorioCargos)(nil)
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

type repositorioFornecedores struct {
	itens             map[uuid.UUID]*domainfornecedor.Fornecedor
	conflitoNaCriacao bool
	falha             error
	gravacoes         int
	antesDeAtualizar  func()
	ultimoFiltro      portsout.FornecedorFiltro
}

func copiaDe(fornecedor *domainfornecedor.Fornecedor) *domainfornecedor.Fornecedor {
	copia := *fornecedor
	return &copia
}

func (r *repositorioFornecedores) Criar(_ context.Context, fornecedor *domainfornecedor.Fornecedor) (uuid.UUID, error) {
	if r.falha != nil {
		return uuid.Nil, r.falha
	}
	if r.conflitoNaCriacao {
		return uuid.Nil, domain.ErroConflito("já existe um registro com estes dados")
	}

	r.itens[fornecedor.ID] = copiaDe(fornecedor)
	r.gravacoes++

	return fornecedor.ID, nil
}

func (r *repositorioFornecedores) Obter(_ context.Context, id uuid.UUID) (*domainfornecedor.Fornecedor, error) {
	if r.falha != nil {
		return nil, r.falha
	}

	fornecedor, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	return copiaDe(fornecedor), nil
}

func (r *repositorioFornecedores) ObterPorDocumento(_ context.Context, documento string) (*domainfornecedor.Fornecedor, error) {
	for _, fornecedor := range r.itens {
		if !fornecedor.Documento.Vazio() && fornecedor.Documento.Valor() == documento {
			return copiaDe(fornecedor), nil
		}
	}

	return nil, nil
}

func (r *repositorioFornecedores) ObterPorIdentificacaoEstrangeira(
	_ context.Context,
	pais domainfornecedor.Pais,
	identificacao string,
) (*domainfornecedor.Fornecedor, error) {
	for _, fornecedor := range r.itens {
		if fornecedor.Pais == pais && fornecedor.IdentificacaoEstrangeira != nil && *fornecedor.IdentificacaoEstrangeira == identificacao {
			return copiaDe(fornecedor), nil
		}
	}

	return nil, nil
}

func (r *repositorioFornecedores) Listar(_ context.Context, filtro portsout.FornecedorFiltro) ([]*domainfornecedor.Fornecedor, error) {
	r.ultimoFiltro = filtro

	itens := make([]*domainfornecedor.Fornecedor, 0, len(r.itens))
	for _, fornecedor := range r.itens {
		if filtro.Pais != "" && fornecedor.Pais != filtro.Pais {
			continue
		}
		if filtro.Moeda != "" && fornecedor.MoedaPadrao != filtro.Moeda {
			continue
		}
		if filtro.Ativo != nil && fornecedor.Ativo != *filtro.Ativo {
			continue
		}
		itens = append(itens, copiaDe(fornecedor))
	}
	sort.Slice(itens, func(i, j int) bool { return itens[i].NomeBusca < itens[j].NomeBusca })

	return itens, nil
}

func (r *repositorioFornecedores) Atualizar(_ context.Context, fornecedor *domainfornecedor.Fornecedor) (bool, error) {
	if r.falha != nil {
		return false, r.falha
	}
	if r.antesDeAtualizar != nil {
		r.antesDeAtualizar()
	}

	atual, ok := r.itens[fornecedor.ID]
	if !ok {
		return false, nil
	}
	if atual.Versao != fornecedor.Versao {
		return false, domainfornecedor.ErroVersaoDivergente()
	}

	fornecedor.Versao++
	r.itens[fornecedor.ID] = copiaDe(fornecedor)
	r.gravacoes++

	return true, nil
}

func (r *repositorioFornecedores) Remover(_ context.Context, id uuid.UUID) (bool, error) {
	if r.falha != nil {
		return false, r.falha
	}
	if _, ok := r.itens[id]; !ok {
		return false, nil
	}

	delete(r.itens, id)

	return true, nil
}

var errBancoFora = errors.New("banco fora")
