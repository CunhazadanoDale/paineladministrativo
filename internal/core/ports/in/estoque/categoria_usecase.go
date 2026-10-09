package estoque

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type CriarCategoriaInput struct {
	UsuarioID      uuid.UUID
	Nome           string
	CategoriaPaiID *uuid.UUID
	Ordem          int
	Icone          string
}

type AlterarCategoriaInput struct {
	UsuarioID   uuid.UUID
	CategoriaID uuid.UUID
	Nome        string
	Ordem       int
	Icone       string
}

type AlternarAtivoCategoriaInput struct {
	UsuarioID   uuid.UUID
	CategoriaID uuid.UUID
	Ativo       bool
}

type ListarCategoriasInput struct {
	Ativo  *bool
	Filtro domain.PaginacaoFiltro
}

type CategoriaUseCase interface {
	Criar(ctx context.Context, input CriarCategoriaInput) (uuid.UUID, error)
	Obter(ctx context.Context, id uuid.UUID) (*domainestoque.Categoria, error)
	Listar(ctx context.Context, input ListarCategoriasInput) ([]*domainestoque.Categoria, error)
	Alterar(ctx context.Context, input AlterarCategoriaInput) error
	AlternarAtivo(ctx context.Context, input AlternarAtivoCategoriaInput) error
}
