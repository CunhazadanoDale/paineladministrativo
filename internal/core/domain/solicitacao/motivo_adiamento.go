package solicitacao

import (
	"strings"
	"unicode/utf8"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

type MotivoAdiamento struct {
	texto string
}

func NovoMotivoAdiamento(texto string) (MotivoAdiamento, error) {
	texto = strings.TrimSpace(texto)

	if texto == "" {
		return MotivoAdiamento{}, domain.ErroValidacao("motivo do adiamento é obrigatório")
	}
	if utf8.RuneCountInString(texto) > tamanhoMaximoMotivo {
		return MotivoAdiamento{}, domain.ErroValidacao("motivo do adiamento deve ter no máximo 500 caracteres")
	}

	return MotivoAdiamento{texto: texto}, nil
}

func (m MotivoAdiamento) Texto() string {
	return m.texto
}
