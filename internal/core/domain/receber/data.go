package receber

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

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

func dataObrigatoria(valor time.Time, campo rotulo) (time.Time, error) {
	if valor.IsZero() {
		return time.Time{}, domain.ErroValidacao(campo.obrigatorio())
	}

	return DataDe(valor), nil
}

func momentoDoFato(valor, agora time.Time, campo rotulo) (time.Time, error) {
	if valor.IsZero() {
		return time.Time{}, domain.ErroValidacao(campo.obrigatorio())
	}
	if valor.After(agora) {
		return time.Time{}, domain.ErroValidacao(campo.nome + " não pode estar no futuro")
	}

	return valor.UTC(), nil
}
