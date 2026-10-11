package estoque

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type LocalFiltro struct {
	domain.PaginacaoFiltro
	Empresa    *uuid.UUID
	EmpresaIDs []uuid.UUID
	Ativo      *bool
}

type LocalRepository interface {
	Criar(ctx context.Context, local *domainestoque.Local) (uuid.UUID, error)
	Obter(ctx context.Context, id uuid.UUID) (*domainestoque.Local, error)
	ObterVarios(ctx context.Context, ids []uuid.UUID) ([]*domainestoque.Local, error)
	Listar(ctx context.Context, filtro LocalFiltro) ([]*domainestoque.Local, error)
	Atualizar(ctx context.Context, local *domainestoque.Local) (bool, error)
	DesativarSemSaldo(ctx context.Context, id uuid.UUID, agora time.Time) (bool, error)
	Remover(ctx context.Context, id uuid.UUID) (bool, error)
}
