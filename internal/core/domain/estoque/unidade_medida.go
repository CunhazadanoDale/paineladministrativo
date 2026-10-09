package estoque

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type UnidadeMedida string

const (
	UnidadeMedidaUn  UnidadeMedida = "un"
	UnidadeMedidaM   UnidadeMedida = "m"
	UnidadeMedidaM2  UnidadeMedida = "m2"
	UnidadeMedidaM3  UnidadeMedida = "m3"
	UnidadeMedidaKg  UnidadeMedida = "kg"
	UnidadeMedidaT   UnidadeMedida = "t"
	UnidadeMedidaCx  UnidadeMedida = "cx"
	UnidadeMedidaSc  UnidadeMedida = "sc"
	UnidadeMedidaPct UnidadeMedida = "pct"
	UnidadeMedidaLt  UnidadeMedida = "lt"
)

var unidadesMedidaValidas = []UnidadeMedida{
	UnidadeMedidaUn,
	UnidadeMedidaM,
	UnidadeMedidaM2,
	UnidadeMedidaM3,
	UnidadeMedidaKg,
	UnidadeMedidaT,
	UnidadeMedidaCx,
	UnidadeMedidaSc,
	UnidadeMedidaPct,
	UnidadeMedidaLt,
}

func NovaUnidadeMedida(valor string) (UnidadeMedida, error) {
	unidade := UnidadeMedida(strings.ToLower(strings.TrimSpace(valor)))

	if err := unidade.Validado(); err != nil {
		return "", err
	}

	return unidade, nil
}

func UnidadeMedidaDe(valor string) UnidadeMedida {
	return UnidadeMedida(valor)
}

func (u UnidadeMedida) String() string {
	return string(u)
}

func (u UnidadeMedida) Validado() error {
	for _, valida := range unidadesMedidaValidas {
		if u == valida {
			return nil
		}
	}

	return domain.ErroValidacao("unidade de medida deve ser un, m, m2, m3, kg, t, cx, sc, pct ou lt")
}
