package estoque

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type TipoMovimento string

const (
	TipoMovimentoEntrada       TipoMovimento = "entrada"
	TipoMovimentoSaida         TipoMovimento = "saida"
	TipoMovimentoAjuste        TipoMovimento = "ajuste"
	TipoMovimentoPerda         TipoMovimento = "perda"
	TipoMovimentoDevolucao     TipoMovimento = "devolucao"
	TipoMovimentoImplantacao   TipoMovimento = "implantacao"
	TipoMovimentoTransferencia TipoMovimento = "transferencia"
)

var tiposMovimentoValidos = []TipoMovimento{
	TipoMovimentoEntrada,
	TipoMovimentoSaida,
	TipoMovimentoAjuste,
	TipoMovimentoPerda,
	TipoMovimentoDevolucao,
	TipoMovimentoImplantacao,
	TipoMovimentoTransferencia,
}

func NovoTipoMovimento(valor string) (TipoMovimento, error) {
	tipo := TipoMovimento(strings.ToLower(strings.TrimSpace(valor)))

	if err := tipo.Validado(); err != nil {
		return "", err
	}

	return tipo, nil
}

func TipoMovimentoDe(valor string) TipoMovimento {
	return TipoMovimento(valor)
}

func (t TipoMovimento) String() string {
	return string(t)
}

func (t TipoMovimento) Validado() error {
	for _, valido := range tiposMovimentoValidos {
		if t == valido {
			return nil
		}
	}

	return domain.ErroValidacao("tipo de movimento deve ser entrada, saida, ajuste, perda, devolucao, implantacao ou transferencia")
}

func (t TipoMovimento) Efeito(quantidade Quantidade) int {
	switch t {
	case TipoMovimentoImplantacao, TipoMovimentoTransferencia:
		return 0
	default:
		return t.EfeitoNoLocal(quantidade)
	}
}

func (t TipoMovimento) EfeitoNoLocal(quantidade Quantidade) int {
	switch t {
	case TipoMovimentoEntrada, TipoMovimentoDevolucao, TipoMovimentoImplantacao:
		return quantidade.Absoluta()
	case TipoMovimentoAjuste:
		return quantidade.Valor()
	default:
		return -quantidade.Absoluta()
	}
}
