package divida_test

import (
	"context"
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaindivida "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/divida"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/divida"
)

func TestResumoRespeitaEscopoEContasPagas(t *testing.T) {
	c := novoCenario()
	visao := c.criar(t)
	c.memoria.solicitacoes[visao.Divida.Parcelas[0].ContaAmortizacao.SolicitacaoID].Status = domainsolicitacao.StatusPago
	usecase := c.memoria.usecase()

	resumo, err := usecase.Resumo(context.Background(), portsin.ResumoInput{UsuarioID: c.financeiro})
	if err != nil {
		t.Fatalf("Resumo falhou: %v", err)
	}
	if resumo.SaldoDevedorCentavos != 100000 || resumo.Dividas != 1 || !resumo.Hoje.Equal(c.hoje) {
		t.Fatalf("resumo = %+v", resumo)
	}
	if resumo.PorCredor[0].CredorTipo != domaindivida.CredorBanco || resumo.PorCredor[0].SaldoDevedorCentavos != 100000 ||
		len(resumo.PorEmpresa) != 1 || resumo.PorEmpresa[0].EmpresaID != c.empresa {
		t.Errorf("agrupamentos = %+v %+v", resumo.PorCredor, resumo.PorEmpresa)
	}
	if resumo.Vencido.Parcelas != 0 || resumo.AVencer != (domaindivida.Totais{AmortizacaoCentavos: 100000, JurosCentavos: 3000, Parcelas: 2}) {
		t.Errorf("vencimentos = %+v %+v", resumo.Vencido, resumo.AVencer)
	}

	outra := c.memoria.empresa(true)
	if _, err := usecase.Resumo(context.Background(), portsin.ResumoInput{UsuarioID: c.financeiro, Empresa: &outra}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("empresa fora do acesso: %v", err)
	}
	deOutra, err := usecase.Resumo(context.Background(), portsin.ResumoInput{UsuarioID: c.memoria.usuario(true, outra)})
	if err != nil || deOutra.Dividas != 0 || deOutra.SaldoDevedorCentavos != 0 || len(deOutra.PorCredor) != 4 {
		t.Errorf("resumo de outra empresa = %+v, %v", deOutra, err)
	}
	if _, err := usecase.Resumo(context.Background(), portsin.ResumoInput{UsuarioID: c.memoria.usuario(false, c.empresa)}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("sem financeiro: %v", err)
	}
}
