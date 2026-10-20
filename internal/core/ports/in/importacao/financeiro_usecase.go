package importacao

import (
	"context"
	"time"

	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	"github.com/google/uuid"
)

type DadosParcelaInput struct {
	Descricao            string
	ValorMoedaCentesimos int64
	VencimentoPrevisto   time.Time
}

type IncluirParcelaInput struct {
	UsuarioID    uuid.UUID
	ImportacaoID uuid.UUID
	Versao       int
	Dados        DadosParcelaInput
}

type AlterarParcelaInput struct {
	UsuarioID    uuid.UUID
	ImportacaoID uuid.UUID
	ParcelaID    uuid.UUID
	Versao       int
	Dados        DadosParcelaInput
}

type ParcelaInput struct {
	UsuarioID    uuid.UUID
	ImportacaoID uuid.UUID
	ParcelaID    uuid.UUID
	Versao       int
}

type FecharCambioInput struct {
	UsuarioID             uuid.UUID
	ImportacaoID          uuid.UUID
	ParcelaID             uuid.UUID
	Versao                int
	CambioMicro           *int64
	Vencimento            time.Time
	CategoriaFinanceiraID uuid.UUID
}

type LancarCustoInput struct {
	UsuarioID             uuid.UUID
	ImportacaoID          uuid.UUID
	Versao                int
	Tipo                  string
	Descricao             string
	FornecedorID          uuid.UUID
	ValorCentavos         int64
	Vencimento            time.Time
	CompoeCusto           bool
	CategoriaFinanceiraID uuid.UUID
}

type CustoInput struct {
	UsuarioID    uuid.UUID
	ImportacaoID uuid.UUID
	CustoID      uuid.UUID
	Versao       int
}

type FinanceiroUseCase interface {
	IncluirParcela(ctx context.Context, input IncluirParcelaInput) (*Visao, error)
	AlterarParcela(ctx context.Context, input AlterarParcelaInput) (*Visao, error)
	RemoverParcela(ctx context.Context, input ParcelaInput) (*Visao, error)
	FecharCambio(ctx context.Context, input FecharCambioInput) (*Visao, error)
	ReabrirParcela(ctx context.Context, input ParcelaInput) (*Visao, error)
	LancarCusto(ctx context.Context, input LancarCustoInput) (*Visao, error)
	CancelarCusto(ctx context.Context, input CustoInput) (*Visao, error)
	CustoPrevisto(ctx context.Context, importacaoID, usuarioID uuid.UUID, cambioMicro *int64) (*domainimportacao.CalculoCusto, error)
}
