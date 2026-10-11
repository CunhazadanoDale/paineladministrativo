package empresa

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type UF string

var ufsValidas = map[UF]struct{}{
	"AC": {}, "AL": {}, "AP": {}, "AM": {}, "BA": {}, "CE": {}, "DF": {}, "ES": {}, "GO": {},
	"MA": {}, "MT": {}, "MS": {}, "MG": {}, "PA": {}, "PB": {}, "PR": {}, "PE": {}, "PI": {},
	"RJ": {}, "RN": {}, "RS": {}, "RO": {}, "RR": {}, "SC": {}, "SP": {}, "SE": {}, "TO": {},
}

func NovaUF(valor string) (UF, error) {
	uf := UF(strings.ToUpper(strings.TrimSpace(valor)))
	if err := uf.Validado(); err != nil {
		return "", err
	}

	return uf, nil
}

func UFDe(valor string) UF {
	return UF(valor)
}

func (u UF) String() string {
	return string(u)
}

func (u UF) Validado() error {
	if _, ok := ufsValidas[u]; !ok {
		return domain.ErroValidacao("uf deve ser a sigla de um estado brasileiro")
	}

	return nil
}
