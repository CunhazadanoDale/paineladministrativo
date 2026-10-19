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

func ErroLancamentoNaoEncontrado() error {
	return domain.ErroNaoEncontrado("lançamento do extrato não encontrado")
}

func ErroLancamentoJaConciliado() error {
	return domain.ErroConflito("lançamento do extrato já conciliado; desfaça a conciliação antes")
}

func ErroMovimentoJaConciliado() error {
	return domain.ErroConflito("movimento já conciliado com outro lançamento do extrato")
}

func ErroMovimentoAlterado() error {
	return domain.ErroConflito("um dos movimentos foi estornado ou alterado; recarregue e tente de novo")
}

func ErroConciliacaoNaoEncontrada() error {
	return domain.ErroNaoEncontrado("conciliação não encontrada")
}

func ErroConciliacaoJaDesfeita() error {
	return domain.ErroConflito("conciliação já desfeita")
}

func ErroImportacaoComConciliacao() error {
	return domain.ErroConflito("importação com lançamento que já passou por conciliação não pode ser desfeita")
}
