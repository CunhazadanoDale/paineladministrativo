package empresa

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	"github.com/google/uuid"
)

type EmpresaFiltro struct {
	domain.PaginacaoFiltro
	Ativa      *bool
	EmpresaIDs []uuid.UUID
}

type EmpresaRepository interface {
	Criar(ctx context.Context, empresa *domainempresa.Empresa) (uuid.UUID, error)
	Obter(ctx context.Context, id uuid.UUID) (*domainempresa.Empresa, error)
	ObterPorCNPJ(ctx context.Context, cnpj domainempresa.CNPJ) (*domainempresa.Empresa, error)
	Listar(ctx context.Context, filtro EmpresaFiltro) ([]*domainempresa.Empresa, error)
	Atualizar(ctx context.Context, empresa *domainempresa.Empresa) (bool, error)
	Remover(ctx context.Context, id uuid.UUID) (bool, error)
	AcessoDoUsuario(ctx context.Context, usuarioID uuid.UUID) (domainempresa.Acesso, error)
	VinculosDoUsuario(ctx context.Context, usuarioID uuid.UUID) (*domainempresa.Vinculos, error)
	DefinirVinculos(ctx context.Context, usuarioID uuid.UUID, vinculos domainempresa.Vinculos) error
}
