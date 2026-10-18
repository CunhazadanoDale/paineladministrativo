package receber_test

import (
	"context"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/receber"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/receber"
	receberusecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/receber"
	"github.com/google/uuid"
)

type resumosFalsos struct {
	filtros []portsout.FiltroResumo
}

func (r *resumosFalsos) ResumoSemanal(_ context.Context, filtro portsout.FiltroResumo) ([]domainreceber.LinhaResumoSemanal, error) {
	r.filtros = append(r.filtros, filtro)
	return []domainreceber.LinhaResumoSemanal{{Semana: filtro.De, AReceber: domainreceber.TotalResumo{Quantidade: 1, SaldoCentavos: 100}}}, nil
}

func (r *resumosFalsos) Atrasados(_ context.Context, filtro portsout.FiltroResumo) ([]domainreceber.AtrasoPorEmpresa, error) {
	return []domainreceber.AtrasoPorEmpresa{{Vencido: domainreceber.TotalResumo{Quantidade: 2, SaldoCentavos: 40}}}, nil
}

func TestResumoDeContasAReceberRespeitaOAcesso(t *testing.T) {
	c := novoCenario(t)
	resumos := &resumosFalsos{}
	usecase := receberusecases.NewResumoContasAReceberUsecase(resumos, repoEmpresas{c.m}, repoUsuarios{c.m}, repoCargos{c.m})

	resumo, err := usecase.Consultar(c.ctx, portsin.ConsultarResumoInput{UsuarioID: c.financeiro})
	if err != nil {
		t.Fatalf("Consultar falhou: %v", err)
	}
	if resumo.Totais.AReceber.SaldoCentavos != 100 || resumo.Totais.Vencido.Quantidade != 2 {
		t.Fatalf("resumo = %+v", resumo)
	}
	filtro := resumos.filtros[0]
	if len(filtro.EmpresaIDs) != 2 || filtro.De.Weekday() != time.Monday || !filtro.Hoje.Equal(domainreceber.Hoje(time.Now().UTC())) {
		t.Fatalf("filtro = %+v", filtro)
	}

	if _, err := usecase.Consultar(c.ctx, portsin.ConsultarResumoInput{UsuarioID: c.financeiro, Empresa: &c.mia}); err != nil {
		t.Fatal(err)
	}
	if ids := resumos.filtros[1].EmpresaIDs; len(ids) != 1 || ids[0] != c.mia {
		t.Fatalf("filtro da empresa = %v", ids)
	}
	if _, err := usecase.Consultar(c.ctx, portsin.ConsultarResumoInput{UsuarioID: c.administrador}); err != nil {
		t.Fatal(err)
	}
	if resumos.filtros[2].EmpresaIDs != nil {
		t.Fatalf("administrador sem filtro de empresa: %v", resumos.filtros[2].EmpresaIDs)
	}

	_, err = usecase.Consultar(c.ctx, portsin.ConsultarResumoInput{UsuarioID: c.financeiro, Empresa: &c.red})
	esperarErro(t, err, domain.ErrPermissao, "sem acesso")
	_, err = usecase.Consultar(c.ctx, portsin.ConsultarResumoInput{UsuarioID: c.comercial})
	esperarErro(t, err, domain.ErrPermissao, "contas a receber")
	de, ate := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	_, err = usecase.Consultar(c.ctx, portsin.ConsultarResumoInput{UsuarioID: c.financeiro, De: &de, Ate: &ate})
	esperarErro(t, err, domain.ErrValidacao, "anterior ao início")
}

func TestBoletosPendentes(t *testing.T) {
	c := novoCenario(t)
	pendente := c.emBanco(t)
	enviado := c.emBanco(t)
	if _, err := c.titulos.RegistrarEnvioBoleto(c.ctx, portsin.RegistrarEnvioBoletoInput{
		UsuarioID: c.financeiro, TituloID: enviado.ID, Versao: enviado.Versao, Canal: "email", EnviadoEm: time.Now().Add(-time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	c.novoTitulo(t)

	dias := 40
	titulos, err := c.titulos.BoletosPendentes(c.ctx, portsin.BoletosPendentesInput{UsuarioID: c.financeiro, Dias: &dias})
	if err != nil || len(titulos) != 1 || titulos[0].ID != pendente.ID {
		t.Fatalf("pendentes = %+v, %v", titulos, err)
	}
	if limite := c.m.filtros[len(c.m.filtros)-1].BoletoPendenteAte; !limite.Equal(domainreceber.Hoje(time.Now().UTC()).AddDate(0, 0, 40)) {
		t.Fatalf("limite = %v", limite)
	}

	nenhum, err := c.titulos.BoletosPendentes(c.ctx, portsin.BoletosPendentesInput{UsuarioID: c.financeiro})
	if err != nil || len(nenhum) != 0 {
		t.Fatalf("com o padrão de 5 dias = %+v, %v", nenhum, err)
	}

	demais := 91
	_, err = c.titulos.BoletosPendentes(c.ctx, portsin.BoletosPendentesInput{UsuarioID: c.financeiro, Dias: &demais})
	esperarErro(t, err, domain.ErrValidacao, "entre 0 e 90")
	_, err = c.titulos.BoletosPendentes(c.ctx, portsin.BoletosPendentesInput{UsuarioID: c.financeiroRed, Empresa: &c.mia})
	esperarErro(t, err, domain.ErrPermissao, "sem acesso")
	_, err = c.titulos.BoletosPendentes(c.ctx, portsin.BoletosPendentesInput{UsuarioID: uuid.New()})
	esperarErro(t, err, domain.ErrNotFound, "usuário")
}
