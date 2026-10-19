package comissao

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	"github.com/google/uuid"
)

type CriarAjusteInput struct {
	UsuarioID uuid.UUID
	Dados     domaincomissao.DadosAjuste
}

type ListarAjustesInput struct {
	UsuarioID       uuid.UUID
	Empresa         *uuid.UUID
	RepresentanteID *uuid.UUID
	PedidoID        *uuid.UUID
	Competencia     string
	Aberto          *bool
	Filtro          domain.PaginacaoFiltro
}

type AjusteUseCase interface {
	Criar(ctx context.Context, input CriarAjusteInput) (*domaincomissao.Ajuste, error)
	Listar(ctx context.Context, input ListarAjustesInput) ([]*domaincomissao.Ajuste, error)
}
