package fornecedor

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	"github.com/google/uuid"
)

type DadosFornecedorInput struct {
	IdentificacaoEstrangeira *string
	Nome                     string
	NomeFantasia             *string
	MoedaPadrao              string
	UF                       *string
	Cidade                   *string
	ContatoNome              *string
	Email                    *string
	Telefone                 *string
	Observacoes              string
}

type CriarFornecedorInput struct {
	UsuarioID uuid.UUID
	Pais      string
	Documento *string
	Dados     DadosFornecedorInput
}

type AlterarFornecedorInput struct {
	UsuarioID    uuid.UUID
	FornecedorID uuid.UUID
	Versao       int
	Dados        DadosFornecedorInput
}

type SituacaoFornecedorInput struct {
	UsuarioID    uuid.UUID
	FornecedorID uuid.UUID
	Versao       int
}

type ListarFornecedoresInput struct {
	Busca  string
	Pais   string
	Moeda  string
	Ativo  *bool
	Filtro domain.PaginacaoFiltro
}

type FornecedorUseCase interface {
	Criar(ctx context.Context, input CriarFornecedorInput) (*domainfornecedor.Fornecedor, error)
	Obter(ctx context.Context, fornecedorID uuid.UUID) (*domainfornecedor.Fornecedor, error)
	Listar(ctx context.Context, input ListarFornecedoresInput) ([]*domainfornecedor.Fornecedor, error)
	Alterar(ctx context.Context, input AlterarFornecedorInput) (*domainfornecedor.Fornecedor, error)
	Ativar(ctx context.Context, input SituacaoFornecedorInput) (*domainfornecedor.Fornecedor, error)
	Desativar(ctx context.Context, input SituacaoFornecedorInput) (*domainfornecedor.Fornecedor, error)
	Remover(ctx context.Context, fornecedorID, usuarioID uuid.UUID) error
}
