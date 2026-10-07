package solicitacao

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

type AprovadorRepository interface {
	Create(ctx context.Context, aprovador *domainsolicitacao.Aprovador) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domainsolicitacao.Aprovador, error)
	GetByUsuarioID(ctx context.Context, usuarioID uuid.UUID) (*domainsolicitacao.Aprovador, error)
	List(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*domainsolicitacao.Aprovador, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
