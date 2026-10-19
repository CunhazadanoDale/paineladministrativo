package pedido

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
)

const (
	entradaImportacaoPontosBase  = 3000
	prazoBoletoProntaEntregaDias = 30
	prazoEntregaImportacaoDias   = 120
	prazoBoletoMaximoDias        = 365
	prazoEntregaMaximoDias       = 730
)

type CondicaoInput struct {
	EntradaBP        *int
	PrazoBoletoDias  *int
	PrazoEntregaDias *int
}

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

func NovaCondicao(modelo Modelo, input *CondicaoInput) (Condicao, error) {
	condicao := CondicaoDoModelo(modelo)
	if input == nil {
		return condicao, nil
	}

	if input.EntradaBP != nil {
		entrada, err := domaincliente.NovoPercentual(*input.EntradaBP)
		if err != nil {
			return Condicao{}, domain.ErroValidacao("entrada deve estar entre 0 e 10000 pontos-base (0% a 100%)")
		}
		condicao.entrada = entrada
	}
	if input.PrazoBoletoDias != nil {
		if *input.PrazoBoletoDias < 1 || *input.PrazoBoletoDias > prazoBoletoMaximoDias {
			return Condicao{}, domain.ErroValidacao("prazo do boleto deve estar entre 1 e 365 dias")
		}
		condicao.prazoBoletoDias = copiarInteiro(input.PrazoBoletoDias)
	}
	if input.PrazoEntregaDias != nil {
		if modelo != ModeloImportacao {
			return Condicao{}, domain.ErroValidacao("prazo de entrega em dias só vale na importação; na pronta entrega use o prazo-limite")
		}
		if *input.PrazoEntregaDias < 1 || *input.PrazoEntregaDias > prazoEntregaMaximoDias {
			return Condicao{}, domain.ErroValidacao("prazo de entrega deve estar entre 1 e 730 dias")
		}
		condicao.prazoEntregaDias = copiarInteiro(input.PrazoEntregaDias)
	}

	return condicao, nil
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
