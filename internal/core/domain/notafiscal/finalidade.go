package notafiscal

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type Finalidade string

const (
	FinalidadeNormal       Finalidade = "normal"
	FinalidadeComplementar Finalidade = "complementar"
	FinalidadeAjuste       Finalidade = "ajuste"
	FinalidadeDevolucao    Finalidade = "devolucao"
)

var finalidadesPorCodigo = map[string]Finalidade{
	"1": FinalidadeNormal,
	"2": FinalidadeComplementar,
	"3": FinalidadeAjuste,
	"4": FinalidadeDevolucao,
}

func FinalidadeDoCodigo(codigo string) (Finalidade, error) {
	finalidade, ok := finalidadesPorCodigo[strings.TrimSpace(codigo)]
	if !ok {
		return "", domain.ErroValidacao("finalidade da nota (finNFe) inválida")
	}

	return finalidade, nil
}

func FinalidadeDe(valor string) Finalidade {
	return Finalidade(valor)
}

func (f Finalidade) String() string {
	return string(f)
}

type TipoOperacao string

const (
	TipoOperacaoEntrada TipoOperacao = "entrada"
	TipoOperacaoSaida   TipoOperacao = "saida"
)

func TipoOperacaoDoCodigo(codigo string) (TipoOperacao, error) {
	switch strings.TrimSpace(codigo) {
	case "0":
		return TipoOperacaoEntrada, nil
	case "1":
		return TipoOperacaoSaida, nil
	default:
		return "", domain.ErroValidacao("tipo de operação da nota (tpNF) inválido")
	}
}

func TipoOperacaoDe(valor string) TipoOperacao {
	return TipoOperacao(valor)
}

func (t TipoOperacao) String() string {
	return string(t)
}
