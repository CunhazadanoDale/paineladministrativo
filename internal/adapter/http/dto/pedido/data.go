package pedido

import (
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

const formatoData = "2006-01-02"

func ParaData(valor, campo string) (time.Time, error) {
	valor = strings.TrimSpace(valor)
	if data, err := time.Parse(formatoData, valor); err == nil {
		return data, nil
	}
	if data, err := time.Parse(time.RFC3339, valor); err == nil {
		return data, nil
	}

	return time.Time{}, domain.ErroValidacao(campo + " deve estar no formato AAAA-MM-DD")
}

func paraDataOpcional(valor *string, campo string) (*time.Time, error) {
	if valor == nil || strings.TrimSpace(*valor) == "" {
		return nil, nil
	}

	data, err := ParaData(*valor, campo)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func paraDataOuZero(valor *string, campo string) (time.Time, error) {
	data, err := paraDataOpcional(valor, campo)
	if err != nil || data == nil {
		return time.Time{}, err
	}

	return *data, nil
}
