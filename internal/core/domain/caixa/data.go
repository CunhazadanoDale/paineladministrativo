package caixa

import "time"

const deslocamentoDoFusoDoGrupo = -3 * 60 * 60

func fusoDoGrupo() *time.Location {
	return time.FixedZone("BRT", deslocamentoDoFusoDoGrupo)
}

func DataDe(momento time.Time) time.Time {
	ano, mes, dia := momento.Date()

	return time.Date(ano, mes, dia, 0, 0, 0, 0, time.UTC)
}

func Hoje(agora time.Time) time.Time {
	return DataDe(agora.In(fusoDoGrupo()))
}
