package solicitacao

import (
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
)

const (
	primeiroDiaDoMes  = 1
	ultimoDiaPossivel = 31
)

type DiaVencimento struct {
	dia int
}

func NovoDiaVencimento(dia int) (DiaVencimento, error) {
	if dia < primeiroDiaDoMes || dia > ultimoDiaPossivel {
		return DiaVencimento{}, domain.ErroValidacao("dia de vencimento deve estar entre 1 e 31")
	}

	return DiaVencimento{dia: dia}, nil
}

func DiaVencimentoDe(dia int) DiaVencimento {
	return DiaVencimento{dia: dia}
}

func (d DiaVencimento) Dia() int {
	return d.dia
}
