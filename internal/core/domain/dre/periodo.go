package dre

import (
	"fmt"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

const (
	mesesPadraoDoPeriodo      = 12
	mesesMaximosDaDRE         = 24
	mesesMaximosDoFaturamento = 60
)

type PeriodoMensal struct {
	de  Mes
	ate Mes
}

func NovoPeriodoDaDRE(de, ate *Mes, hoje time.Time) (PeriodoMensal, error) {
	return novoPeriodoMensal(de, ate, hoje, mesesMaximosDaDRE)
}

func NovoPeriodoDoFaturamento(de, ate *Mes, hoje time.Time) (PeriodoMensal, error) {
	return novoPeriodoMensal(de, ate, hoje, mesesMaximosDoFaturamento)
}

func PeriodoMensalDe(de, ate Mes) PeriodoMensal {
	return PeriodoMensal{de: de, ate: ate}
}

func novoPeriodoMensal(de, ate *Mes, hoje time.Time, maximo int) (PeriodoMensal, error) {
	fim := MesDe(hoje)
	if ate != nil {
		fim = *ate
	}
	inicio := fim.Somar(1 - mesesPadraoDoPeriodo)
	if de != nil {
		inicio = *de
	}

	if fim.Antes(inicio) {
		return PeriodoMensal{}, domain.ErroValidacao("mês final não pode ser anterior ao mês inicial")
	}
	if mesesEntre(inicio, fim) > maximo {
		return PeriodoMensal{}, domain.ErroValidacao(fmt.Sprintf("período deve ter no máximo %d meses", maximo))
	}

	return PeriodoMensal{de: inicio, ate: fim}, nil
}

func (p PeriodoMensal) De() Mes {
	return p.de
}

func (p PeriodoMensal) Ate() Mes {
	return p.ate
}

func (p PeriodoMensal) Meses() []Mes {
	quantidade := mesesEntre(p.de, p.ate)
	meses := make([]Mes, 0, quantidade)
	for i := 0; i < quantidade; i++ {
		meses = append(meses, p.de.Somar(i))
	}

	return meses
}
