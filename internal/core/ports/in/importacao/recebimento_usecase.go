package importacao

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type AlocacaoInput struct {
	ImportacaoItemID uuid.UUID
	PedidoItemID     uuid.UUID
	Quantidade       int
}

type DefinirAlocacoesInput struct {
	UsuarioID    uuid.UUID
	ImportacaoID uuid.UUID
	Versao       int
	Alocacoes    []AlocacaoInput
}

type ReceberInput struct {
	UsuarioID    uuid.UUID
	ImportacaoID uuid.UUID
	Versao       int
	RecebidaEm   time.Time
	LocalID      uuid.UUID
	CambioMicro  *int64
	Recebidas    map[uuid.UUID]int
}

type RecebimentoUseCase interface {
	DefinirAlocacoes(ctx context.Context, input DefinirAlocacoesInput) (*Visao, error)
	Receber(ctx context.Context, input ReceberInput) (*Visao, error)
}
