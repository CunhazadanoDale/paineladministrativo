package estoque

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type MovimentoFiltro struct {
	domain.PaginacaoFiltro
	ProdutoID  *uuid.UUID
	Tipo       domainestoque.TipoMovimento
	LocalID    *uuid.UUID
	EmpresaIDs []uuid.UUID
}

type MovimentoRepository interface {
	Listar(ctx context.Context, filtro MovimentoFiltro) ([]*domainestoque.Movimento, error)
	ListarResumo(ctx context.Context, limite int, empresaIDs []uuid.UUID) ([]*domainestoque.MovimentoResumo, error)
}
