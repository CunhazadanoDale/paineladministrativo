package solicitacao

import (
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type Status string

const (
	StatusPendenteAprovacao Status = "pendente_aprovacao"
	StatusAprovado          Status = "aprovado"
	StatusPago              Status = "pago"
	StatusRejeitado         Status = "rejeitado"
	StatusCancelado         Status = "cancelado"
)

var transicoesPermitidas = map[Status][]Status{
	StatusPendenteAprovacao: {StatusAprovado, StatusRejeitado, StatusCancelado},
	StatusAprovado:          {StatusPago, StatusCancelado},
	StatusPago:              {},
	StatusRejeitado:         {},
	StatusCancelado:         {},
}

func NovoStatus(valor string) (Status, error) {
	status := Status(valor)

	if _, ok := transicoesPermitidas[status]; !ok {
		return "", domain.ErroValidacao("status de solicitação inválido")
	}

	return status, nil
}

func StatusDe(valor string) Status {
	return Status(valor)
}

func (s Status) String() string {
	return string(s)
}

func (s Status) PodeTransicionarPara(proximo Status) bool {
	for _, permitido := range transicoesPermitidas[s] {
		if permitido == proximo {
			return true
		}
	}

	return false
}
