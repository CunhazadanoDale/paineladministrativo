package importacao

import (
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

const formatoData = "2006-01-02"

func ParaData(valor, campo string) (time.Time, error) {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return time.Time{}, nil
	}
	if data, err := time.Parse(formatoData, valor); err == nil {
		return data, nil
	}

	return time.Time{}, domain.ErroValidacao(campo + " deve estar no formato AAAA-MM-DD")
}

func paraDataOpcional(valor *string, campo string) (*time.Time, error) {
	if valor == nil {
		return nil, nil
	}

	data, err := ParaData(*valor, campo)
	if err != nil || data.IsZero() {
		return nil, err
	}

	return &data, nil
}

func formatarDataOpcional(valor *time.Time) *string {
	if valor == nil {
		return nil
	}

	texto := valor.Format(formatoData)

	return &texto
}
