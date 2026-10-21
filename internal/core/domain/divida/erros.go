package divida

import (
	"fmt"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

func ErroDividaNaoEncontrada() error {
	return domain.ErroNaoEncontrado("dívida não encontrada")
}

func ErroVersaoDivergente() error {
	return domain.ErroConflito("a dívida foi alterada por outra pessoa; recarregue e tente de novo")
}

func ErroEmpresaNaoEncontrada() error {
	return domain.ErroValidacao("empresa da dívida não encontrada")
}

func ErroEmpresaInativa() error {
	return domain.ErroValidacao("empresa inativa não recebe dívida nova")
}

func ErroFornecedorNaoEncontrado() error {
	return domain.ErroValidacao("credor da dívida não encontrado")
}

func ErroFornecedorInativo(nome string) error {
	if nome == "" {
		return domain.ErroValidacao("credor inativo não recebe dívida nova")
	}

	return domain.ErroValidacao(fmt.Sprintf("credor %q está inativo e não recebe dívida nova", nome))
}

func ErroContaAPagarAlterada() error {
	return domain.ErroConflito("uma conta a pagar da dívida mudou no contas a pagar; recarregue a dívida e tente de novo")
}
