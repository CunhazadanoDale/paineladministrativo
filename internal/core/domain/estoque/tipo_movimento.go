package estoque

import (
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type TipoMovimento string

const (
	TipoMovimentoEntrada   TipoMovimento = "entrada"
	TipoMovimentoSaida     TipoMovimento = "saida"
	TipoMovimentoAjuste    TipoMovimento = "ajuste"
	TipoMovimentoPerda     TipoMovimento = "perda"
	TipoMovimentoDevolucao TipoMovimento = "devolucao"
)

var tiposMovimentoValidos = []TipoMovimento{
	TipoMovimentoEntrada,
	TipoMovimentoSaida,
	TipoMovimentoAjuste,
	TipoMovimentoPerda,
	TipoMovimentoDevolucao,
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

	return domain.ErroValidacao("tipo de movimento deve ser entrada, saida, ajuste, perda ou devolucao")
}

func (t TipoMovimento) Efeito(quantidade Quantidade) int {
	switch t {
	case TipoMovimentoEntrada, TipoMovimentoDevolucao:
		return quantidade.Absoluta()
	case TipoMovimentoAjuste:
		return quantidade.Valor()
	default:
		return -quantidade.Absoluta()
	}
}
