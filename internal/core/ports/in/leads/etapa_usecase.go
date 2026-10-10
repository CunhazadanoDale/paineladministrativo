package leads

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	"github.com/google/uuid"
)

type EtapaUseCase interface {
	Criar(ctx context.Context, etapa *lead.Etapa) (uuid.UUID, error)
	Atualizar(ctx context.Context, etapa *lead.Etapa) error
	Obter(ctx context.Context, id uuid.UUID) (*lead.Etapa, error)
	ListarPorFunil(ctx context.Context, funilID uuid.UUID) ([]*lead.Etapa, error)
	ListarPorFunilOrdenado(ctx context.Context, funilID uuid.UUID) ([]*lead.Etapa, error)
	Remover(ctx context.Context, id uuid.UUID) error
	Reordenar(ctx context.Context, funilID uuid.UUID, etapas []*lead.Etapa) error
	ObterProxima(ctx context.Context, currentEtapaID uuid.UUID) (*lead.Etapa, error)
	ObterAnterior(ctx context.Context, currentEtapaID uuid.UUID) (*lead.Etapa, error)
	ExistePorFunil(ctx context.Context, funilID uuid.UUID) (bool, error)
}
