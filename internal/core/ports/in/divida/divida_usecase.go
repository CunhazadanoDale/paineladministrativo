package divida

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaindivida "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/divida"
	"github.com/google/uuid"
)

type DadosInformativosInput struct {
	CredorTipo             string
	Descricao              string
	Contrato               *string
	ValorOriginalCentavos  *int64
	ContratadaEm           *time.Time
	TaxaBP                 *int
	TaxaPeriodo            string
	Indexador              string
	ContaLiberacaoID       *uuid.UUID
	LiberadaEm             *time.Time
	CategoriaAmortizacaoID uuid.UUID
	CategoriaJurosID       uuid.UUID
	Observacoes            string
}

type ParcelaInput struct {
	Vencimento          time.Time
	AmortizacaoCentavos int64
	JurosCentavos       int64
}

type NovaDividaInput struct {
	FornecedorID         uuid.UUID
	Dados                DadosInformativosInput
	SaldoInicialCentavos int64
	DataSaldo            time.Time
	Parcelas             []ParcelaInput
}

type CriarDividaInput struct {
	NovaDividaInput
	UsuarioID uuid.UUID
	EmpresaID uuid.UUID
}

type RenegociarDividaInput struct {
	UsuarioID uuid.UUID
	DividaID  uuid.UUID
	Versao    int
	Motivo    string
	Nova      NovaDividaInput
}

type ReabrirParcelaInput struct {
	UsuarioID uuid.UUID
	DividaID  uuid.UUID
	ParcelaID uuid.UUID
	Versao    int
}

type AlterarDividaInput struct {
	UsuarioID uuid.UUID
	DividaID  uuid.UUID
	Versao    int
	Dados     DadosInformativosInput
}

type AlterarParcelaInput struct {
	UsuarioID     uuid.UUID
	DividaID      uuid.UUID
	ParcelaID     uuid.UUID
	Versao        int
	Vencimento    time.Time
	JurosCentavos int64
}

type CancelarDividaInput struct {
	UsuarioID uuid.UUID
	DividaID  uuid.UUID
	Versao    int
	Motivo    string
}

type ListarDividasInput struct {
	UsuarioID    uuid.UUID
	Empresa      *uuid.UUID
	FornecedorID *uuid.UUID
	CredorTipo   string
	Status       string
	Filtro       domain.PaginacaoFiltro
}

type GerarContasInput struct {
	UsuarioID   uuid.UUID
	Competencia string
	Empresa     *uuid.UUID
}

type GeracaoDaDivida struct {
	DividaID uuid.UUID
	Parcelas int
	Contas   int
}

type DividaRecusada struct {
	DividaID uuid.UUID
	Motivo   string
}

type ResultadoGeracao struct {
	Competencia string
	Geradas     []GeracaoDaDivida
	Recusadas   []DividaRecusada
}

type Visao struct {
	Divida *domaindivida.Divida
	Hoje   time.Time
}

type DividaUseCase interface {
	Criar(ctx context.Context, input CriarDividaInput) (*Visao, error)
	Obter(ctx context.Context, dividaID, usuarioID uuid.UUID) (*Visao, error)
	Listar(ctx context.Context, input ListarDividasInput) ([]*domaindivida.Divida, error)
	Alterar(ctx context.Context, input AlterarDividaInput) (*Visao, error)
	AlterarParcela(ctx context.Context, input AlterarParcelaInput) (*Visao, error)
	Cancelar(ctx context.Context, input CancelarDividaInput) (*Visao, error)
	Renegociar(ctx context.Context, input RenegociarDividaInput) (*Visao, error)
	ReabrirParcela(ctx context.Context, input ReabrirParcelaInput) (*Visao, error)
	ListarHistorico(ctx context.Context, dividaID, usuarioID uuid.UUID) ([]*domaindivida.Historico, error)
	GerarContas(ctx context.Context, input GerarContasInput) (ResultadoGeracao, error)
}
