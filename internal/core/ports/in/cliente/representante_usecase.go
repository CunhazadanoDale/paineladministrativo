package cliente

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	"github.com/google/uuid"
)

type DadosRepresentanteInput struct {
	Nome                    string
	Documento               *string
	UsuarioVinculadoID      *uuid.UUID
	Email                   *string
	Telefone                *string
	ComissaoProntaEntregaBP int
	ComissaoImportacaoBP    int
}

type CriarRepresentanteInput struct {
	UsuarioID uuid.UUID
	Dados     DadosRepresentanteInput
}

type AlterarRepresentanteInput struct {
	UsuarioID       uuid.UUID
	RepresentanteID uuid.UUID
	Dados           DadosRepresentanteInput
}

type ListarRepresentantesInput struct {
	UsuarioID uuid.UUID
	Busca     string
	Ativo     *bool
	Filtro    domain.PaginacaoFiltro
}

type RepresentanteUseCase interface {
	Criar(ctx context.Context, input CriarRepresentanteInput) (*domaincliente.Representante, error)
	Obter(ctx context.Context, representanteID, usuarioID uuid.UUID) (*domaincliente.Representante, error)
	Listar(ctx context.Context, input ListarRepresentantesInput) ([]*domaincliente.Representante, error)
	Alterar(ctx context.Context, input AlterarRepresentanteInput) (*domaincliente.Representante, error)
	Ativar(ctx context.Context, representanteID, usuarioID uuid.UUID) (*domaincliente.Representante, error)
	Desativar(ctx context.Context, representanteID, usuarioID uuid.UUID) (*domaincliente.Representante, error)
}
