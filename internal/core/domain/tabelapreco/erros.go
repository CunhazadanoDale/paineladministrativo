package tabelapreco

import "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"

func ErroProdutoNaoEncontrado() error {
	return domain.ErroValidacao("produto do preço não encontrado")
}

func ErroProdutoInativo() error {
	return domain.ErroValidacao("produto inativo não recebe preço novo")
}

func ErroVigenciaRepetida() error {
	return domain.ErroConflito("já existe preço deste produto e modelo com início nesta data")
}

func ErroItemNaoEncontrado() error {
	return domain.ErroNaoEncontrado("preço da tabela não encontrado")
}

func ErroVigenciaJaComecou() error {
	return domain.ErroConflito("só é possível excluir preço cuja vigência ainda não começou")
}
