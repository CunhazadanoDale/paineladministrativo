package estoque

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type SaldoDoLocalFiltro struct {
	domain.PaginacaoFiltro
	LocalID  uuid.UUID
	ComSaldo bool
}

type EstoqueRepository interface {
	Carregar(ctx context.Context, produtoID uuid.UUID, localIDs []uuid.UUID) (*domainestoque.EstoqueProduto, error)
	Gravar(ctx context.Context, operacao *domainestoque.OperacaoEstoque) error
	SaldosDoProduto(ctx context.Context, produtoID uuid.UUID, empresaIDs []uuid.UUID) ([]*domainestoque.SaldoLocalDetalhe, error)
	SaldosDoLocal(ctx context.Context, filtro SaldoDoLocalFiltro) ([]*domainestoque.SaldoLocalDetalhe, error)
	TotalAlocado(ctx context.Context, produtoID uuid.UUID) (int, error)
}
