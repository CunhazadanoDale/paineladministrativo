package estoque

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type CriarProdutoInput struct {
	UsuarioID                uuid.UUID
	CategoriaID              uuid.UUID
	Nome                     string
	Descricao                string
	Codigo                   *string
	UnidadeMedida            string
	PrecoCentavos            *int64
	PrecoPromocionalCentavos *int64
	EstoqueMinimo            *int
	PesoKg                   *float64
	Destaque                 bool
}

type AlterarProdutoInput struct {
	UsuarioID                uuid.UUID
	ProdutoID                uuid.UUID
	CategoriaID              uuid.UUID
	Nome                     string
	Descricao                string
	Codigo                   *string
	UnidadeMedida            string
	PrecoCentavos            *int64
	PrecoPromocionalCentavos *int64
	EstoqueMinimo            *int
	PesoKg                   *float64
	Destaque                 bool
}

type AlternarAtivoProdutoInput struct {
	UsuarioID uuid.UUID
	ProdutoID uuid.UUID
	Ativo     bool
}

type AlternarDestaqueProdutoInput struct {
	UsuarioID uuid.UUID
	ProdutoID uuid.UUID
	Destaque  bool
}

type ListarProdutosInput struct {
	CategoriaID  *uuid.UUID
	Ativo        *bool
	Destaque     *bool
	EstoqueBaixo bool
	Busca        string
	Filtro       domain.PaginacaoFiltro
}

type ProdutoUseCase interface {
	Criar(ctx context.Context, input CriarProdutoInput) (uuid.UUID, error)
	Obter(ctx context.Context, id uuid.UUID) (*domainestoque.Produto, error)
	Listar(ctx context.Context, input ListarProdutosInput) ([]*domainestoque.Produto, error)
	Alterar(ctx context.Context, input AlterarProdutoInput) error
	AlternarAtivo(ctx context.Context, input AlternarAtivoProdutoInput) error
	AlternarDestaque(ctx context.Context, input AlternarDestaqueProdutoInput) error
}
