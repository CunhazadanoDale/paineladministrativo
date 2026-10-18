package notafiscal

import (
	"context"

	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	"github.com/google/uuid"
)

type EmpresaRepository interface {
	ObterPorCNPJ(ctx context.Context, cnpj domainempresa.CNPJ) (*domainempresa.Empresa, error)
	AcessoDoUsuario(ctx context.Context, usuarioID uuid.UUID) (domainempresa.Acesso, error)
}
