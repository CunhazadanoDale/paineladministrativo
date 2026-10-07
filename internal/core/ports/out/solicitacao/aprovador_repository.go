package solicitacao

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

type AprovadorRepository interface {
	Criar(ctx context.Context, aprovador *domainsolicitacao.Aprovador) (uuid.UUID, error)
	Obter(ctx context.Context, id uuid.UUID) (*domainsolicitacao.Aprovador, error)
	ObterPorUsuarioID(ctx context.Context, usuarioID uuid.UUID) (*domainsolicitacao.Aprovador, error)
	Listar(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*domainsolicitacao.Aprovador, error)
	Remover(ctx context.Context, id uuid.UUID) error
}
