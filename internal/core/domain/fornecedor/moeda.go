package fornecedor

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type Moeda string

const (
	MoedaReal       Moeda = "BRL"
	MoedaDolar      Moeda = "USD"
	MoedaEuro       Moeda = "EUR"
	MoedaYuanChines Moeda = "CNY"
)

func NovaMoeda(valor string) (Moeda, error) {
	moeda := Moeda(strings.ToUpper(strings.TrimSpace(valor)))
	if moeda == "" {
		return MoedaReal, nil
	}
	if err := moeda.Validado(); err != nil {
		return "", err
	}

	return moeda, nil
}

func MoedaDe(valor string) Moeda {
	return Moeda(valor)
}

func (m Moeda) String() string {
	return string(m)
}

func (m Moeda) Validado() error {
	switch m {
	case MoedaReal, MoedaDolar, MoedaEuro, MoedaYuanChines:
		return nil
	default:
		return domain.ErroValidacao("moeda deve ser BRL, USD, EUR ou CNY")
	}
}
