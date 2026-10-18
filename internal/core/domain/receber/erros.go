package receber

import "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"

func ErroTituloNaoEncontrado() error {
	return domain.ErroNaoEncontrado("título não encontrado")
}

func ErroEmpresaNaoEncontrada() error {
	return domain.ErroValidacao("empresa do título não encontrada")
}

func ErroEmpresaInativa() error {
	return domain.ErroValidacao("empresa inativa não recebe título")
}

func ErroClienteNaoEncontrado() error {
	return domain.ErroValidacao("cliente do título não encontrado")
}

func ErroClienteInativo() error {
	return domain.ErroValidacao("cliente inativo não recebe título novo")
}

func ErroCategoriaFinanceiraNaoEncontrada() error {
	return domain.ErroValidacao("categoria financeira não encontrada")
}

func ErroCategoriaFinanceiraNaoLancavel() error {
	return domain.ErroValidacao("categoria financeira deve ser uma subcategoria ativa")
}

func ErroCategoriaFinanceiraNaoEhReceita() error {
	return domain.ErroValidacao("categoria financeira do título deve ser de receita")
}

func ErroContaBancariaNaoEncontrada() error {
	return domain.ErroValidacao("conta bancária não encontrada")
}

func ErroContaBancariaDeOutraEmpresa() error {
	return domain.ErroValidacao("conta bancária deve ser da mesma empresa do título")
}

func ErroContaBancariaInativa() error {
	return domain.ErroValidacao("conta bancária inativa não recebe cobrança")
}

func ErroNumeroRepetido() error {
	return domain.ErroConflito("já existe um título com este número e parcela na empresa")
}

func ErroEmpresaObrigatoria() error {
	return domain.ErroValidacao("empresa do título é obrigatória")
}

func ErroClienteObrigatorio() error {
	return domain.ErroValidacao("cliente do título é obrigatório")
}

func ErroOrigemNaoSuportada() error {
	return domain.ErroValidacao("origem ainda não suportada na geração de títulos")
}

func ErroOrigemNaoEncontrada() error {
	return domain.ErroValidacao("origem do título não encontrada")
}

func ErroOrigemCancelada() error {
	return domain.ErroConflito("origem cancelada não gera título")
}

func ErroFomentoNaoEncontrado() error {
	return domain.ErroValidacao("fomento da antecipação não encontrado")
}

func ErroFomentoInativo() error {
	return domain.ErroValidacao("fomento inativo não recebe antecipação")
}

func ErroClienteProibeAntecipacao(nome string) error {
	return domain.ErroValidacao("cliente não permite antecipação: " + nome)
}

func ErroTituloJaAntecipado() error {
	return domain.ErroConflito("título já está numa antecipação vigente")
}

func ErroContaAPagarJaGerada() error {
	return domain.ErroConflito("a conta a pagar desta origem já foi gerada")
}
