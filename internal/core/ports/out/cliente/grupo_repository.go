package cliente

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	"github.com/google/uuid"
)

type GrupoFiltro struct {
	domain.PaginacaoFiltro
	Busca string
	Ativo *bool
}

type GrupoRepository interface {
	Criar(ctx context.Context, grupo *domaincliente.Grupo) (uuid.UUID, error)
	Obter(ctx context.Context, id uuid.UUID) (*domaincliente.Grupo, error)
	ObterPorNomeBusca(ctx context.Context, nomeBusca string) (*domaincliente.Grupo, error)
	Listar(ctx context.Context, filtro GrupoFiltro) ([]*domaincliente.Grupo, error)
	Atualizar(ctx context.Context, grupo *domaincliente.Grupo) (bool, error)
}
