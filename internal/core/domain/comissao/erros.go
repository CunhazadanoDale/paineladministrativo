package comissao

import "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"

func ErroPedidoNaoEncontrado() error {
	return domain.ErroNaoEncontrado("pedido não encontrado")
}

func ErroPedidoSemRepresentante() error {
	return domain.ErroValidacao("pedido sem representante não tem comissão")
}

func ErroPedidoCancelado() error {
	return domain.ErroConflito("pedido cancelado não muda a comissão")
}

func ErroPercentualTravado() error {
	return domain.ErroConflito("a comissão do pedido não muda depois da primeira saída de entrega")
}

func ErroVersaoDivergente() error {
	return domain.ErroConflito("a comissão do pedido foi alterada por outra pessoa; recarregue e tente de novo")
}

func ErroRepresentanteNaoDefineComissao() error {
	return domain.ErroPermissao("representante não define a própria comissão")
}

func ErroRepresentanteNaoEncontrado() error {
	return domain.ErroValidacao("representante do pedido não encontrado")
}

func ErroPercentualMudouNaApuracao() error {
	return domain.ErroConflito("a comissão do pedido mudou durante a apuração; apure de novo")
}

func ErroRepresentanteMudouNaApuracao() error {
	return domain.ErroConflito("o representante do pedido mudou durante a apuração; apure de novo")
}

func ErroRepresentanteDoAjusteNaoEncontrado() error {
	return domain.ErroValidacao("representante do ajuste não encontrado")
}

func ErroPedidoDoAjusteNaoEncontrado() error {
	return domain.ErroValidacao("pedido do ajuste não encontrado")
}

func ErroPedidoDeOutraEmpresaOuRepresentante() error {
	return domain.ErroValidacao("pedido do ajuste deve ser da mesma empresa e do mesmo representante")
}
