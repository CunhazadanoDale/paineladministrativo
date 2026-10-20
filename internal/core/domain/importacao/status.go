package importacao

import (
	"fmt"
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type Status string

const (
	StatusAberta        Status = "aberta"
	StatusEmbarcada     Status = "embarcada"
	StatusDesembaracada Status = "desembaracada"
	StatusRecebida      Status = "recebida"
	StatusCancelada     Status = "cancelada"
)

var transicoesPermitidas = map[Status][]Status{
	StatusAberta:        {StatusEmbarcada, StatusCancelada},
	StatusEmbarcada:     {StatusDesembaracada, StatusCancelada},
	StatusDesembaracada: {StatusRecebida, StatusCancelada},
	StatusRecebida:      {},
	StatusCancelada:     {},
}

func NovoStatus(valor string) (Status, error) {
	status := Status(strings.ToLower(strings.TrimSpace(valor)))

	if _, ok := transicoesPermitidas[status]; !ok {
		return "", domain.ErroValidacao("status da importação deve ser aberta, embarcada, desembaracada, recebida ou cancelada")
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

func (s Status) Editavel() bool {
	return s == StatusAberta || s == StatusEmbarcada
}

func erroTransicao(atual, proximo Status) error {
	return domain.ErroConflito(fmt.Sprintf("não é possível mudar a importação de %q para %q", atual, proximo))
}
