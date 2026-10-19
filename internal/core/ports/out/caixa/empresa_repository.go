package caixa

import (
	"context"

	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	"github.com/google/uuid"
)

type EmpresaRepository interface {
	Obter(ctx context.Context, id uuid.UUID) (*domainempresa.Empresa, error)
	AcessoDoUsuario(ctx context.Context, usuarioID uuid.UUID) (domainempresa.Acesso, error)
}
