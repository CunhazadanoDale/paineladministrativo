package financeiro

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	"github.com/google/uuid"
)

type DadosCategoriaFinanceiraInput struct {
	Nome   string
	Codigo *string
	Ordem  int
}

type CriarCategoriaFinanceiraInput struct {
	UsuarioID      uuid.UUID
	CategoriaPaiID *uuid.UUID
	Natureza       *string
	LinhaDRE       *string
	Dados          DadosCategoriaFinanceiraInput
}

type AlterarCategoriaFinanceiraInput struct {
	UsuarioID   uuid.UUID
	CategoriaID uuid.UUID
	Dados       DadosCategoriaFinanceiraInput
	Versao      int
}

type MudarSituacaoCategoriaFinanceiraInput struct {
	UsuarioID   uuid.UUID
	CategoriaID uuid.UUID
	Versao      int
}

type ListarCategoriasFinanceirasInput struct {
	Pai         *uuid.UUID
	SomenteRaiz bool
	Natureza    string
	LinhaDRE    string
	Ativa       *bool
	Busca       string
	Filtro      domain.PaginacaoFiltro
}

type CategoriaFinanceiraUseCase interface {
	Criar(ctx context.Context, input CriarCategoriaFinanceiraInput) (*domainfinanceiro.CategoriaFinanceira, error)
	Obter(ctx context.Context, categoriaID uuid.UUID) (*domainfinanceiro.CategoriaFinanceira, error)
	Listar(ctx context.Context, input ListarCategoriasFinanceirasInput) ([]*domainfinanceiro.CategoriaFinanceira, error)
	Plano(ctx context.Context, somenteAtivas bool) ([]domainfinanceiro.RamoDoPlano, error)
	Alterar(ctx context.Context, input AlterarCategoriaFinanceiraInput) (*domainfinanceiro.CategoriaFinanceira, error)
	Ativar(ctx context.Context, input MudarSituacaoCategoriaFinanceiraInput) (*domainfinanceiro.CategoriaFinanceira, error)
	Desativar(ctx context.Context, input MudarSituacaoCategoriaFinanceiraInput) (*domainfinanceiro.CategoriaFinanceira, error)
	Remover(ctx context.Context, categoriaID, usuarioID uuid.UUID) error
}
