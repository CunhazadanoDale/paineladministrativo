package caixa

import (
	"time"

	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	"github.com/google/uuid"
)

type SaldoConta struct {
	ContaBancariaID                  uuid.UUID
	EmpresaID                        uuid.UUID
	Apelido                          string
	Banco                            domainfinanceiro.Banco
	Ativa                            bool
	DataCorte                        *time.Time
	SaldoInicialCentavos             *int64
	MovimentoCentavos                int64
	UltimoLancamentoEm               *time.Time
	SaldoBancoCentavos               *int64
	SaldoBancoEm                     *time.Time
	MovimentoAteSaldoDoBancoCentavos int64
	LancamentosPendentes             int
}

func (s *SaldoConta) TemAbertura() bool {
	return s.DataCorte != nil && s.SaldoInicialCentavos != nil
}

func (s *SaldoConta) SaldoAtualCentavos() *int64 {
	if !s.TemAbertura() {
		return nil
	}

	saldo := *s.SaldoInicialCentavos + s.MovimentoCentavos

	return &saldo
}

func (s *SaldoConta) DiferencaDoBancoCentavos() *int64 {
	if !s.TemAbertura() || s.SaldoBancoCentavos == nil || s.SaldoBancoEm == nil || s.SaldoBancoEm.Before(*s.DataCorte) {
		return nil
	}

	diferenca := *s.SaldoBancoCentavos - (*s.SaldoInicialCentavos + s.MovimentoAteSaldoDoBancoCentavos)

	return &diferenca
}
