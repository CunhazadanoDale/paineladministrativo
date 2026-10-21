package divida

import (
	"bytes"
	"sort"
	"time"

	"github.com/google/uuid"
)

type ParcelaEmAberto struct {
	DividaID            uuid.UUID
	EmpresaID           uuid.UUID
	CredorTipo          TipoCredor
	Vencimento          time.Time
	AmortizacaoCentavos int64
	JurosCentavos       int64
}

type Totais struct {
	AmortizacaoCentavos int64
	JurosCentavos       int64
	Parcelas            int
}

func (t *Totais) somar(parcela ParcelaEmAberto) {
	t.AmortizacaoCentavos += parcela.AmortizacaoCentavos
	t.JurosCentavos += parcela.JurosCentavos
	t.Parcelas++
}

type SaldoDoCredor struct {
	CredorTipo           TipoCredor
	SaldoDevedorCentavos int64
	Dividas              int
}

type SaldoDaEmpresa struct {
	EmpresaID            uuid.UUID
	SaldoDevedorCentavos int64
	Dividas              int
}

type Resumo struct {
	Hoje                 time.Time
	SaldoDevedorCentavos int64
	Dividas              int
	PorCredor            []SaldoDoCredor
	PorEmpresa           []SaldoDaEmpresa
	Vencido              Totais
	AVencer              Totais
	AVencer30            Totais
	AVencer90            Totais
	AVencer365           Totais
}

var ordemDosCredores = []TipoCredor{CredorBanco, CredorInvestidor, CredorFisco, CredorFornecedor}

func NovoResumo(parcelas []ParcelaEmAberto, hoje time.Time) Resumo {
	resumo := Resumo{Hoje: hoje}
	porCredor := make(map[TipoCredor]*SaldoDoCredor, len(ordemDosCredores))
	for _, tipo := range ordemDosCredores {
		porCredor[tipo] = &SaldoDoCredor{CredorTipo: tipo}
	}
	porEmpresa := map[uuid.UUID]*SaldoDaEmpresa{}
	contadas := map[uuid.UUID]bool{}

	for _, parcela := range parcelas {
		if parcela.AmortizacaoCentavos == 0 && parcela.JurosCentavos == 0 {
			continue
		}

		credor, ok := porCredor[parcela.CredorTipo]
		if !ok {
			credor = &SaldoDoCredor{CredorTipo: parcela.CredorTipo}
			porCredor[parcela.CredorTipo] = credor
		}
		empresa, ok := porEmpresa[parcela.EmpresaID]
		if !ok {
			empresa = &SaldoDaEmpresa{EmpresaID: parcela.EmpresaID}
			porEmpresa[parcela.EmpresaID] = empresa
		}
		if !contadas[parcela.DividaID] {
			contadas[parcela.DividaID] = true
			resumo.Dividas++
			credor.Dividas++
			empresa.Dividas++
		}

		resumo.SaldoDevedorCentavos += parcela.AmortizacaoCentavos
		credor.SaldoDevedorCentavos += parcela.AmortizacaoCentavos
		empresa.SaldoDevedorCentavos += parcela.AmortizacaoCentavos

		vencimento := DataDe(parcela.Vencimento)
		if vencimento.Before(hoje) {
			resumo.Vencido.somar(parcela)
			continue
		}
		resumo.AVencer.somar(parcela)
		for dias, totais := range map[int]*Totais{30: &resumo.AVencer30, 90: &resumo.AVencer90, 365: &resumo.AVencer365} {
			if !vencimento.After(hoje.AddDate(0, 0, dias)) {
				totais.somar(parcela)
			}
		}
	}

	for _, tipo := range ordemDosCredores {
		resumo.PorCredor = append(resumo.PorCredor, *porCredor[tipo])
		delete(porCredor, tipo)
	}
	extras := make([]SaldoDoCredor, 0, len(porCredor))
	for _, credor := range porCredor {
		extras = append(extras, *credor)
	}
	sort.Slice(extras, func(i, j int) bool { return extras[i].CredorTipo < extras[j].CredorTipo })
	resumo.PorCredor = append(resumo.PorCredor, extras...)

	resumo.PorEmpresa = make([]SaldoDaEmpresa, 0, len(porEmpresa))
	for _, empresa := range porEmpresa {
		resumo.PorEmpresa = append(resumo.PorEmpresa, *empresa)
	}
	sort.Slice(resumo.PorEmpresa, func(i, j int) bool {
		a, b := resumo.PorEmpresa[i], resumo.PorEmpresa[j]
		if a.SaldoDevedorCentavos != b.SaldoDevedorCentavos {
			return a.SaldoDevedorCentavos > b.SaldoDevedorCentavos
		}

		return bytes.Compare(a.EmpresaID[:], b.EmpresaID[:]) < 0
	})

	return resumo
}

func (d *Divida) VencidoCentavos(hoje time.Time) int64 {
	if d.Status != StatusAtiva {
		return 0
	}

	var vencido int64
	for _, parcela := range d.Parcelas {
		if parcela.Vencimento.Before(hoje) {
			vencido += parcela.AmortizacaoEmAberto() + parcela.JurosEmAberto()
		}
	}

	return vencido
}
