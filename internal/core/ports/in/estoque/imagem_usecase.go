package estoque

import (
	"context"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type AnexarImagemInput struct {
	UsuarioID uuid.UUID
	ProdutoID uuid.UUID
	ArquivoID uuid.UUID
	Ordem     int
	Alt       string
}

type ImagemUseCase interface {
	Anexar(ctx context.Context, input AnexarImagemInput) (*domainestoque.Imagem, error)
	Remover(ctx context.Context, usuarioID, produtoID, imagemID uuid.UUID) error
	ObterPublica(ctx context.Context, imagemID uuid.UUID) (*domainestoque.Imagem, error)
}
