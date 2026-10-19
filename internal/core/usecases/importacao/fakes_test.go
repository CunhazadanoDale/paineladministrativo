package importacao_test

import (
	"context"
	"slices"
	"sort"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/importacao"
	"github.com/google/uuid"
)

type memoria struct {
	usuarios     map[uuid.UUID]*domainusuarios.Usuario
	cargos       map[uuid.UUID]*domainusuarios.Cargo
	empresas     map[uuid.UUID]*domainempresa.Empresa
	acessos      map[uuid.UUID]domainempresa.Acesso
	fornecedores map[uuid.UUID]*domainfornecedor.Fornecedor
	produtos     map[uuid.UUID]*domainestoque.Produto
	importacoes  map[uuid.UUID]*domainimportacao.Importacao
	historicos   []*domainimportacao.Historico
	sequencias   map[uuid.UUID]int
	gravacoes    []*portsout.Gravacao
	categorias   map[uuid.UUID]*domainfinanceiro.CategoriaFinanceira
	contas       map[uuid.UUID]*domainsolicitacao.Solicitacao
}

func novaMemoria() *memoria {
	return &memoria{
		usuarios:     map[uuid.UUID]*domainusuarios.Usuario{},
		cargos:       map[uuid.UUID]*domainusuarios.Cargo{},
		empresas:     map[uuid.UUID]*domainempresa.Empresa{},
		acessos:      map[uuid.UUID]domainempresa.Acesso{},
		fornecedores: map[uuid.UUID]*domainfornecedor.Fornecedor{},
		produtos:     map[uuid.UUID]*domainestoque.Produto{},
		importacoes:  map[uuid.UUID]*domainimportacao.Importacao{},
		sequencias:   map[uuid.UUID]int{},
		categorias:   map[uuid.UUID]*domainfinanceiro.CategoriaFinanceira{},
		contas:       map[uuid.UUID]*domainsolicitacao.Solicitacao{},
	}
}

func copiar[T any](item *T) *T {
	if item == nil {
		return nil
	}
	copia := *item

	return &copia
}

