package comissao

import (
	"context"

	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	"github.com/google/uuid"
)

type RepresentanteRepository interface {
	Obter(ctx context.Context, id uuid.UUID) (*domaincliente.Representante, error)
	ObterPorUsuario(ctx context.Context, usuarioID uuid.UUID) (*domaincliente.Representante, error)
}
