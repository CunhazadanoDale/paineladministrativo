package dre

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

const diasDaSemana = 7

type Semana struct {
	inicio time.Time
}

func NovaSemana(dia *time.Time, hoje time.Time) (Semana, error) {
	semanaAtual := InicioDaSemana(hoje)
	if dia == nil {
		return Semana{inicio: semanaAtual.AddDate(0, 0, -diasDaSemana)}, nil
	}

	inicio := InicioDaSemana(*dia)
	if inicio.After(semanaAtual) {
		return Semana{}, domain.ErroValidacao("semana não pode estar no futuro")
	}

	return Semana{inicio: inicio}, nil
}

func SemanaDe(dia time.Time) Semana {
	return Semana{inicio: InicioDaSemana(dia)}
}

func (s Semana) Inicio() time.Time {
	return s.inicio
}

func (s Semana) Fim() time.Time {
	return s.inicio.AddDate(0, 0, diasDaSemana-1)
}
