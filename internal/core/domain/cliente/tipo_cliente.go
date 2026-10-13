package cliente

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type TipoCliente string

const (
	TipoClienteConstrutora TipoCliente = "construtora"
	TipoClienteLoja        TipoCliente = "loja"
	TipoClientePF          TipoCliente = "pf"
	TipoClienteOutro       TipoCliente = "outro"
)

func NovoTipoCliente(valor string) (TipoCliente, error) {
	tipo := TipoCliente(strings.ToLower(strings.TrimSpace(valor)))
	if err := tipo.Validado(); err != nil {
		return "", err
	}

	return tipo, nil
}

func TipoClienteDe(valor string) TipoCliente {
	return TipoCliente(valor)
}

func (t TipoCliente) String() string {
	return string(t)
}

func (t TipoCliente) Validado() error {
	switch t {
	case TipoClienteConstrutora, TipoClienteLoja, TipoClientePF, TipoClienteOutro:
		return nil
	default:
		return domain.ErroValidacao("tipo de cliente deve ser construtora, loja, pf ou outro")
	}
}
