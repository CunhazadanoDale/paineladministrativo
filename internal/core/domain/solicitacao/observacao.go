package solicitacao

import (
	"strings"
	"unicode/utf8"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

const tamanhoMaximoObservacao = 1000

type Observacao struct {
	texto string
}

func NovaObservacao(texto string) (Observacao, error) {
	texto = strings.TrimSpace(texto)

	if texto == "" {
		return Observacao{}, domain.ErroValidacao("observação é obrigatória")
	}
	if utf8.RuneCountInString(texto) > tamanhoMaximoObservacao {
		return Observacao{}, domain.ErroValidacao("observação deve ter no máximo 1000 caracteres")
	}

	return Observacao{texto: texto}, nil
}

func ObservacaoDe(texto string) Observacao {
	return Observacao{texto: texto}
}

func (o Observacao) Texto() string {
	return o.texto
}
