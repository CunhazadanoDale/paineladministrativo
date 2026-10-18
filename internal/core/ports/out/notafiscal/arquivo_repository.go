package notafiscal

import (
	"context"

	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

type ArquivoRepository interface {
	Obter(ctx context.Context, id uuid.UUID) (*domainsolicitacao.Arquivo, error)
	VinculadoASolicitacao(ctx context.Context, arquivoID uuid.UUID) (bool, error)
}
