package notafiscal

import (
	"time"

	"github.com/google/uuid"
)

type ItemPendente struct {
	NotaFiscalID uuid.UUID
	EmpresaID    uuid.UUID
	Chave        Chave
	NumeroNota   int
	Serie        int
	EmitidaEm    time.Time
	Item         *Item
}

type EntregaSemNota struct {
	EntregaID      uuid.UUID
	PedidoID       uuid.UUID
	EmpresaID      uuid.UUID
	PedidoNumero   int
	ClienteID      uuid.UUID
	Status         string
	SaiuEm         *time.Time
	EntregueEm     *time.Time
	Transportadora *string
}
