package solicitacao

import (
	"strings"
	"unicode/utf8"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

const tamanhoMaximoMotivo = 500

type MotivoRejeicao struct {
	texto string
}

func NovoMotivoRejeicao(texto string) (MotivoRejeicao, error) {
	texto = strings.TrimSpace(texto)

	if texto == "" {
		return MotivoRejeicao{}, domain.ErroValidacao("motivo da rejeição é obrigatório")
	}
	if utf8.RuneCountInString(texto) > tamanhoMaximoMotivo {
		return MotivoRejeicao{}, domain.ErroValidacao("motivo da rejeição deve ter no máximo 500 caracteres")
	}

	return MotivoRejeicao{texto: texto}, nil
}

func MotivoRejeicaoDe(texto string) MotivoRejeicao {
	return MotivoRejeicao{texto: texto}
}

func (m MotivoRejeicao) Texto() string {
	return m.texto
}
