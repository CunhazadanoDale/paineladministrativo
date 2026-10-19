package tabelapreco

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaintabelapreco "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/tabelapreco"
	"github.com/google/uuid"
)

type CriarItemInput struct {
	UsuarioID uuid.UUID
	Dados     domaintabelapreco.DadosItem
}

type ListarItensInput struct {
	UsuarioID uuid.UUID
	ProdutoID *uuid.UUID
	Modelo    string
	Filtro    domain.PaginacaoFiltro
}

type VigentesInput struct {
	UsuarioID uuid.UUID
	Modelo    string
	Data      *time.Time
	ProdutoID *uuid.UUID
	Busca     string
	Filtro    domain.PaginacaoFiltro
}

type TabelaPrecoUseCase interface {
	Criar(ctx context.Context, input CriarItemInput) (*domaintabelapreco.Item, error)
	Listar(ctx context.Context, input ListarItensInput) ([]*domaintabelapreco.Item, error)
	Vigentes(ctx context.Context, input VigentesInput) ([]*domaintabelapreco.PrecoVigente, error)
	Remover(ctx context.Context, itemID, usuarioID uuid.UUID) error
}
