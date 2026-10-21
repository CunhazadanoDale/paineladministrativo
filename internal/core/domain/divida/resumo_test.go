package divida_test

import (
	"testing"

	domaindivida "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/divida"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

func TestResumoAgrupaSaldoEVencimentos(t *testing.T) {
	hoje := data(2026, 10, 20)
	mia, red := uuid.New(), uuid.New()
	banco, fisco, outroBanco := uuid.New(), uuid.New(), uuid.New()

	resumo := domaindivida.NovoResumo([]domaindivida.ParcelaEmAberto{
		{DividaID: banco, EmpresaID: mia, CredorTipo: domaindivida.CredorBanco, Vencimento: data(2026, 10, 10), AmortizacaoCentavos: 1000, JurosCentavos: 100},
		{DividaID: banco, EmpresaID: mia, CredorTipo: domaindivida.CredorBanco, Vencimento: data(2026, 10, 20), AmortizacaoCentavos: 1000, JurosCentavos: 80},
		{DividaID: banco, EmpresaID: mia, CredorTipo: domaindivida.CredorBanco, Vencimento: data(2026, 12, 20), AmortizacaoCentavos: 1000, JurosCentavos: 60},
		{DividaID: fisco, EmpresaID: red, CredorTipo: domaindivida.CredorFisco, Vencimento: data(2027, 6, 1), AmortizacaoCentavos: 5000},
		{DividaID: fisco, EmpresaID: red, CredorTipo: domaindivida.CredorFisco, Vencimento: data(2028, 1, 1), AmortizacaoCentavos: 5000},
		{DividaID: outroBanco, EmpresaID: red, CredorTipo: domaindivida.CredorBanco, Vencimento: data(2026, 10, 1), JurosCentavos: 300},
		{DividaID: uuid.New(), EmpresaID: red, CredorTipo: domaindivida.CredorFornecedor, Vencimento: data(2026, 11, 1)},
	}, hoje)

	if resumo.SaldoDevedorCentavos != 13000 || resumo.Dividas != 3 || !resumo.Hoje.Equal(hoje) {
		t.Fatalf("total = %d, dívidas %d", resumo.SaldoDevedorCentavos, resumo.Dividas)
	}
	if len(resumo.PorCredor) != 4 || resumo.PorCredor[0].CredorTipo != domaindivida.CredorBanco || resumo.PorCredor[0].SaldoDevedorCentavos != 3000 ||
		resumo.PorCredor[0].Dividas != 2 || resumo.PorCredor[1].Dividas != 0 || resumo.PorCredor[2].SaldoDevedorCentavos != 10000 ||
		resumo.PorCredor[3].Dividas != 0 {
		t.Errorf("por credor = %+v", resumo.PorCredor)
	}
	if len(resumo.PorEmpresa) != 2 || resumo.PorEmpresa[0].EmpresaID != red || resumo.PorEmpresa[0].SaldoDevedorCentavos != 10000 ||
		resumo.PorEmpresa[0].Dividas != 2 || resumo.PorEmpresa[1].SaldoDevedorCentavos != 3000 {
		t.Errorf("por empresa = %+v", resumo.PorEmpresa)
	}
	if resumo.Vencido != (domaindivida.Totais{AmortizacaoCentavos: 1000, JurosCentavos: 400, Parcelas: 2}) {
		t.Errorf("vencido = %+v", resumo.Vencido)
	}
	if resumo.AVencer != (domaindivida.Totais{AmortizacaoCentavos: 12000, JurosCentavos: 140, Parcelas: 4}) {
		t.Errorf("a vencer = %+v", resumo.AVencer)
	}
	if resumo.AVencer30.Parcelas != 1 || resumo.AVencer90.Parcelas != 2 || resumo.AVencer365.AmortizacaoCentavos != 7000 {
		t.Errorf("faixas = %+v %+v %+v", resumo.AVencer30, resumo.AVencer90, resumo.AVencer365)
	}
}

func TestVencidoDaDivida(t *testing.T) {
	divida := novaDivida(t)
	if _, err := divida.GerarContasAte(data(2026, 11, 30), uuid.New(), agoraDoTeste); err != nil {
		t.Fatalf("GerarContasAte falhou: %v", err)
	}
	divida.Parcelas[0].ContaJuros.Status = domainsolicitacao.StatusPago

	if vencido := divida.VencidoCentavos(data(2026, 11, 15)); vencido != 204000 {
		t.Errorf("vencido = %d", vencido)
	}
	if vencido := divida.VencidoCentavos(data(2026, 10, 10)); vencido != 0 {
		t.Errorf("vencido no dia do vencimento = %d", vencido)
	}
}
