package divida

import (
	"fmt"
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type Status string

const (
	StatusAtiva       Status = "ativa"
	StatusRenegociada Status = "renegociada"
	StatusCancelada   Status = "cancelada"
)

func NovoStatus(valor string) (Status, error) {
	status := Status(strings.ToLower(strings.TrimSpace(valor)))

	switch status {
	case StatusAtiva, StatusRenegociada, StatusCancelada:
		return status, nil
	default:
		return "", domain.ErroValidacao("status da dívida deve ser ativa, renegociada ou cancelada")
	}
}

func StatusDe(valor string) Status {
	return Status(valor)
}

func (s Status) String() string {
	return string(s)
}

func erroDividaEncerrada(status Status) error {
	return domain.ErroConflito(fmt.Sprintf("dívida %s não pode ser alterada", status))
}
