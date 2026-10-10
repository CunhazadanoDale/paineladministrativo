package leads

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	"github.com/google/uuid"
)

type LeadRepository interface {
	Criar(ctx context.Context, lead *lead.Lead) (uuid.UUID, error)
	Atualizar(ctx context.Context, lead *lead.Lead) error
	Obter(ctx context.Context, id uuid.UUID) (*lead.Lead, error)
	ListarPorFunil(ctx context.Context, funilID uuid.UUID) ([]*lead.Lead, error)
	ListarPorEtapa(ctx context.Context, etapaID uuid.UUID) ([]*lead.Lead, error)
	ListarAtivos(ctx context.Context, paginacao domain.PaginacaoFiltro) ([]*lead.Lead, error)
	Buscar(ctx context.Context, query string, paginacao domain.PaginacaoFiltro) ([]*lead.Lead, error)
	Remover(ctx context.Context, id uuid.UUID) error
	ContarPorEtapa(ctx context.Context, etapaID uuid.UUID) (int, error)
	ContarPorFunil(ctx context.Context, funilID uuid.UUID) (int, error)
	MoverParaEtapa(ctx context.Context, leadID uuid.UUID, etapaAnteriorID uuid.UUID, etapaAtualID uuid.UUID) error
}