func copiarImportacao(importacao *domainimportacao.Importacao) *domainimportacao.Importacao {
	copia := *importacao
	copia.Itens = make([]*domainimportacao.Item, 0, len(importacao.Itens))
	for _, item := range importacao.Itens {
		copia.Itens = append(copia.Itens, copiar(item))
	}
	copia.Parcelas = make([]*domainimportacao.Parcela, 0, len(importacao.Parcelas))
	for _, parcela := range importacao.Parcelas {
		copia.Parcelas = append(copia.Parcelas, copiar(parcela))
	}
	copia.Custos = make([]*domainimportacao.Custo, 0, len(importacao.Custos))
	for _, custo := range importacao.Custos {
		copia.Custos = append(copia.Custos, copiar(custo))
	}

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

type repoFornecedores struct{ *memoria }

func (r repoFornecedores) Obter(_ context.Context, id uuid.UUID) (*domainfornecedor.Fornecedor, error) {
	return copiar(r.fornecedores[id]), nil
}

type repoProdutos struct{ *memoria }

func (r repoProdutos) Obter(_ context.Context, id uuid.UUID) (*domainestoque.Produto, error) {
	return copiar(r.produtos[id]), nil
}

type repoImportacoes struct{ *memoria }

func (r repoImportacoes) Gravar(_ context.Context, gravacao *portsout.Gravacao) error {
	importacao := copiarImportacao(gravacao.Importacao)
	if gravacao.Novo {
		if empresa := r.empresas[importacao.EmpresaID]; empresa == nil || !empresa.Ativa {
			return domain.ErroValidacao("empresa inativa não recebe importação nova")
		}
		importacao.Numero = r.sequencias[importacao.EmpresaID] + 1
	} else {
		atual, ok := r.importacoes[importacao.ID]
		if !ok {
			return domain.ErroNaoEncontrado("importação não encontrada")
		}
		if atual.Versao != gravacao.VersaoLida || atual.Status != gravacao.StatusAnterior {
			return domainimportacao.ErroVersaoDivergente()
		}
		importacao.Versao = atual.Versao + 1
	}
	if gravacao.FornecedorTrocado {
		if fornecedor := r.fornecedores[importacao.FornecedorID]; fornecedor == nil || !fornecedor.Ativo {
			return domain.ErroValidacao("fornecedor inativo não recebe importação nova")
		}
	}
	for _, id := range gravacao.ContasExigidasEncerradas {
		if conta := r.contas[id]; conta == nil || (conta.Status != domainsolicitacao.StatusCancelado && conta.Status != domainsolicitacao.StatusRejeitado) {
			return domainimportacao.ErroContasAPagarAbertas()
		}
	}
	for _, conta := range gravacao.ContasAPagar {
		if _, existe := r.contas[conta.Solicitacao.ID]; existe {
			return domain.ErroConflito("conta a pagar já gerada")
		}
	}

	if gravacao.Novo {
		r.sequencias[importacao.EmpresaID] = importacao.Numero
	}
	r.importacoes[importacao.ID] = importacao
	for _, conta := range gravacao.ContasAPagar {
		r.contas[conta.Solicitacao.ID] = copiar(conta.Solicitacao)
	}
	r.historicos = append(r.historicos, gravacao.Historicos...)
	r.gravacoes = append(r.gravacoes, gravacao)
	gravacao.Importacao.Numero = importacao.Numero
	gravacao.Importacao.Versao = importacao.Versao

	return nil
}

func (r repoImportacoes) Obter(_ context.Context, id uuid.UUID) (*domainimportacao.Importacao, error) {
	importacao, ok := r.importacoes[id]
	if !ok {
		return nil, nil
	}

	copia := copiarImportacao(importacao)
	for _, parcela := range copia.Parcelas {
		if parcela.SolicitacaoID != nil {
			status := r.contas[*parcela.SolicitacaoID].Status
			parcela.StatusContaAPagar = &status
		}
	}
	for _, custo := range copia.Custos {
		status := r.contas[custo.SolicitacaoID].Status
		custo.StatusContaAPagar = &status
	}

	return copia, nil
}

func (r repoImportacoes) Listar(_ context.Context, filtro portsout.ImportacaoFiltro) ([]*domainimportacao.Importacao, error) {
	encontradas := make([]*domainimportacao.Importacao, 0)
	for _, importacao := range r.importacoes {
		switch {
		case filtro.Empresa != nil && importacao.EmpresaID != *filtro.Empresa:
		case filtro.EmpresaIDs != nil && !slices.Contains(filtro.EmpresaIDs, importacao.EmpresaID):
		case filtro.FornecedorID != nil && importacao.FornecedorID != *filtro.FornecedorID:
		case filtro.Status != "" && importacao.Status != filtro.Status:
		case filtro.Numero != nil && importacao.Numero != *filtro.Numero:
		case filtro.ChegadaAte != nil && (importacao.ChegadaPrevista == nil || importacao.ChegadaPrevista.After(*filtro.ChegadaAte)):
		default:
			encontradas = append(encontradas, copiarImportacao(importacao))
		}
	}
	sort.Slice(encontradas, func(i, j int) bool { return encontradas[i].ID.String() < encontradas[j].ID.String() })

	return encontradas, nil
}

func (r repoImportacoes) ListarHistorico(_ context.Context, importacaoID uuid.UUID) ([]*domainimportacao.Historico, error) {
	historicos := make([]*domainimportacao.Historico, 0)
	for _, historico := range r.historicos {
		if historico.ImportacaoID == importacaoID {
			historicos = append(historicos, historico)
		}
	}

	return historicos, nil
}

type repoCategorias struct{ *memoria }

func (r repoCategorias) Obter(_ context.Context, id uuid.UUID) (*domainfinanceiro.CategoriaFinanceira, error) {
	return copiar(r.categorias[id]), nil
}
