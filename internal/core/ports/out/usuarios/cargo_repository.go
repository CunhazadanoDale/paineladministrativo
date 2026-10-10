package usuarios

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	"github.com/google/uuid"
)

type CargoRepository interface {
	Criar(ctx context.Context, cargo *usuarios.Cargo) (uuid.UUID, error)
	Atualizar(ctx context.Context, cargo *usuarios.Cargo) error
	Obter(ctx context.Context, id uuid.UUID) (*usuarios.Cargo, error)
	ObterPorNome(ctx context.Context, nome string) (*usuarios.Cargo, error)
	Listar(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*usuarios.Cargo, error)
	ListarAtivos(ctx context.Context, filtro domain.PaginacaoFiltro) ([]*usuarios.Cargo, error)
	Remover(ctx context.Context, id uuid.UUID) error
}
