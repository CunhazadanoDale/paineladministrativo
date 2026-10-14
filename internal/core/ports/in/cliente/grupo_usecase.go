package cliente

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	"github.com/google/uuid"
)

type CriarGrupoInput struct {
	UsuarioID uuid.UUID
	Nome      string
}

type RenomearGrupoInput struct {
	UsuarioID uuid.UUID
	GrupoID   uuid.UUID
	Nome      string
}

type ListarGruposInput struct {
	Busca  string
	Ativo  *bool
	Filtro domain.PaginacaoFiltro
}

type GrupoUseCase interface {
	Criar(ctx context.Context, input CriarGrupoInput) (*domaincliente.Grupo, error)
	Obter(ctx context.Context, grupoID uuid.UUID) (*domaincliente.Grupo, error)
	Listar(ctx context.Context, input ListarGruposInput) ([]*domaincliente.Grupo, error)
	Renomear(ctx context.Context, input RenomearGrupoInput) (*domaincliente.Grupo, error)
	Ativar(ctx context.Context, grupoID, usuarioID uuid.UUID) (*domaincliente.Grupo, error)
	Desativar(ctx context.Context, grupoID, usuarioID uuid.UUID) (*domaincliente.Grupo, error)
}
