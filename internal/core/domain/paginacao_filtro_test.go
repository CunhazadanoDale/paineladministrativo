package domain_test

import (
	"math"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

func TestPaginacaoNormalizada(t *testing.T) {
	casos := []struct {
		entrada  domain.PaginacaoFiltro
		esperado domain.PaginacaoFiltro
	}{
		{domain.PaginacaoFiltro{}, domain.PaginacaoFiltro{Page: 1, Size: domain.TamanhoPaginaPadrao}},
		{domain.PaginacaoFiltro{Page: -3, Size: 500}, domain.PaginacaoFiltro{Page: 1, Size: domain.TamanhoPaginaMaximo}},
		{domain.PaginacaoFiltro{Page: math.MaxInt, Size: domain.TamanhoPaginaMaximo}, domain.PaginacaoFiltro{Page: domain.PaginaMaxima, Size: domain.TamanhoPaginaMaximo}},
		{domain.PaginacaoFiltro{Page: 7, Size: 30}, domain.PaginacaoFiltro{Page: 7, Size: 30}},
	}

	for _, caso := range casos {
		normalizada := caso.entrada.Normalizada()
		if normalizada != caso.esperado {
			t.Errorf("%+v normalizou para %+v, esperado %+v", caso.entrada, normalizada, caso.esperado)
		}
		if deslocamento := (normalizada.Page - 1) * normalizada.Size; deslocamento < 0 {
			t.Errorf("%+v gerou deslocamento negativo %d", caso.entrada, deslocamento)
		}
	}
}
