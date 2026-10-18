package receber

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/receber"
	"github.com/google/uuid"
)

type classificacaoDaContaAPagar struct {
	categorias portsout.CategoriaFinanceiraRepository
}

func (c *classificacaoDaContaAPagar) exigirCategoriaDeDespesa(ctx context.Context, categoriaID uuid.UUID) error {
	if categoriaID == uuid.Nil {
		return domain.ErroValidacao("categoria financeira da conta a pagar é obrigatória")
	}

	categoria, err := c.categorias.Obter(ctx, categoriaID)
	if err != nil {
		return err
	}
	if categoria == nil {
		return domainreceber.ErroCategoriaFinanceiraNaoEncontrada()
	}
	if !categoria.Lancavel() {
		return domainreceber.ErroCategoriaFinanceiraNaoLancavel()
	}
	if categoria.Natureza != domainfinanceiro.NaturezaDespesa {
		return domain.ErroValidacao("categoria financeira da conta a pagar deve ser de despesa")
	}

	return nil
}
