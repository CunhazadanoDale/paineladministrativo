package dre

import "time"

type Intervalo struct {
	Inicio time.Time
	Fim    time.Time
}

func (i Intervalo) InicioInstante() time.Time {
	return instanteDoInicioDoDia(i.Inicio)
}

func (i Intervalo) FimInstante() time.Time {
	return instanteDoInicioDoDia(i.Fim)
}

func instanteDoInicioDoDia(data time.Time) time.Time {
	ano, mes, dia := data.Date()

	return time.Date(ano, mes, dia, 0, 0, 0, 0, fusoDoGrupo()).UTC()
}

func (p PeriodoMensal) Intervalo() Intervalo {
	return Intervalo{Inicio: p.de.Inicio(), Fim: p.ate.Somar(1).Inicio()}
}

func (s Semana) Intervalo() Intervalo {
	return Intervalo{Inicio: s.inicio, Fim: s.inicio.AddDate(0, 0, diasDaSemana)}
}
