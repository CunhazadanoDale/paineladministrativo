package comissao

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	"github.com/google/uuid"
)

type EntregaFiltro struct {
	domain.PaginacaoFiltro
	Empresa      *uuid.UUID
	EmpresaIDs   []uuid.UUID
	SaidaAntesDe *time.Time
}

type ComissaoFiltro struct {
	domain.PaginacaoFiltro
	Empresa         *uuid.UUID
	EmpresaIDs      []uuid.UUID
	RepresentanteID *uuid.UUID
	PedidoID        *uuid.UUID
	Competencia     *domaincomissao.Competencia
}

type ComissaoRepository interface {
	ObterPercentual(ctx context.Context, pedidoID uuid.UUID) (*domaincomissao.PercentualPedido, error)
	DefinirPercentual(ctx context.Context, percentual *domaincomissao.PercentualPedido, versaoLida *int) error
	ListarEntregasParaApurar(ctx context.Context, filtro EntregaFiltro) ([]*domaincomissao.EntregaParaApurar, error)
	GravarApuracao(ctx context.Context, apuracao *domaincomissao.Apuracao) (bool, error)
	Listar(ctx context.Context, filtro ComissaoFiltro) ([]*domaincomissao.Comissao, error)
}
