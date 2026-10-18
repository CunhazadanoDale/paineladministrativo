package comissao

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
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
	Aberta          *bool
}

type AjusteFiltro struct {
	domain.PaginacaoFiltro
	Empresa         *uuid.UUID
	EmpresaIDs      []uuid.UUID
	RepresentanteID *uuid.UUID
	PedidoID        *uuid.UUID
	Competencia     *domaincomissao.Competencia
	Aberto          *bool
}

type SaldoFiltro struct {
	EmpresaID       uuid.UUID
	Ate             domaincomissao.Competencia
	RepresentanteID *uuid.UUID
}

type FechamentoFiltro struct {
	domain.PaginacaoFiltro
	Empresa         *uuid.UUID
	EmpresaIDs      []uuid.UUID
	RepresentanteID *uuid.UUID
	Competencia     *domaincomissao.Competencia
}

type GravacaoFechamento struct {
	Fechamento  *domaincomissao.Fechamento
	Solicitacao *domainsolicitacao.Solicitacao
	Historico   *domainsolicitacao.Historico
}

type ComissaoRepository interface {
	ObterPercentual(ctx context.Context, pedidoID uuid.UUID) (*domaincomissao.PercentualPedido, error)
	DefinirPercentual(ctx context.Context, percentual *domaincomissao.PercentualPedido, versaoLida *int) error
	ListarEntregasParaApurar(ctx context.Context, filtro EntregaFiltro) ([]*domaincomissao.EntregaParaApurar, error)
	GravarApuracao(ctx context.Context, apuracao *domaincomissao.Apuracao) (bool, error)
	Listar(ctx context.Context, filtro ComissaoFiltro) ([]*domaincomissao.Comissao, error)
	CriarAjuste(ctx context.Context, ajuste *domaincomissao.Ajuste) error
	ListarAjustes(ctx context.Context, filtro AjusteFiltro) ([]*domaincomissao.Ajuste, error)
	SaldosAbertos(ctx context.Context, filtro SaldoFiltro) ([]domaincomissao.SaldoAberto, error)
	Fechar(ctx context.Context, gravacao *GravacaoFechamento) error
	ObterFechamento(ctx context.Context, id uuid.UUID) (*domaincomissao.Fechamento, error)
	ListarFechamentos(ctx context.Context, filtro FechamentoFiltro) ([]*domaincomissao.Fechamento, error)
	ItensDoFechamento(ctx context.Context, fechamentoID uuid.UUID) ([]*domaincomissao.Comissao, []*domaincomissao.Ajuste, error)
}
