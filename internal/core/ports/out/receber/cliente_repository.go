package receber

import (
	"context"

	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	"github.com/google/uuid"
)

type ClienteRepository interface {
	Obter(ctx context.Context, id uuid.UUID) (*domaincliente.Cliente, error)
}
