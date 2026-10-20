package dre_test

import (
	"testing"

	domaindre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/dre"
	"github.com/google/uuid"
)

func TestTicketMedioArredondaEZeraSemNotas(t *testing.T) {
	if ticket := (domaindre.TotalFaturamento{Quantidade: 3, ValorCentavos: 1000}).TicketMedioCentavos(); ticket != 333 {
		t.Errorf("ticket = %d, esperado 333", ticket)
	}
	if ticket := (domaindre.TotalFaturamento{Quantidade: 2, ValorCentavos: 1001}).TicketMedioCentavos(); ticket != 501 {
		t.Errorf("ticket = %d, esperado 501", ticket)
	}
	if ticket := (domaindre.TotalFaturamento{}).TicketMedioCentavos(); ticket != 0 {
		t.Errorf("ticket sem notas = %d, esperado 0", ticket)
	}
}

func TestResumoFaturamentoSomaPorMesEmpresaEConsolidado(t *testing.T) {
	agosto, setembro := mes(t, "2026-08"), mes(t, "2026-09")
	periodo := domaindre.PeriodoMensalDe(agosto, setembro)
	faturamentos := []domaindre.Faturamento{
		{Mes: agosto, EmpresaID: empresaB, Quantidade: 2, ValorCentavos: 2000},
		{Mes: setembro, EmpresaID: empresaA, Quantidade: 1, ValorCentavos: 500},
		{Mes: setembro, EmpresaID: empresaB, Quantidade: 1, ValorCentavos: 100},
		{Mes: mes(t, "2026-10"), EmpresaID: empresaA, Quantidade: 9, ValorCentavos: 9000},
	}

	resumo := domaindre.NovoResumoFaturamento(periodo, nil, faturamentos)

	if len(resumo.Meses) != 2 || len(resumo.Meses[0].Empresas) != 2 {
		t.Fatalf("resumo com %d meses e %d empresas", len(resumo.Meses), len(resumo.Meses[0].Empresas))
	}
	if resumo.Meses[0].Empresas[0].EmpresaID != empresaA || resumo.Meses[0].Empresas[0].Total.Quantidade != 0 {
		t.Errorf("empresa A em agosto deveria aparecer zerada")
	}
	if resumo.Meses[1].Consolidado != (domaindre.TotalFaturamento{Quantidade: 2, ValorCentavos: 600}) {
		t.Errorf("setembro = %+v, esperado 2 notas e 600", resumo.Meses[1].Consolidado)
	}
	if resumo.Empresas[1].Total != (domaindre.TotalFaturamento{Quantidade: 3, ValorCentavos: 2100}) {
		t.Errorf("empresa B = %+v, esperado 3 notas e 2100", resumo.Empresas[1].Total)
	}
	if resumo.Consolidado != (domaindre.TotalFaturamento{Quantidade: 4, ValorCentavos: 2600}) {
		t.Errorf("consolidado = %+v, esperado 4 notas e 2600 sem o mês fora do período", resumo.Consolidado)
	}
}

func TestResumoFaturamentoIncluiEmpresaExigidaSemNotas(t *testing.T) {
	empresa := uuid.New()

	resumo := domaindre.NovoResumoFaturamento(periodoDeUmMes(t, "2026-09"), []uuid.UUID{empresa}, nil)

	if len(resumo.Empresas) != 1 || resumo.Empresas[0].EmpresaID != empresa || len(resumo.Meses[0].Empresas) != 1 {
		t.Errorf("empresa filtrada deveria aparecer zerada: %+v", resumo)
	}
}
