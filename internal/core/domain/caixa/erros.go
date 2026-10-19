package caixa

import "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"

func ErroContaNaoEncontrada() error {
	return domain.ErroNaoEncontrado("conta bancária não encontrada")
}

func ErroContaInativa() error {
	return domain.ErroValidacao("conta bancária inativa não recebe saldo inicial nem extrato")
}

func ErroEmpresaInativa() error {
	return domain.ErroValidacao("empresa inativa não recebe saldo inicial nem extrato")
}

func ErroAberturaNaoEncontrada() error {
	return domain.ErroNaoEncontrado("conta sem saldo inicial")
}

func ErroContaSemAbertura() error {
	return domain.ErroValidacao("conta sem saldo inicial; informe a abertura antes de importar o extrato")
}

func ErroAberturaJaDefinida() error {
	return domain.ErroConflito("a conta já tem saldo inicial; envie a versão para alterar")
}

func ErroAberturaAlterada() error {
	return domain.ErroConflito("o saldo inicial da conta foi alterado por outra pessoa; recarregue e tente de novo")
}

func ErroArquivoJaImportado() error {
	return domain.ErroConflito("arquivo já importado em outro extrato")
}

func ErroImportacaoNaoEncontrada() error {
	return domain.ErroNaoEncontrado("importação de extrato não encontrada")
}

func ErroImportacaoNaoEhAUltima() error {
	return domain.ErroConflito("só a importação mais recente da conta pode ser desfeita")
}
