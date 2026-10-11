package estoque

import (
	"bytes"
	"slices"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

func ordenarFIFO(lotes []*Lote) []*Lote {
	ordenados := slices.Clone(lotes)
	slices.SortStableFunc(ordenados, func(a, b *Lote) int {
		if comparacao := a.EntradaEm.Compare(b.EntradaEm); comparacao != 0 {
			return comparacao
		}

		return bytes.Compare(a.ID[:], b.ID[:])
	})

	return ordenados
}

func consumirFIFO(lotes []*Lote, quantidade int) ([]ConsumoLote, error) {
	consumos := make([]ConsumoLote, 0, 1)
	restante := quantidade

	for _, lote := range ordenarFIFO(lotes) {
		if restante == 0 {
			break
		}
		if lote.QuantidadeAtual <= 0 {
			continue
		}

		consumido := min(lote.QuantidadeAtual, restante)
		consumos = append(consumos, ConsumoLote{
			LoteID:     lote.ID,
			Quantidade: consumido,
			Custo:      lote.Custo,
		})
		restante -= consumido
	}

	if restante > 0 {
		return nil, domain.ErroConflito("lotes do local não cobrem o saldo, recarregue e tente novamente")
	}

	return consumos, nil
}
