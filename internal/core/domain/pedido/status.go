package pedido

import (
	"fmt"
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type Status string

const (
	StatusProposta             Status = "proposta"
	StatusConfirmado           Status = "confirmado"
	StatusParcialmenteEntregue Status = "parcialmente_entregue"
	StatusEntregue             Status = "entregue"
	StatusCancelado            Status = "cancelado"
)

var transicoesPermitidas = map[Status][]Status{
	StatusProposta:             {StatusConfirmado, StatusCancelado},
	StatusConfirmado:           {StatusParcialmenteEntregue, StatusEntregue, StatusCancelado},
	StatusParcialmenteEntregue: {StatusEntregue, StatusCancelado},
	StatusEntregue:             {},
	StatusCancelado:            {},
}

func NovoStatus(valor string) (Status, error) {
	status := Status(strings.ToLower(strings.TrimSpace(valor)))

	if _, ok := transicoesPermitidas[status]; !ok {
		return "", domain.ErroValidacao("status do pedido inválido")
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

func (s Status) Final() bool {
	return len(transicoesPermitidas[s]) == 0
}

func (s Status) EmAtendimento() bool {
	return s == StatusConfirmado || s == StatusParcialmenteEntregue
}

func erroTransicao(atual, proximo Status) error {
	return domain.ErroConflito(fmt.Sprintf("não é possível mudar o pedido de %q para %q", atual, proximo))
}
