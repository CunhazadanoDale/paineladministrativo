package solicitacao

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

type DespesaRecorrenteFiltro struct {
	domain.PaginacaoFiltro
	EmpresaIDs []uuid.UUID
	Ativa      *bool
	Fornecedor *uuid.UUID
}

type DespesaRecorrenteRepository interface {
	Criar(ctx context.Context, despesa *domainsolicitacao.DespesaRecorrente) error
	Obter(ctx context.Context, id uuid.UUID) (*domainsolicitacao.DespesaRecorrente, error)
	Listar(ctx context.Context, filtro DespesaRecorrenteFiltro) ([]*domainsolicitacao.DespesaRecorrente, error)
	ListarVigentes(ctx context.Context, competencia domainsolicitacao.Competencia, empresaIDs []uuid.UUID) ([]*domainsolicitacao.DespesaRecorrente, error)
	Atualizar(ctx context.Context, despesa *domainsolicitacao.DespesaRecorrente) error
	Ativar(ctx context.Context, despesa *domainsolicitacao.DespesaRecorrente) error
	Desativar(ctx context.Context, despesa *domainsolicitacao.DespesaRecorrente) error
	Remover(ctx context.Context, id uuid.UUID) error
}
