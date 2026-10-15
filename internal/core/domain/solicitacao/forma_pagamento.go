package solicitacao

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type FormaPagamento string

const (
	FormaPagamentoPix              FormaPagamento = "pix"
	FormaPagamentoCartao           FormaPagamento = "cartao"
	FormaPagamentoBoleto           FormaPagamento = "boleto"
	FormaPagamentoTransferencia    FormaPagamento = "transferencia"
	FormaPagamentoDebitoAutomatico FormaPagamento = "debito_automatico"
)

func NovaFormaPagamento(valor string) (FormaPagamento, error) {
	forma := FormaPagamento(strings.ToLower(strings.TrimSpace(valor)))

	switch forma {
	case FormaPagamentoPix, FormaPagamentoCartao, FormaPagamentoBoleto, FormaPagamentoTransferencia, FormaPagamentoDebitoAutomatico:
		return forma, nil
	default:
		return "", domain.ErroValidacao("forma de pagamento deve ser pix, cartao, boleto, transferencia ou debito_automatico")
	}
}

func (f FormaPagamento) String() string {
	return string(f)
}

func (f FormaPagamento) Validado() error {
	switch f {
	case FormaPagamentoPix, FormaPagamentoCartao, FormaPagamentoBoleto, FormaPagamentoTransferencia, FormaPagamentoDebitoAutomatico:
		return nil
	default:
		return domain.ErroValidacao("forma de pagamento deve ser pix, cartao, boleto, transferencia ou debito_automatico")
	}
}
