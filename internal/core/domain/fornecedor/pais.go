package fornecedor

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

const PaisBrasil Pais = "BR"

type Pais string

func NovoPais(valor string) (Pais, error) {
	pais := Pais(strings.ToUpper(strings.TrimSpace(valor)))
	if pais == "" {
		return PaisBrasil, nil
	}
	if err := pais.Validado(); err != nil {
		return "", err
	}

	return pais, nil
}

func PaisDe(valor string) Pais {
	return Pais(valor)
}

func (p Pais) String() string {
	return string(p)
}

func (p Pais) Brasil() bool {
	return p == PaisBrasil
}

func (p Pais) Validado() error {
	if len(p) != 2 {
		return domain.ErroValidacao("país deve ser o código ISO de duas letras, como BR, CN ou US")
	}
	for _, letra := range p {
		if letra < 'A' || letra > 'Z' {
			return domain.ErroValidacao("país deve ser o código ISO de duas letras, como BR, CN ou US")
		}
	}

	return nil
}
