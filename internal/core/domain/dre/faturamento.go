package dre

import (
	"math/big"

	"github.com/google/uuid"
)

type Faturamento struct {
	Mes           Mes
	EmpresaID     uuid.UUID
	Quantidade    int
	ValorCentavos int64
}

type TotalFaturamento struct {
	Quantidade    int
	ValorCentavos int64
}

func (t TotalFaturamento) somar(faturamento Faturamento) TotalFaturamento {
	return TotalFaturamento{
		Quantidade:    t.Quantidade + faturamento.Quantidade,
		ValorCentavos: t.ValorCentavos + faturamento.ValorCentavos,
	}
}

func (t TotalFaturamento) TicketMedioCentavos() int64 {
	if t.Quantidade <= 0 {
		return 0
	}

	ticket := dividirArredondado(big.NewInt(t.ValorCentavos), big.NewInt(int64(t.Quantidade)))

	return *ticket
}

type FaturamentoEmpresa struct {
	EmpresaID uuid.UUID
	Total     TotalFaturamento
}

type MesFaturamento struct {
	Mes         Mes
	Empresas    []FaturamentoEmpresa
	Consolidado TotalFaturamento
}

type ResumoFaturamento struct {
	De          Mes
	Ate         Mes
	Meses       []MesFaturamento
	Empresas    []FaturamentoEmpresa
	Consolidado TotalFaturamento
}

func NovoResumoFaturamento(periodo PeriodoMensal, empresasExigidas []uuid.UUID, faturamentos []Faturamento) ResumoFaturamento {
	meses := periodo.Meses()
	posicaoDoMes := make(map[Mes]int, len(meses))
	for i, mes := range meses {
		posicaoDoMes[mes] = i
	}

	presentes := append([]uuid.UUID{}, empresasExigidas...)
	for _, faturamento := range faturamentos {
		if _, ok := posicaoDoMes[faturamento.Mes]; ok {
			presentes = append(presentes, faturamento.EmpresaID)
		}
	}
	empresas := empresasOrdenadas(presentes)

	porMes := make([]map[uuid.UUID]TotalFaturamento, len(meses))
	for i := range porMes {
		porMes[i] = make(map[uuid.UUID]TotalFaturamento, len(empresas))
	}
	porEmpresa := make(map[uuid.UUID]TotalFaturamento, len(empresas))

	resumo := ResumoFaturamento{De: periodo.De(), Ate: periodo.Ate(), Meses: make([]MesFaturamento, 0, len(meses))}
	for _, faturamento := range faturamentos {
		posicao, ok := posicaoDoMes[faturamento.Mes]
		if !ok {
			continue
		}
		porMes[posicao][faturamento.EmpresaID] = porMes[posicao][faturamento.EmpresaID].somar(faturamento)
		porEmpresa[faturamento.EmpresaID] = porEmpresa[faturamento.EmpresaID].somar(faturamento)
		resumo.Consolidado = resumo.Consolidado.somar(faturamento)
	}

	for i, mes := range meses {
		linha := MesFaturamento{Mes: mes, Empresas: make([]FaturamentoEmpresa, 0, len(empresas))}
		for _, empresaID := range empresas {
			total := porMes[i][empresaID]
			linha.Empresas = append(linha.Empresas, FaturamentoEmpresa{EmpresaID: empresaID, Total: total})
			linha.Consolidado.Quantidade += total.Quantidade
			linha.Consolidado.ValorCentavos += total.ValorCentavos
		}
		resumo.Meses = append(resumo.Meses, linha)
	}

	resumo.Empresas = make([]FaturamentoEmpresa, 0, len(empresas))
	for _, empresaID := range empresas {
		resumo.Empresas = append(resumo.Empresas, FaturamentoEmpresa{EmpresaID: empresaID, Total: porEmpresa[empresaID]})
	}

	return resumo
}
