package cliente

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	"github.com/google/uuid"
)

type ClienteFiltro struct {
	domain.PaginacaoFiltro
	Busca           string
	GrupoID         *uuid.UUID
	RepresentanteID *uuid.UUID
	Tipo            domaincliente.TipoCliente
	UF              string
	Ativo           *bool
	LeadID          *uuid.UUID
}

type ClienteRepository interface {
	Criar(ctx context.Context, cliente *domaincliente.Cliente) (uuid.UUID, error)
	Obter(ctx context.Context, id uuid.UUID) (*domaincliente.Cliente, error)
	ObterPorDocumento(ctx context.Context, documento string) (*domaincliente.Cliente, error)
	Listar(ctx context.Context, filtro ClienteFiltro) ([]*domaincliente.Cliente, error)
	Atualizar(ctx context.Context, cliente *domaincliente.Cliente) (bool, error)
	Remover(ctx context.Context, id uuid.UUID) (bool, error)
}
