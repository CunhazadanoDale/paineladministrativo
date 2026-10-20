package importacao

import (
	"context"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type LocalRepository interface {
	Obter(ctx context.Context, id uuid.UUID) (*domainestoque.Local, error)
}
