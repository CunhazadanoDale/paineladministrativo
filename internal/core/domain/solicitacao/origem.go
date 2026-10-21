package solicitacao

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type Origem string

const (
	OrigemManual            Origem = "manual"
	OrigemRecorrencia       Origem = "recorrencia"
	OrigemRecompra          Origem = "recompra"
	OrigemRepasseFomento    Origem = "repasse_fomento"
	OrigemComissao          Origem = "comissao"
	OrigemImportacaoParcela Origem = "importacao_parcela"
	OrigemImportacaoCusto   Origem = "importacao_custo"
	OrigemDividaAmortizacao Origem = "divida_amortizacao"
	OrigemDividaJuros       Origem = "divida_juros"
)

func NovaOrigem(valor string) (Origem, error) {
	origem := Origem(strings.ToLower(strings.TrimSpace(valor)))

	switch origem {
	case OrigemManual, OrigemRecorrencia, OrigemRecompra, OrigemRepasseFomento, OrigemComissao,
		OrigemImportacaoParcela, OrigemImportacaoCusto, OrigemDividaAmortizacao, OrigemDividaJuros:
		return origem, nil
	default:
		return "", domain.ErroValidacao("origem deve ser manual, recorrencia, recompra, repasse_fomento, comissao, importacao_parcela, importacao_custo, divida_amortizacao ou divida_juros")
	}
}

func OrigemDe(valor string) Origem {
	return Origem(valor)
}

func (o Origem) String() string {
	return string(o)
}

func (o Origem) Sistemica() bool {
	return o == OrigemRecompra || o == OrigemRepasseFomento || o == OrigemComissao ||
		o == OrigemImportacaoParcela || o == OrigemImportacaoCusto || o == OrigemDividaAmortizacao || o == OrigemDividaJuros
}
