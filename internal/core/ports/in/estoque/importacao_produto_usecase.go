package estoque

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

type LinhaImportacaoProduto struct {
	Numero           int
	Nome             string
	Codigo           string
	Categoria        string
	UnidadeMedida    string
	Preco            string
	PrecoPromocional string
	EstoqueMinimo    string
	PesoKg           string
	Descricao        string
	Destaque         string
}

type ImportarProdutosInput struct {
	UsuarioID uuid.UUID
	Linhas    []LinhaImportacaoProduto
	Simular   bool
}

type ImportacaoProdutoUseCase interface {
	Importar(ctx context.Context, input ImportarProdutosInput) (*domain.RelatorioImportacao, error)
}
