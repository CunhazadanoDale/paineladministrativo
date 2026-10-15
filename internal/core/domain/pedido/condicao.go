package pedido

import (
	"time"

	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
)

const (
	entradaImportacaoPontosBase  = 3000
	prazoBoletoProntaEntregaDias = 30
	prazoEntregaImportacaoDias   = 120
)

type Condicao struct {
	entrada          domaincliente.Percentual
	prazoBoletoDias  *int
	prazoEntregaDias *int
}

func CondicaoDoModelo(modelo Modelo) Condicao {
	if modelo == ModeloImportacao {
		prazo := prazoEntregaImportacaoDias

		return Condicao{
			entrada:          domaincliente.PercentualDe(entradaImportacaoPontosBase),
			prazoEntregaDias: &prazo,
		}
	}

	prazo := prazoBoletoProntaEntregaDias

	return Condicao{
		entrada:         domaincliente.PercentualDe(0),
		prazoBoletoDias: &prazo,
	}
}

func CondicaoDe(entradaPontosBase int, prazoBoletoDias, prazoEntregaDias *int) Condicao {
	return Condicao{
		entrada:          domaincliente.PercentualDe(entradaPontosBase),
		prazoBoletoDias:  copiarInteiro(prazoBoletoDias),
		prazoEntregaDias: copiarInteiro(prazoEntregaDias),
	}
}

func (c Condicao) Entrada() domaincliente.Percentual {
	return c.entrada
}

func (c Condicao) PrazoBoletoDias() *int {
	return copiarInteiro(c.prazoBoletoDias)
}

func (c Condicao) PrazoEntregaDias() *int {
	return copiarInteiro(c.prazoEntregaDias)
}

func (c Condicao) PrazoLimiteDaEntrada(pagaEm time.Time) *time.Time {
	if c.prazoEntregaDias == nil {
		return nil
	}

	limite := DataDe(pagaEm).AddDate(0, 0, *c.prazoEntregaDias)

	return &limite
}

func copiarInteiro(valor *int) *int {
	if valor == nil {
		return nil
	}

	copia := *valor

	return &copia
}
