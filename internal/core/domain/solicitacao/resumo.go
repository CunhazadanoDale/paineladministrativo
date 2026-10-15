package solicitacao

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const (
	semanasPadraoDoResumo = 8
	diasMaximosDoResumo   = 366
)

type TotalResumo struct {
	Quantidade    int
	ValorCentavos int64
}

func (t TotalResumo) Somar(outro TotalResumo) TotalResumo {
	return TotalResumo{Quantidade: t.Quantidade + outro.Quantidade, ValorCentavos: t.ValorCentavos + outro.ValorCentavos}
}

type LinhaResumoSemanal struct {
	Semana           time.Time
	EmpresaID        uuid.UUID
	Previsto         TotalResumo
	APagar           TotalResumo
	Vencido          TotalResumo
	EstimadoCentavos int64
}

type AtrasoPorEmpresa struct {
	EmpresaID uuid.UUID
	Vencido   TotalResumo
}

type TotaisResumo struct {
	Previsto         TotalResumo
	APagar           TotalResumo
	Vencido          TotalResumo
	EstimadoCentavos int64
}

type ResumoContasAPagar struct {
	De        time.Time
	Ate       time.Time
	Semanas   []LinhaResumoSemanal
	Atrasadas []AtrasoPorEmpresa
	Totais    TotaisResumo
}

type PeriodoResumo struct {
	De  time.Time
	Ate time.Time
}

func NovoPeriodoResumo(de, ate *time.Time, agora time.Time) (PeriodoResumo, error) {
	inicio := InicioDaSemana(agora)
	if de != nil {
		inicio = inicioDoDia(*de)
	}

	fim := inicio.AddDate(0, 0, semanasPadraoDoResumo*7-1)
	if ate != nil {
		fim = inicioDoDia(*ate)
	}

	if fim.Before(inicio) {
		return PeriodoResumo{}, domain.ErroValidacao("fim do resumo não pode ser anterior ao início")
	}
	if fim.Sub(inicio) > diasMaximosDoResumo*24*time.Hour {
		return PeriodoResumo{}, domain.ErroValidacao("intervalo do resumo deve ter no máximo 1 ano")
	}

	return PeriodoResumo{De: inicio, Ate: fim}, nil
}

func InicioDaSemana(data time.Time) time.Time {
	dia := inicioDoDia(data)
	recuo := (int(dia.Weekday()) + 6) % 7

	return dia.AddDate(0, 0, -recuo)
}

func NovoResumoContasAPagar(periodo PeriodoResumo, semanas []LinhaResumoSemanal, atrasadas []AtrasoPorEmpresa) ResumoContasAPagar {
	resumo := ResumoContasAPagar{
		De:        periodo.De,
		Ate:       periodo.Ate,
		Semanas:   semanas,
		Atrasadas: atrasadas,
	}

	for _, linha := range semanas {
		resumo.Totais.Previsto = resumo.Totais.Previsto.Somar(linha.Previsto)
		resumo.Totais.APagar = resumo.Totais.APagar.Somar(linha.APagar)
		resumo.Totais.Vencido = resumo.Totais.Vencido.Somar(linha.Vencido)
		resumo.Totais.EstimadoCentavos += linha.EstimadoCentavos
	}
	for _, atraso := range atrasadas {
		resumo.Totais.Vencido = resumo.Totais.Vencido.Somar(atraso.Vencido)
	}

	return resumo
}

type Pendencias struct {
	Aprovacao  *int
	Financeiro *int
	Vencidas   *int
}
