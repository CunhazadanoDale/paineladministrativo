package estoque

import (
	"time"

	"github.com/google/uuid"
)

type SaldoLocal struct {
	ProdutoID    uuid.UUID
	LocalID      uuid.UUID
	Quantidade   int
	Reservado    int
	AtualizadoEm time.Time
}

func (s SaldoLocal) Disponivel() int {
	return s.Quantidade - s.Reservado
}

func (s SaldoLocal) Vazio() bool {
	return s.Quantidade == 0
}
