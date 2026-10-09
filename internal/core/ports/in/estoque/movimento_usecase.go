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
	ProdutoID *uuid.UUID
	Tipo      string
	Filtro    domain.PaginacaoFiltro
}

type MovimentoUseCase interface {
	Movimentar(ctx context.Context, input MovimentarEstoqueInput) (*domainestoque.Movimento, error)
	Listar(ctx context.Context, input ListarMovimentosInput) ([]*domainestoque.Movimento, error)
}
