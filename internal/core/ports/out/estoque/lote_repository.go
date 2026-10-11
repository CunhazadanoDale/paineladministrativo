package estoque

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type LoteFiltro struct {
	domain.PaginacaoFiltro
	ProdutoID  uuid.UUID
	LocalID    *uuid.UUID
	EmpresaIDs []uuid.UUID
	ComSaldo   bool
}

type LoteRepository interface {
	Listar(ctx context.Context, filtro LoteFiltro) ([]*domainestoque.Lote, error)
}
