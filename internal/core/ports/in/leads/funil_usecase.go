package leads

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	"github.com/google/uuid"
)

type FunilUseCase interface {
	Criar(ctx context.Context, funil *lead.Funil) (uuid.UUID, error)
	Atualizar(ctx context.Context, funil *lead.Funil) error
	Obter(ctx context.Context, funilID uuid.UUID) (*lead.Funil, error)
	Listar(ctx context.Context, filtro domain.PaginacaoFiltro) ([]lead.Funil, error)
	ListarAtivos(ctx context.Context, filtro domain.PaginacaoFiltro) ([]lead.Funil, error)
	Remover(ctx context.Context, funilID uuid.UUID) error
	Existe(ctx context.Context, funilID uuid.UUID) (bool, error)
}
