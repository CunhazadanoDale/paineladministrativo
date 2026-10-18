package comissao

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	"github.com/google/uuid"
)

type FecharInput struct {
	UsuarioID             uuid.UUID
	EmpresaID             uuid.UUID
	Competencia           string
	CategoriaFinanceiraID uuid.UUID
	RepresentanteID       *uuid.UUID
}

type ListarFechamentosInput struct {
	UsuarioID       uuid.UUID
	Empresa         *uuid.UUID
	RepresentanteID *uuid.UUID
	Competencia     string
	Filtro          domain.PaginacaoFiltro
}

type FechamentoUseCase interface {
	Fechar(ctx context.Context, input FecharInput) (domaincomissao.ResultadoFechamento, error)
	Listar(ctx context.Context, input ListarFechamentosInput) ([]*domaincomissao.Fechamento, error)
	Obter(ctx context.Context, fechamentoID, usuarioID uuid.UUID) (*domaincomissao.FechamentoComItens, error)
}
