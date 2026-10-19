package caixa

import (
	"context"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	"github.com/google/uuid"
)

const LimiteDaConciliacaoAutomatica = 500

type ReferenciaInput struct {
	Tipo string
	ID   uuid.UUID
}

type ConciliarInput struct {
	UsuarioID    uuid.UUID
	LancamentoID uuid.UUID
	Movimentos   []ReferenciaInput
}

type ConciliarAutomaticoInput struct {
	UsuarioID       uuid.UUID
	ContaBancariaID uuid.UUID
	De              time.Time
	Ate             time.Time
}

type ResultadoAutomatico struct {
	Conciliadas  int
	Ambiguos     int
	SemPar       int
	Falhas       int
	RestamLinhas bool
}

type DesfazerConciliacaoInput struct {
	UsuarioID     uuid.UUID
	ConciliacaoID uuid.UUID
	Motivo        string
}

type PendenciasInput struct {
	UsuarioID       uuid.UUID
	ContaBancariaID uuid.UUID
	De              *time.Time
	Ate             *time.Time
	Filtro          domain.PaginacaoFiltro
}

type PagamentosSemContaInput struct {
	UsuarioID uuid.UUID
	Empresa   *uuid.UUID
	De        *time.Time
	Ate       *time.Time
	Filtro    domain.PaginacaoFiltro
}

type ConciliacaoUseCase interface {
	Sugestoes(ctx context.Context, lancamentoID, usuarioID uuid.UUID) ([]*domaincaixa.MovimentoSistema, error)
	Conciliar(ctx context.Context, input ConciliarInput) (*domaincaixa.Conciliacao, error)
	ConciliarAutomatico(ctx context.Context, input ConciliarAutomaticoInput) (*ResultadoAutomatico, error)
	Obter(ctx context.Context, conciliacaoID, usuarioID uuid.UUID) (*domaincaixa.Conciliacao, error)
	Desfazer(ctx context.Context, input DesfazerConciliacaoInput) (*domaincaixa.Conciliacao, error)
	Pendencias(ctx context.Context, input PendenciasInput) ([]*domaincaixa.MovimentoSistema, error)
	PagamentosSemConta(ctx context.Context, input PagamentosSemContaInput) ([]*domaincaixa.MovimentoSistema, error)
}
