package estoque

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type MovimentarEstoqueInput struct {
	UsuarioID    uuid.UUID
	ProdutoID    uuid.UUID
	Tipo         string
	Quantidade   int
	DocumentoRef *string
	Observacao   string
}

type ListarMovimentosInput struct {
	UsuarioID uuid.UUID
	ProdutoID *uuid.UUID
	LocalID   *uuid.UUID
	Tipo      string
	Filtro    domain.PaginacaoFiltro
}

type TransferirEstoqueInput struct {
	UsuarioID      uuid.UUID
	ProdutoID      uuid.UUID
	LocalOrigemID  uuid.UUID
	LocalDestinoID uuid.UUID
	Quantidade     int
	DocumentoRef   *string
	Observacao     string
}

type ListarLotesInput struct {
	UsuarioID uuid.UUID
	ProdutoID uuid.UUID
	LocalID   *uuid.UUID
	ComSaldo  bool
	Filtro    domain.PaginacaoFiltro
}

type MovimentoUseCase interface {
	Movimentar(ctx context.Context, input MovimentarEstoqueInput) (*domainestoque.Movimento, error)
	Transferir(ctx context.Context, input TransferirEstoqueInput) (*domainestoque.Movimento, error)
	Listar(ctx context.Context, input ListarMovimentosInput) ([]*domainestoque.Movimento, error)
	Saldo(ctx context.Context, produtoID, usuarioID uuid.UUID) (*domainestoque.SaldoProduto, error)
	ListarLotes(ctx context.Context, input ListarLotesInput) ([]*domainestoque.Lote, error)
}
