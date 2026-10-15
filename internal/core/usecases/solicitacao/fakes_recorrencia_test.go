package solicitacao_test

import (
	"context"

	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/solicitacao"
	"github.com/google/uuid"
)

var _ portsout.DespesaRecorrenteRepository = (*repositorioRecorrencias)(nil)

type repositorioRecorrencias struct {
	itens map[uuid.UUID]*domainsolicitacao.DespesaRecorrente
}

func novoRepositorioRecorrencias() *repositorioRecorrencias {
	return &repositorioRecorrencias{itens: map[uuid.UUID]*domainsolicitacao.DespesaRecorrente{}}
}

func (r *repositorioRecorrencias) Criar(_ context.Context, despesa *domainsolicitacao.DespesaRecorrente) error {
	copia := *despesa
	r.itens[despesa.ID] = &copia

	return nil
}

func (r *repositorioRecorrencias) Obter(_ context.Context, id uuid.UUID) (*domainsolicitacao.DespesaRecorrente, error) {
	despesa, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *despesa
	return &copia, nil
}

func (r *repositorioRecorrencias) Listar(_ context.Context, filtro portsout.DespesaRecorrenteFiltro) ([]*domainsolicitacao.DespesaRecorrente, error) {
	itens := make([]*domainsolicitacao.DespesaRecorrente, 0)
	for _, despesa := range r.itens {
		if filtro.EmpresaIDs != nil && !contemEmpresa(filtro.EmpresaIDs, despesa.EmpresaID) {
			continue
		}
		if filtro.Ativa != nil && despesa.Ativa != *filtro.Ativa {
			continue
		}
		if filtro.Fornecedor != nil && (despesa.FornecedorID == nil || *despesa.FornecedorID != *filtro.Fornecedor) {
			continue
		}
		copia := *despesa
		itens = append(itens, &copia)
	}

	return paginar(itens, filtro.PaginacaoFiltro), nil
}

func (r *repositorioRecorrencias) ListarVigentes(_ context.Context, competencia domainsolicitacao.Competencia, empresaIDs []uuid.UUID) ([]*domainsolicitacao.DespesaRecorrente, error) {
	itens := make([]*domainsolicitacao.DespesaRecorrente, 0)
	for _, despesa := range r.itens {
		if empresaIDs != nil && !contemEmpresa(empresaIDs, despesa.EmpresaID) {
			continue
		}
		if !despesa.VigenteEm(competencia) {
			continue
		}
		copia := *despesa
		itens = append(itens, &copia)
	}

	return itens, nil
}

func (r *repositorioRecorrencias) Atualizar(_ context.Context, despesa *domainsolicitacao.DespesaRecorrente) error {
	return r.gravar(despesa)
}

func (r *repositorioRecorrencias) Ativar(_ context.Context, despesa *domainsolicitacao.DespesaRecorrente) error {
	return r.gravar(despesa)
}

func (r *repositorioRecorrencias) Desativar(_ context.Context, despesa *domainsolicitacao.DespesaRecorrente) error {
	return r.gravar(despesa)
}

func (r *repositorioRecorrencias) Remover(_ context.Context, id uuid.UUID) error {
	if _, ok := r.itens[id]; !ok {
		return domainsolicitacao.ErroDespesaRecorrenteNaoEncontrada()
	}
	delete(r.itens, id)

	return nil
}

func (r *repositorioRecorrencias) gravar(despesa *domainsolicitacao.DespesaRecorrente) error {
	salva, ok := r.itens[despesa.ID]
	if !ok {
		return domainsolicitacao.ErroDespesaRecorrenteNaoEncontrada()
	}
	if salva.Versao != despesa.Versao {
		return domainsolicitacao.ErroDespesaRecorrenteAlterada()
	}

	despesa.Versao++
	copia := *despesa
	r.itens[despesa.ID] = &copia

	return nil
}

func (r *repositorioSolicitacoes) GerarDeRecorrencia(_ context.Context, solicitacao *domainsolicitacao.Solicitacao, historico *domainsolicitacao.Historico) (bool, error) {
	for _, existente := range r.itens {
		if existente.Origem == domainsolicitacao.OrigemRecorrencia && *existente.OrigemID == *solicitacao.OrigemID &&
			existente.Competencia == solicitacao.Competencia {
			return false, nil
		}
	}

	copia := *solicitacao
	r.itens[solicitacao.ID] = &copia
	r.registrarHistorico(solicitacao.ID, historico)

	return true, nil
}
