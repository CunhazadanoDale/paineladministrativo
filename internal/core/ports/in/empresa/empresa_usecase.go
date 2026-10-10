package empresa

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	"github.com/google/uuid"
)

type CriarEmpresaInput struct {
	UsuarioID        uuid.UUID
	CNPJ             string
	RazaoSocial      string
	NomeFantasia     string
	UF               string
	RegimeTributario string
}

type AlterarEmpresaInput struct {
	UsuarioID        uuid.UUID
	EmpresaID        uuid.UUID
	RazaoSocial      string
	NomeFantasia     string
	UF               string
	RegimeTributario string
}

type AlternarAtivaEmpresaInput struct {
	UsuarioID uuid.UUID
	EmpresaID uuid.UUID
	Ativa     bool
}

type ListarEmpresasInput struct {
	UsuarioID uuid.UUID
	Ativa     *bool
	Filtro    domain.PaginacaoFiltro
}

type DefinirVinculosInput struct {
	UsuarioID   uuid.UUID
	AlvoID      uuid.UUID
	AcessoTodas bool
	EmpresaIDs  []uuid.UUID
}

type EmpresaUseCase interface {
	Criar(ctx context.Context, input CriarEmpresaInput) (uuid.UUID, error)
	Obter(ctx context.Context, empresaID, usuarioID uuid.UUID) (*domainempresa.Empresa, error)
	Listar(ctx context.Context, input ListarEmpresasInput) ([]*domainempresa.Empresa, error)
	Alterar(ctx context.Context, input AlterarEmpresaInput) error
	AlternarAtiva(ctx context.Context, input AlternarAtivaEmpresaInput) error
	Remover(ctx context.Context, empresaID, usuarioID uuid.UUID) error
	ObterVinculos(ctx context.Context, alvoID, usuarioID uuid.UUID) (*domainempresa.Vinculos, error)
	DefinirVinculos(ctx context.Context, input DefinirVinculosInput) error
}
