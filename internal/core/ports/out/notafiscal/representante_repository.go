package notafiscal

import (
	"context"

	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	"github.com/google/uuid"
)

type RepresentanteRepository interface {
	ObterPorUsuario(ctx context.Context, usuarioID uuid.UUID) (*domaincliente.Representante, error)
}
