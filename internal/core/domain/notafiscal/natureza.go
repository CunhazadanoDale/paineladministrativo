package notafiscal

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type Natureza string

const (
	NaturezaVenda         Natureza = "venda"
	NaturezaRemessa       Natureza = "remessa"
	NaturezaTransferencia Natureza = "transferencia"
	NaturezaDevolucao     Natureza = "devolucao"
)

var naturezasValidas = map[Natureza]bool{
	NaturezaVenda:         true,
	NaturezaRemessa:       true,
	NaturezaTransferencia: true,
	NaturezaDevolucao:     true,
}

var cfopsDeRemessa = map[string]bool{
	"5116": true, "5117": true, "6116": true, "6117": true,
	"5905": true, "6905": true,
	"5910": true, "5911": true, "5912": true, "5913": true, "5914": true, "5915": true, "5916": true, "5917": true,
	"6910": true, "6911": true, "6912": true, "6913": true, "6914": true, "6915": true, "6916": true, "6917": true,
	"5923": true, "6923": true,
	"5934": true, "6934": true,
	"5949": true, "6949": true,
}

func NovaNatureza(valor string) (Natureza, error) {
	natureza := Natureza(strings.ToLower(strings.TrimSpace(valor)))
	if !naturezasValidas[natureza] {
		return "", domain.ErroValidacao("natureza deve ser venda, remessa, transferencia ou devolucao")
	}

	return natureza, nil
}

func NaturezaDe(valor string) Natureza {
	return Natureza(valor)
}

func ClassificarNatureza(entreEmpresas bool, finalidade Finalidade, cfopPredominante string) Natureza {
	switch {
	case entreEmpresas:
		return NaturezaTransferencia
	case finalidade == FinalidadeDevolucao:
		return NaturezaDevolucao
	case CFOPDeRemessa(cfopPredominante):
		return NaturezaRemessa
	default:
		return NaturezaVenda
	}
}

func CFOPDeRemessa(cfop string) bool {
	return cfopsDeRemessa[cfop]
}

func (n Natureza) String() string {
	return string(n)
}

func (n Natureza) GeraTitulo() bool {
	return n == NaturezaVenda
}
