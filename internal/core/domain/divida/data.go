package divida

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

func FimDoMes(data time.Time) time.Time {
	ano, mes, _ := data.Date()

	return time.Date(ano, mes+1, 0, 0, 0, 0, 0, time.UTC)
}

func dataOpcionalDoFato(valor *time.Time, agora time.Time, campo string) (*time.Time, error) {
	if valor == nil || valor.IsZero() {
		return nil, nil
	}

	data := DataDe(*valor)
	if data.After(Hoje(agora)) {
		return nil, domain.ErroValidacao(campo + " não pode estar no futuro")
	}

	return &data, nil
}

func dataDoFato(valor, agora time.Time, campo rotulo) (time.Time, error) {
	if valor.IsZero() {
		return time.Time{}, domain.ErroValidacao(campo.obrigatorio())
	}

	data := DataDe(valor)
	if data.After(Hoje(agora)) {
		return time.Time{}, domain.ErroValidacao(campo.nome + " não pode estar no futuro")
	}

	return data, nil
}

func formatarData(data time.Time) string {
	return data.Format("02/01/2006")
}
