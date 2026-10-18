package receber

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type Status string

const (
	StatusCarteira   Status = "carteira"
	StatusEmBanco    Status = "em_banco"
	StatusAntecipado Status = "antecipado"
	StatusRecebido   Status = "recebido"
	StatusCancelado  Status = "cancelado"
)

func NovoStatus(valor string) (Status, error) {
	status := Status(strings.ToLower(strings.TrimSpace(valor)))

	switch status {
	case StatusCarteira, StatusEmBanco, StatusAntecipado, StatusRecebido, StatusCancelado:
		return status, nil
	default:
		return "", domain.ErroValidacao("status do título deve ser carteira, em_banco, antecipado, recebido ou cancelado")
	}
}

func StatusDe(valor string) Status {
	return Status(valor)
}

func (s Status) String() string {
	return string(s)
}

func (s Status) EmAberto() bool {
	return s == StatusCarteira || s == StatusEmBanco || s == StatusAntecipado
}

func (s Status) NaEmpresa() bool {
	return s == StatusCarteira || s == StatusEmBanco
}
