package estoque

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type CriarLocalInput struct {
	UsuarioID uuid.UUID
	EmpresaID uuid.UUID
	Nome      string
}

type RenomearLocalInput struct {
	UsuarioID uuid.UUID
	LocalID   uuid.UUID
	Nome      string
}

type ListarLocaisInput struct {
	UsuarioID uuid.UUID
	Empresa   *uuid.UUID
	Ativo     *bool
	Filtro    domain.PaginacaoFiltro
}

type ListarSaldosDoLocalInput struct {
	UsuarioID uuid.UUID
	LocalID   uuid.UUID
	ComSaldo  bool
	Filtro    domain.PaginacaoFiltro
}

type LocalUseCase interface {
	Criar(ctx context.Context, input CriarLocalInput) (uuid.UUID, error)
	Obter(ctx context.Context, localID, usuarioID uuid.UUID) (*domainestoque.Local, error)
	Listar(ctx context.Context, input ListarLocaisInput) ([]*domainestoque.Local, error)
	Renomear(ctx context.Context, input RenomearLocalInput) (*domainestoque.Local, error)
	Ativar(ctx context.Context, localID, usuarioID uuid.UUID) (*domainestoque.Local, error)
	Desativar(ctx context.Context, localID, usuarioID uuid.UUID) (*domainestoque.Local, error)
	Remover(ctx context.Context, localID, usuarioID uuid.UUID) error
	ListarSaldos(ctx context.Context, input ListarSaldosDoLocalInput) ([]*domainestoque.SaldoLocalDetalhe, error)
}
