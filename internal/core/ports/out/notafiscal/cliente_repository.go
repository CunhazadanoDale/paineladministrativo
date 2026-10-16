package notafiscal

import (
	"context"

	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	"github.com/google/uuid"
)

type ClienteRepository interface {
	Obter(ctx context.Context, id uuid.UUID) (*domaincliente.Cliente, error)
	ObterPorDocumento(ctx context.Context, documento string) (*domaincliente.Cliente, error)
	Criar(ctx context.Context, cliente *domaincliente.Cliente) (uuid.UUID, error)
}
