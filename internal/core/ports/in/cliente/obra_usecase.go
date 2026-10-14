package cliente

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	"github.com/google/uuid"
)

type DadosObraInput struct {
	Nome        string
	CNPJ        *string
	Logradouro  string
	Numero      *string
	Complemento *string
	Bairro      *string
	Cidade      string
	UF          string
	CEP         *string
}

type CriarObraInput struct {
	UsuarioID uuid.UUID
	ClienteID uuid.UUID
	Dados     DadosObraInput
}

type AlterarObraInput struct {
	UsuarioID uuid.UUID
	ObraID    uuid.UUID
	Dados     DadosObraInput
}

type ListarObrasInput struct {
	UsuarioID uuid.UUID
	ClienteID uuid.UUID
	Ativo     *bool
	Filtro    domain.PaginacaoFiltro
}

type ObraUseCase interface {
	Criar(ctx context.Context, input CriarObraInput) (*domaincliente.Obra, error)
	Obter(ctx context.Context, obraID, usuarioID uuid.UUID) (*domaincliente.Obra, error)
	Listar(ctx context.Context, input ListarObrasInput) ([]*domaincliente.Obra, error)
	Alterar(ctx context.Context, input AlterarObraInput) (*domaincliente.Obra, error)
	Ativar(ctx context.Context, obraID, usuarioID uuid.UUID) (*domaincliente.Obra, error)
	Desativar(ctx context.Context, obraID, usuarioID uuid.UUID) (*domaincliente.Obra, error)
}
