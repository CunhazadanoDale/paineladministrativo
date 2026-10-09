package estoque

import (
	"context"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
)

type ResumoUseCase interface {
	Resumo(ctx context.Context) (*domainestoque.ResumoEstoque, error)
}
