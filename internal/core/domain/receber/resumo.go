package receber

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const (
	semanasPadraoDoResumo    = 8
	diasMaximosDoResumo      = 366
	diasPadraoDoAlertaBoleto = 5
	diasMaximosDoAlerta      = 90
)

type TotalResumo struct {
	Quantidade    int
	SaldoCentavos int64
}

func (t TotalResumo) Somar(outro TotalResumo) TotalResumo {
	return TotalResumo{Quantidade: t.Quantidade + outro.Quantidade, SaldoCentavos: t.SaldoCentavos + outro.SaldoCentavos}
}

type LinhaResumoSemanal struct {
	Semana     time.Time
	EmpresaID  uuid.UUID
	AReceber   TotalResumo
	Antecipado TotalResumo
	Vencido    TotalResumo
}

type AtrasoPorEmpresa struct {
	EmpresaID uuid.UUID
	Vencido   TotalResumo
}

type TotaisResumo struct {
	AReceber   TotalResumo
	Antecipado TotalResumo
	Vencido    TotalResumo
}

type ResumoContasAReceber struct {
	De        time.Time
	Ate       time.Time
	Semanas   []LinhaResumoSemanal
	Atrasados []AtrasoPorEmpresa
	Totais    TotaisResumo
}

type PeriodoResumo struct {
	De  time.Time
	Ate time.Time
}

func NovoPeriodoResumo(de, ate *time.Time, agora time.Time) (PeriodoResumo, error) {
	inicio := InicioDaSemana(Hoje(agora))
	if de != nil {
		inicio = DataDe(*de)
	}

	fim := inicio.AddDate(0, 0, semanasPadraoDoResumo*7-1)
	if ate != nil {
		fim = DataDe(*ate)
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
	dia := DataDe(data)
	recuo := (int(dia.Weekday()) + 6) % 7

	return dia.AddDate(0, 0, -recuo)
}

func NovoResumoContasAReceber(periodo PeriodoResumo, semanas []LinhaResumoSemanal, atrasados []AtrasoPorEmpresa) ResumoContasAReceber {
	resumo := ResumoContasAReceber{
		De:        periodo.De,
		Ate:       periodo.Ate,
		Semanas:   semanas,
		Atrasados: atrasados,
	}

	for _, linha := range semanas {
		resumo.Totais.AReceber = resumo.Totais.AReceber.Somar(linha.AReceber)
		resumo.Totais.Antecipado = resumo.Totais.Antecipado.Somar(linha.Antecipado)
		resumo.Totais.Vencido = resumo.Totais.Vencido.Somar(linha.Vencido)
	}
	for _, atraso := range atrasados {
		resumo.Totais.Vencido = resumo.Totais.Vencido.Somar(atraso.Vencido)
	}

	return resumo
}

func LimiteDoAlertaDeBoleto(dias *int, agora time.Time) (time.Time, error) {
	quantidade := diasPadraoDoAlertaBoleto
	if dias != nil {
		quantidade = *dias
	}
	if quantidade < 0 || quantidade > diasMaximosDoAlerta {
		return time.Time{}, domain.ErroValidacao("dias do alerta de boleto devem estar entre 0 e 90")
	}

	return Hoje(agora).AddDate(0, 0, quantidade), nil
}
