package empresa

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type RegimeTributario string

const (
	RegimeSimples   RegimeTributario = "simples"
	RegimePresumido RegimeTributario = "presumido"
	RegimeReal      RegimeTributario = "real"
)

func NovoRegimeTributario(valor string) (RegimeTributario, error) {
	regime := RegimeTributario(strings.ToLower(strings.TrimSpace(valor)))
	if err := regime.Validado(); err != nil {
		return "", err
	}

	return regime, nil
}

func RegimeTributarioDe(valor string) RegimeTributario {
	return RegimeTributario(valor)
}

func (r RegimeTributario) String() string {
	return string(r)
}

func (r RegimeTributario) Validado() error {
	switch r {
	case RegimeSimples, RegimePresumido, RegimeReal:
		return nil
	default:
		return domain.ErroValidacao("regime tributário deve ser simples, presumido ou real")
	}
}
