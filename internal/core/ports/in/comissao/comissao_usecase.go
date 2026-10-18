package comissao

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	"github.com/google/uuid"
)

type ApurarInput struct {
	UsuarioID   uuid.UUID
	Empresa     *uuid.UUID
	Competencia string
}

type PendenciasInput struct {
	UsuarioID   uuid.UUID
	Empresa     *uuid.UUID
	Competencia string
	Filtro      domain.PaginacaoFiltro
}

type ListarComissoesInput struct {
	UsuarioID       uuid.UUID
	Empresa         *uuid.UUID
	RepresentanteID *uuid.UUID
	PedidoID        *uuid.UUID
	Competencia     string
	Aberta          *bool
	Filtro          domain.PaginacaoFiltro
}

type ComissaoUseCase interface {
	Apurar(ctx context.Context, input ApurarInput) (domaincomissao.ResultadoApuracao, error)
	Pendencias(ctx context.Context, input PendenciasInput) ([]*domaincomissao.EntregaParaApurar, error)
	Listar(ctx context.Context, input ListarComissoesInput) ([]*domaincomissao.Comissao, error)
}
