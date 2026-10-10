package estoque

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type ProdutoFiltro struct {
	domain.PaginacaoFiltro
	CategoriaID  *uuid.UUID
	Ativo        *bool
	Destaque     *bool
	EstoqueBaixo bool
	ComSaldo     bool
	Busca        string
	NaVitrine    bool
}

type ProdutoRepository interface {
	Criar(ctx context.Context, produto *domainestoque.Produto) (uuid.UUID, error)
	Obter(ctx context.Context, id uuid.UUID) (*domainestoque.Produto, error)
	ObterPorSlug(ctx context.Context, slug string) (*domainestoque.Produto, error)
	Listar(ctx context.Context, filtro ProdutoFiltro) ([]*domainestoque.Produto, error)
	Resumo(ctx context.Context) (domainestoque.ResumoProdutos, error)
	Atualizar(ctx context.Context, produto *domainestoque.Produto) (bool, error)
	Movimentar(ctx context.Context, produto *domainestoque.Produto, movimento *domainestoque.Movimento) error
}
