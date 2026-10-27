package tabelapreco

import (
	"context"
	"time"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	domaintabelapreco "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/tabelapreco"
	"github.com/google/uuid"
)

type TabelaPrecoImportacaoRepository interface {
	Criar(ctx context.Context, item *domaintabelapreco.Item) error
	ExisteVigencia(ctx context.Context, produtoID uuid.UUID, modelo domainpedido.Modelo, vigenteDe time.Time) (bool, error)
}

type ProdutoImportacaoRepository interface {
	Obter(ctx context.Context, id uuid.UUID) (*domainestoque.Produto, error)
	IDsPorCodigos(ctx context.Context, codigos []string) (map[string]uuid.UUID, error)
}
