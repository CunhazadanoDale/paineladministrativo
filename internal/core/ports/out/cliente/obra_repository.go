package cliente

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	"github.com/google/uuid"
)

type ObraFiltro struct {
	domain.PaginacaoFiltro
	ClienteID uuid.UUID
	Ativo     *bool
}

type ObraRepository interface {
	Criar(ctx context.Context, obra *domaincliente.Obra) (uuid.UUID, error)
	Obter(ctx context.Context, id uuid.UUID) (*domaincliente.Obra, error)
	Listar(ctx context.Context, filtro ObraFiltro) ([]*domaincliente.Obra, error)
	Atualizar(ctx context.Context, obra *domaincliente.Obra) (bool, error)
}
