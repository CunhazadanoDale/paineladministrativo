package financeiro

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	"github.com/google/uuid"
)

type CategoriaFinanceiraFiltro struct {
	domain.PaginacaoFiltro
	Pai         *uuid.UUID
	SomenteRaiz bool
	Natureza    *domainfinanceiro.Natureza
	LinhaDRE    *domainfinanceiro.LinhaDRE
	Ativa       *bool
	Busca       string
}

type CategoriaFinanceiraRepository interface {
	Criar(ctx context.Context, categoria *domainfinanceiro.CategoriaFinanceira) error
	Obter(ctx context.Context, id uuid.UUID) (*domainfinanceiro.CategoriaFinanceira, error)
	Listar(ctx context.Context, filtro CategoriaFinanceiraFiltro) ([]*domainfinanceiro.CategoriaFinanceira, error)
	ListarPlano(ctx context.Context, somenteAtivas bool) ([]*domainfinanceiro.CategoriaFinanceira, error)
	PossuiSubcategoriasAtivas(ctx context.Context, id uuid.UUID) (bool, error)
	Atualizar(ctx context.Context, categoria *domainfinanceiro.CategoriaFinanceira) error
	Ativar(ctx context.Context, categoria *domainfinanceiro.CategoriaFinanceira) error
	Desativar(ctx context.Context, categoria *domainfinanceiro.CategoriaFinanceira) error
	Remover(ctx context.Context, id uuid.UUID) error
}
