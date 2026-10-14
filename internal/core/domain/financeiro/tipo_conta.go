package financeiro

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type TipoConta string

const (
	TipoContaCorrente  TipoConta = "corrente"
	TipoContaPoupanca  TipoConta = "poupanca"
	TipoContaPagamento TipoConta = "pagamento"
	TipoContaAplicacao TipoConta = "aplicacao"
)

func NovoTipoConta(valor string) (TipoConta, error) {
	tipo := TipoConta(strings.ToLower(strings.TrimSpace(valor)))
	if err := tipo.Validado(); err != nil {
		return "", err
	}

	return tipo, nil
}

func TipoContaDe(valor string) TipoConta {
	return TipoConta(valor)
}

func (t TipoConta) String() string {
	return string(t)
}

func (t TipoConta) Validado() error {
	switch t {
	case TipoContaCorrente, TipoContaPoupanca, TipoContaPagamento, TipoContaAplicacao:
		return nil
	default:
		return domain.ErroValidacao("tipo da conta deve ser corrente, poupanca, pagamento ou aplicacao")
	}
}
