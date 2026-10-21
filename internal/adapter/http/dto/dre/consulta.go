package dre

import (
	"strings"

	domaindre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/dre"
)

func MesDeConsulta(texto, campo string) (*domaindre.Mes, error) {
	if strings.TrimSpace(texto) == "" {
		return nil, nil
	}

	mes, err := domaindre.NovoMes(texto, campo)
	if err != nil {
		return nil, err
	}

	return &mes, nil
}
