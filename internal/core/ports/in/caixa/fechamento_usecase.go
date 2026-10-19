package caixa

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	"github.com/google/uuid"
)

type PreviaInput struct {
	UsuarioID       uuid.UUID
	ContaBancariaID uuid.UUID
	Competencia     time.Time
}

type FecharInput struct {
	UsuarioID          uuid.UUID
	ContaBancariaID    uuid.UUID
	Competencia        time.Time
	SaldoBancoCentavos *int64
}

type ReabrirInput struct {
	UsuarioID    uuid.UUID
	FechamentoID uuid.UUID
	Motivo       string
}

type ListarFechamentosInput struct {
	UsuarioID       uuid.UUID
	Empresa         *uuid.UUID
	ContaBancariaID *uuid.UUID
	Competencia     *time.Time
	Filtro          domain.PaginacaoFiltro
}

type DetalheFechamento struct {
	Fechamento              *domaincaixa.Fechamento
	ConciliacoesInvalidadas int
	MovimentosSemLinha      []*domaincaixa.MovimentoSistema
}

type FechamentoUseCase interface {
	Previa(ctx context.Context, input PreviaInput) (*domaincaixa.PreviaFechamento, error)
	Fechar(ctx context.Context, input FecharInput) (*DetalheFechamento, error)
	Obter(ctx context.Context, fechamentoID, usuarioID uuid.UUID) (*DetalheFechamento, error)
	Listar(ctx context.Context, input ListarFechamentosInput) ([]*domaincaixa.Fechamento, error)
	Reabrir(ctx context.Context, input ReabrirInput) (*domaincaixa.Fechamento, error)
}
