package financeiro

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type Banco string

const (
	BancoDoBrasil Banco = "001"
	Santander     Banco = "033"
	Inter         Banco = "077"
	Bradesco      Banco = "237"
	Itau          Banco = "341"
	Cora          Banco = "403"
	Sicredi       Banco = "748"
)

var catalogoDeBancos = []struct {
	banco Banco
	nome  string
}{
	{BancoDoBrasil, "Banco do Brasil"},
	{Santander, "Santander"},
	{Inter, "Inter"},
	{Bradesco, "Bradesco"},
	{Itau, "Itaú"},
	{Cora, "Cora"},
	{Sicredi, "Sicredi"},
}

func NovoBanco(valor string) (Banco, error) {
	banco := Banco(strings.TrimSpace(valor))
	if err := banco.Validado(); err != nil {
		return "", err
	}

	return banco, nil
}

func BancoDe(valor string) Banco {
	return Banco(valor)
}

func Bancos() []Banco {
	bancos := make([]Banco, 0, len(catalogoDeBancos))
	for _, item := range catalogoDeBancos {
		bancos = append(bancos, item.banco)
	}

	return bancos
}

func (b Banco) String() string {
	return string(b)
}

func (b Banco) Nome() string {
	for _, item := range catalogoDeBancos {
		if item.banco == b {
			return item.nome
		}
	}

	return ""
}

func (b Banco) Validado() error {
	if b.Nome() == "" {
		return domain.ErroValidacao("banco não suportado")
	}

	return nil
}
