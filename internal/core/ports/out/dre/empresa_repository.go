package dre

import (
	"context"

	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	"github.com/google/uuid"
)

type EmpresaRepository interface {
	AcessoDoUsuario(ctx context.Context, usuarioID uuid.UUID) (domainempresa.Acesso, error)
}
