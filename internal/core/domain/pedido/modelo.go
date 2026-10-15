package pedido

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type Modelo string

const (
	ModeloProntaEntrega Modelo = "pronta_entrega"
	ModeloImportacao    Modelo = "importacao"
)

func NovoModelo(valor string) (Modelo, error) {
	modelo := Modelo(strings.ToLower(strings.TrimSpace(valor)))

	if err := modelo.Validado(); err != nil {
		return "", err
	}

	return modelo, nil
}

func ModeloDe(valor string) Modelo {
	return Modelo(valor)
}

func (m Modelo) Validado() error {
	switch m {
	case ModeloProntaEntrega, ModeloImportacao:
		return nil
	default:
		return domain.ErroValidacao("modelo de venda deve ser pronta_entrega ou importacao")
	}
}

func (m Modelo) String() string {
	return string(m)
}
