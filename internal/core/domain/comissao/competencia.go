package comissao

import (
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

const (
	formatoCompetencia       = "2006-01"
	primeiroAnoDeCompetencia = 2000
	ultimoAnoDeCompetencia   = 2099
)

type Competencia struct {
	inicio time.Time
}

func NovaCompetencia(texto string) (Competencia, error) {
	data, err := time.Parse(formatoCompetencia, strings.TrimSpace(texto))
	if err != nil {
		return Competencia{}, domain.ErroValidacao("competência deve estar no formato AAAA-MM")
	}
	if data.Year() < primeiroAnoDeCompetencia || data.Year() > ultimoAnoDeCompetencia {
		return Competencia{}, domain.ErroValidacao("competência deve estar entre 2000-01 e 2099-12")
	}

	return CompetenciaDe(data), nil
}

func CompetenciaDe(data time.Time) Competencia {
	ano, mes, _ := data.UTC().Date()

	return Competencia{inicio: time.Date(ano, mes, 1, 0, 0, 0, 0, time.UTC)}
}

func (c Competencia) Inicio() time.Time {
	return c.inicio
}

func (c Competencia) Seguinte() Competencia {
	return Competencia{inicio: c.inicio.AddDate(0, 1, 0)}
}

func (c Competencia) String() string {
	return c.inicio.Format(formatoCompetencia)
}
