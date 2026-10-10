package estoque

import (
	"context"

	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	"github.com/google/uuid"
)

type UsuarioRepository interface {
	Obter(ctx context.Context, id uuid.UUID) (*domainusuarios.Usuario, error)
}
