package importacao

import (
	"fmt"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

func ErroEmpresaNaoEncontrada() error {
	return domain.ErroValidacao("empresa da importação não encontrada")
}

func ErroEmpresaInativa() error {
	return domain.ErroValidacao("empresa inativa não recebe importação nova")
}

func ErroFornecedorNaoEncontrado() error {
	return domain.ErroValidacao("fornecedor da importação não encontrado")
}

func ErroFornecedorInativo(nome string) error {
	if nome == "" {
		return domain.ErroValidacao("fornecedor inativo não recebe importação nova")
	}

	return domain.ErroValidacao(fmt.Sprintf("fornecedor %q está inativo e não recebe importação nova", nome))
}
