package cliente

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	"github.com/google/uuid"
)

type RepresentanteFiltro struct {
	domain.PaginacaoFiltro
	Busca string
	Ativo *bool
}

type RepresentanteRepository interface {
	Criar(ctx context.Context, representante *domaincliente.Representante) (uuid.UUID, error)
	Obter(ctx context.Context, id uuid.UUID) (*domaincliente.Representante, error)
	ObterPorDocumento(ctx context.Context, documento string) (*domaincliente.Representante, error)
	ObterPorUsuario(ctx context.Context, usuarioID uuid.UUID) (*domaincliente.Representante, error)
	Listar(ctx context.Context, filtro RepresentanteFiltro) ([]*domaincliente.Representante, error)
	Atualizar(ctx context.Context, representante *domaincliente.Representante) (bool, error)
	DefinirFornecedor(ctx context.Context, representante *domaincliente.Representante) (bool, error)
}
