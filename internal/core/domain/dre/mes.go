package dre

import (
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

const (
	formatoMes       = "2006-01"
	primeiroAnoDoMes = 2000
	ultimoAnoDoMes   = 2099
)

type Mes struct {
	inicio time.Time
}

func NovoMes(texto, campo string) (Mes, error) {
	data, err := time.Parse(formatoMes, strings.TrimSpace(texto))
	if err != nil {
		return Mes{}, domain.ErroValidacao(campo + " deve estar no formato AAAA-MM")
	}
	if data.Year() < primeiroAnoDoMes || data.Year() > ultimoAnoDoMes {
		return Mes{}, domain.ErroValidacao(campo + " deve estar entre 2000-01 e 2099-12")
	}

	return MesDe(data), nil
}

func MesDe(data time.Time) Mes {
	ano, mes, _ := data.Date()

	return Mes{inicio: time.Date(ano, mes, 1, 0, 0, 0, 0, time.UTC)}
}

func (m Mes) Inicio() time.Time {
	return m.inicio
}

func (m Mes) Fim() time.Time {
	return m.inicio.AddDate(0, 1, -1)
}

func (m Mes) Somar(meses int) Mes {
	return Mes{inicio: m.inicio.AddDate(0, meses, 0)}
}

func (m Mes) Antes(outro Mes) bool {
	return m.inicio.Before(outro.inicio)
}

func (m Mes) String() string {
	return m.inicio.Format(formatoMes)
}

func mesesEntre(de, ate Mes) int {
	return (ate.inicio.Year()-de.inicio.Year())*12 + int(ate.inicio.Month()) - int(de.inicio.Month()) + 1
}
