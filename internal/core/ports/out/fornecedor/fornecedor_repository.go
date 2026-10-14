package fornecedor

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	"github.com/google/uuid"
)

type FornecedorFiltro struct {
	domain.PaginacaoFiltro
	Busca string
	Pais  domainfornecedor.Pais
	Moeda domainfornecedor.Moeda
	Ativo *bool
}

type FornecedorRepository interface {
	Criar(ctx context.Context, fornecedor *domainfornecedor.Fornecedor) (uuid.UUID, error)
	Obter(ctx context.Context, id uuid.UUID) (*domainfornecedor.Fornecedor, error)
	ObterPorDocumento(ctx context.Context, documento string) (*domainfornecedor.Fornecedor, error)
	ObterPorIdentificacaoEstrangeira(ctx context.Context, pais domainfornecedor.Pais, identificacao string) (*domainfornecedor.Fornecedor, error)
	Listar(ctx context.Context, filtro FornecedorFiltro) ([]*domainfornecedor.Fornecedor, error)
	Atualizar(ctx context.Context, fornecedor *domainfornecedor.Fornecedor) (bool, error)
	Remover(ctx context.Context, id uuid.UUID) (bool, error)
}
