package caixa

import (
	"slices"
	"time"
)

type ParAutomatico struct {
	Lancamento *LancamentoExtrato
	Movimento  *MovimentoSistema
}

type ResultadoDosPares struct {
	Pares    []ParAutomatico
	Ambiguos int
	SemPar   int
}

type chaveDoPar struct {
	data  time.Time
	valor int64
}

func ParesExatos(lancamentos []*LancamentoExtrato, movimentos []*MovimentoSistema) ResultadoDosPares {
	lancamentosPorChave := map[chaveDoPar]int{}
	for _, lancamento := range lancamentos {
		if lancamento.ConciliacaoID == nil {
			lancamentosPorChave[chaveDoPar{DataDe(lancamento.Data), lancamento.ValorCentavos}]++
		}
	}

	movimentosPorChave := map[chaveDoPar][]*MovimentoSistema{}
	for _, movimento := range movimentos {
		if movimento.Pendente() {
			chave := chaveDoPar{DataDe(movimento.Data), movimento.ValorCentavos}
			movimentosPorChave[chave] = append(movimentosPorChave[chave], movimento)
		}
	}

	var resultado ResultadoDosPares
	for _, lancamento := range lancamentos {
		if lancamento.ConciliacaoID != nil {
			continue
		}

		chave := chaveDoPar{DataDe(lancamento.Data), lancamento.ValorCentavos}
		candidatos := movimentosPorChave[chave]
		switch {
		case len(candidatos) == 0:
			resultado.SemPar++
		case len(candidatos) == 1 && lancamentosPorChave[chave] == 1:
			resultado.Pares = append(resultado.Pares, ParAutomatico{Lancamento: lancamento, Movimento: candidatos[0]})
		default:
			resultado.Ambiguos++
		}
	}

	return resultado
}

func OrdenarSugestoes(lancamento *LancamentoExtrato, movimentos []*MovimentoSistema) []*MovimentoSistema {
	sugestoes := make([]*MovimentoSistema, 0, len(movimentos))
	for _, movimento := range movimentos {
		if movimento.Pendente() && movimento.ContaBancariaID == lancamento.ContaBancariaID &&
			movimento.ValorCentavos == lancamento.ValorCentavos && DentroDaJanela(lancamento.Data, movimento.Data) {
			sugestoes = append(sugestoes, movimento)
		}
	}

	slices.SortStableFunc(sugestoes, func(a, b *MovimentoSistema) int {
		return distanciaEmDias(lancamento.Data, a.Data) - distanciaEmDias(lancamento.Data, b.Data)
	})

	return sugestoes
}

func distanciaEmDias(de, ate time.Time) int {
	dias := int(DataDe(ate).Sub(DataDe(de)).Hours() / 24)
	if dias < 0 {
		return -dias
	}

	return dias
}
