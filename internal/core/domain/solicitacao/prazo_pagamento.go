package solicitacao

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type PrazoPagamento struct {
	data time.Time
}

func NovoPrazoPagamento(data time.Time) (PrazoPagamento, error) {
	if data.IsZero() {
		return PrazoPagamento{}, domain.ErroValidacao("prazo de pagamento é obrigatório")
	}
	if inicioDoDia(data).Before(inicioDoDia(time.Now())) {
		return PrazoPagamento{}, domain.ErroValidacao("prazo de pagamento não pode ser anterior a hoje")
	}

	return PrazoPagamento{data: data.UTC()}, nil
}

func PrazoPagamentoDe(data time.Time) PrazoPagamento {
	return PrazoPagamento{data: data.UTC()}
}

func (p PrazoPagamento) Data() time.Time {
	return p.data
}

func inicioDoDia(data time.Time) time.Time {
	ano, mes, dia := data.UTC().Date()

	return time.Date(ano, mes, dia, 0, 0, 0, 0, time.UTC)
}
