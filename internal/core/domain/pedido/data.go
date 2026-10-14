package pedido

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

func DataDe(momento time.Time) time.Time {
	ano, mes, dia := momento.Date()

	return time.Date(ano, mes, dia, 0, 0, 0, 0, time.UTC)
}

func dataOpcional(valor *time.Time) *time.Time {
	if valor == nil || valor.IsZero() {
		return nil
	}

	data := DataDe(*valor)

	return &data
}

func dataDoFato(valor, agora time.Time, campo rotulo) (time.Time, error) {
	if valor.IsZero() {
		return time.Time{}, domain.ErroValidacao(campo.obrigatorio())
	}

	data := DataDe(valor)
	if data.After(DataDe(agora)) {
		return time.Time{}, domain.ErroValidacao(campo.nome + " não pode estar no futuro")
	}

	return data, nil
}
