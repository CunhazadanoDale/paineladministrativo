package caixa

import (
	"context"
	"time"

	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

type RecorrenciaNaCompetencia struct {
	RecorrenciaID uuid.UUID
	Competencia   time.Time
}

type MovimentoDiarioFiltro struct {
	EmpresaIDs      []uuid.UUID
	ContaBancariaID *uuid.UUID
	De              time.Time
	Ate             time.Time
}

type ProjecaoRepository interface {
	SaldoDePartida(ctx context.Context, empresaIDs []uuid.UUID) (*domaincaixa.SaldoDePartida, error)
	TitulosAReceber(ctx context.Context, empresaIDs []uuid.UUID, ate time.Time) ([]domaincaixa.Previsto, error)
	ContasAPagar(ctx context.Context, empresaIDs []uuid.UUID, ate time.Time) ([]domaincaixa.Previsto, error)
	RecorrenciasGeradas(ctx context.Context, candidatas []RecorrenciaNaCompetencia) (map[RecorrenciaNaCompetencia]bool, error)
	MovimentoDiario(ctx context.Context, filtro MovimentoDiarioFiltro) ([]*domaincaixa.MovimentoDiario, error)
}

type DespesaRecorrenteRepository interface {
	ListarVigentes(ctx context.Context, competencia domainsolicitacao.Competencia, empresaIDs []uuid.UUID) ([]*domainsolicitacao.DespesaRecorrente, error)
}
