package notafiscal

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type Status string

const (
	StatusAutorizada Status = "autorizada"
	StatusCancelada  Status = "cancelada"
)

func NovoStatus(valor string) (Status, error) {
	status := Status(strings.ToLower(strings.TrimSpace(valor)))
	if status != StatusAutorizada && status != StatusCancelada {
		return "", domain.ErroValidacao("status da nota fiscal deve ser autorizada ou cancelada")
	}

	return status, nil
}

func StatusDe(valor string) Status {
	return Status(valor)
}

func (s Status) String() string {
	return string(s)
}

type TipoEvento string

const (
	TipoEventoCancelamento  TipoEvento = "cancelamento"
	TipoEventoCartaCorrecao TipoEvento = "carta_correcao"
)

var tiposEventoPorCodigo = map[string]TipoEvento{
	"110111": TipoEventoCancelamento,
	"110110": TipoEventoCartaCorrecao,
}

func TipoEventoDoCodigo(codigo string) (TipoEvento, error) {
	tipo, ok := tiposEventoPorCodigo[strings.TrimSpace(codigo)]
	if !ok {
		return "", domain.ErroValidacao("tipo de evento não suportado: use cancelamento ou carta de correção")
	}

	return tipo, nil
}

func TipoEventoDe(valor string) TipoEvento {
	return TipoEvento(valor)
}

func (t TipoEvento) String() string {
	return string(t)
}
