package caixa

import (
	"context"
	"time"

	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	"github.com/google/uuid"
)

type FechamentoFiltro struct {
	EmpresaIDs      []uuid.UUID
	ContaBancariaID *uuid.UUID
	Competencia     *time.Time
	Limite          int
	Deslocamento    int
}

type GravacaoFechamento struct {
	Fechamento       *domaincaixa.Fechamento
	UltimoID         *uuid.UUID
	VersaoDaAbertura int
	SaldoBancoLido   *int64
}

type FechamentoRepository interface {
	Ultimo(ctx context.Context, contaBancariaID uuid.UUID) (*domaincaixa.Fechamento, error)
	MovimentoDoMes(ctx context.Context, contaBancariaID uuid.UUID, competencia, dataCorte time.Time) (*domaincaixa.MovimentoDoMes, error)
	Fechar(ctx context.Context, gravacao *GravacaoFechamento) error
	Obter(ctx context.Context, id uuid.UUID) (*domaincaixa.Fechamento, error)
	Listar(ctx context.Context, filtro FechamentoFiltro) ([]*domaincaixa.Fechamento, error)
	Reabrir(ctx context.Context, fechamento *domaincaixa.Fechamento) error
}
