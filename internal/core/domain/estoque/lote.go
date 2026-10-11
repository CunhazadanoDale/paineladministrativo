package estoque

import (
	"time"

	"github.com/google/uuid"
)

type Lote struct {
	ID                uuid.UUID
	ProdutoID         uuid.UUID
	LocalID           uuid.UUID
	MovimentoID       uuid.UUID
	LoteOrigemID      *uuid.UUID
	Custo             *Custo
	QuantidadeInicial int
	QuantidadeAtual   int
	EntradaEm         time.Time
	CriadoEm          time.Time
}

type ConsumoLote struct {
	LoteID     uuid.UUID
	Quantidade int
	Custo      *Custo
}

func novoLote(produtoID, localID, movimentoID uuid.UUID, loteOrigemID *uuid.UUID, custo *Custo, quantidade int, entradaEm, agora time.Time) *Lote {
	return &Lote{
		ID:                uuid.New(),
		ProdutoID:         produtoID,
		LocalID:           localID,
		MovimentoID:       movimentoID,
		LoteOrigemID:      loteOrigemID,
		Custo:             custo,
		QuantidadeInicial: quantidade,
		QuantidadeAtual:   quantidade,
		EntradaEm:         entradaEm,
		CriadoEm:          agora,
	}
}

func (l *Lote) CustoDesconhecido() bool {
	return l.Custo == nil
}
