package divida

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type TipoCredor string

const (
	CredorBanco      TipoCredor = "banco"
	CredorInvestidor TipoCredor = "investidor"
	CredorFisco      TipoCredor = "fisco"
	CredorFornecedor TipoCredor = "fornecedor"
)

func NovoTipoCredor(valor string) (TipoCredor, error) {
	tipo := TipoCredor(strings.ToLower(strings.TrimSpace(valor)))

	switch tipo {
	case CredorBanco, CredorInvestidor, CredorFisco, CredorFornecedor:
		return tipo, nil
	default:
		return "", domain.ErroValidacao("tipo de credor deve ser banco, investidor, fisco ou fornecedor")
	}
}

func TipoCredorDe(valor string) TipoCredor {
	return TipoCredor(valor)
}

func (t TipoCredor) String() string {
	return string(t)
}
