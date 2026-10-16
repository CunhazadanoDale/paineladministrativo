package notafiscal

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/notafiscal"
	"github.com/google/uuid"
)

type Alteracao struct {
	Nota            *domainnotafiscal.NotaFiscal
	VersaoLida      int
	StatusAnterior  domainnotafiscal.Status
	ConferirVinculo bool
	ItensAlterados  []*domainnotafiscal.Item
	Evento          *domainnotafiscal.Evento
}

type NotaFiscalFiltro struct {
	domain.PaginacaoFiltro
	Empresa         *uuid.UUID
	EmpresaIDs      []uuid.UUID
	ClienteID       *uuid.UUID
	RepresentanteID *uuid.UUID
	PedidoID        *uuid.UUID
	Natureza        domainnotafiscal.Natureza
	Status          domainnotafiscal.Status
	EntreEmpresas   *bool
	SemPedido       bool
	EmitidaDe       *time.Time
	EmitidaAte      *time.Time
	Numero          *int
	Chave           string
}

type ItemPendenteFiltro struct {
	domain.PaginacaoFiltro
	Empresa    *uuid.UUID
	EmpresaIDs []uuid.UUID
}

type EntregaSemNotaFiltro struct {
	domain.PaginacaoFiltro
	Empresa         *uuid.UUID
	EmpresaIDs      []uuid.UUID
	ClienteID       *uuid.UUID
	RepresentanteID *uuid.UUID
}

type NotaFiscalRepository interface {
	Criar(ctx context.Context, nota *domainnotafiscal.NotaFiscal) error
	Obter(ctx context.Context, id uuid.UUID) (*domainnotafiscal.NotaFiscal, error)
	ObterPorChave(ctx context.Context, chave string) (*domainnotafiscal.NotaFiscal, error)
	Listar(ctx context.Context, filtro NotaFiscalFiltro) ([]*domainnotafiscal.NotaFiscal, error)
	Atualizar(ctx context.Context, alteracao *Alteracao) error
	ListarEventos(ctx context.Context, notaFiscalID uuid.UUID) ([]*domainnotafiscal.Evento, error)
	ListarItensSemProduto(ctx context.Context, filtro ItemPendenteFiltro) ([]*domainnotafiscal.ItemPendente, error)
	ListarEntregasSemNota(ctx context.Context, filtro EntregaSemNotaFiltro) ([]*domainnotafiscal.EntregaSemNota, error)
	ArquivoVinculado(ctx context.Context, arquivoID uuid.UUID) (bool, error)
}
