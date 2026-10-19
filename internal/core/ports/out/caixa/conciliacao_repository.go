package caixa

import (
	"context"
	"time"

	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	"github.com/google/uuid"
)

type MovimentoFiltro struct {
	ContaBancariaID uuid.UUID
	De              *time.Time
	Ate             *time.Time
	ValorCentavos   *int64
	SoPendentes     bool
	Limite          int
	Deslocamento    int
}

type SemContaFiltro struct {
	EmpresaIDs   []uuid.UUID
	De           *time.Time
	Ate          *time.Time
	Limite       int
	Deslocamento int
}

type ConciliacaoRepository interface {
	ObterLancamento(ctx context.Context, id uuid.UUID) (*domaincaixa.LancamentoExtrato, error)
	LancamentosPendentes(ctx context.Context, contaBancariaID uuid.UUID, de, ate time.Time, limite int) ([]*domaincaixa.LancamentoExtrato, error)
	Movimentos(ctx context.Context, filtro MovimentoFiltro) ([]*domaincaixa.MovimentoSistema, error)
	MovimentosPorReferencia(ctx context.Context, contaBancariaID uuid.UUID, referencias []domaincaixa.ReferenciaMovimento) ([]*domaincaixa.MovimentoSistema, error)
	PagamentosSemConta(ctx context.Context, filtro SemContaFiltro) ([]*domaincaixa.MovimentoSistema, error)
	Criar(ctx context.Context, conciliacao *domaincaixa.Conciliacao) error
	Obter(ctx context.Context, id uuid.UUID) (*domaincaixa.Conciliacao, error)
	Desfazer(ctx context.Context, conciliacao *domaincaixa.Conciliacao) error
}
