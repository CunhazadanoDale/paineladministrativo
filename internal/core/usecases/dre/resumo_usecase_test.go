package dre_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaindre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/dre"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/dre"
	usecasedre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/dre"
	"github.com/google/uuid"
)

func mesDoTeste(t *testing.T, texto string) *domaindre.Mes {
	t.Helper()

	mes, err := domaindre.NovoMes(texto, "mês")
	if err != nil {
		t.Fatalf("NovoMes(%q) falhou: %v", texto, err)
	}

	return &mes
}

func (c *cenario) resumoDRE() *usecasedre.ResumoDREUsecaseImpl {
	return usecasedre.NewResumoDREUsecase(c.resumos, c.usuarios, c.cargos, c.empresas)
}

func (c *cenario) faturamento() *usecasedre.FaturamentoUsecaseImpl {
	return usecasedre.NewFaturamentoUsecase(c.resumos, c.usuarios, c.cargos, c.empresas)
}

func TestResumoDREExigePerfilFinanceiro(t *testing.T) {
	c := novoCenario(t)

	for _, usuarioID := range []uuid.UUID{c.financeiro, c.administrador} {
		if _, err := c.resumoDRE().Consultar(context.Background(), portsin.ConsultarPeriodoInput{UsuarioID: usuarioID}); err != nil {
			t.Errorf("usuário com financeiro ou administrador deveria consultar: %v", err)
		}
	}

	_, err := c.resumoDRE().Consultar(context.Background(), portsin.ConsultarPeriodoInput{UsuarioID: c.comercial})
	if !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("comercial = %v, esperado erro de permissão", err)
	}
	if _, err := c.faturamento().Consultar(context.Background(), portsin.ConsultarPeriodoInput{UsuarioID: c.comercial}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("faturamento do comercial = %v, esperado erro de permissão", err)
	}

	if _, err := c.resumoDRE().Consultar(context.Background(), portsin.ConsultarPeriodoInput{UsuarioID: uuid.New()}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("usuário inexistente = %v, esperado não encontrado", err)
	}
	if _, err := c.resumoDRE().Consultar(context.Background(), portsin.ConsultarPeriodoInput{}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("usuário vazio = %v, esperado erro de validação", err)
	}
	if len(c.resumos.filtros) != 4 {
		t.Errorf("consultas ao banco = %d, esperado só as dos dois perfis permitidos", len(c.resumos.filtros))
	}
}

func TestResumoDREFiltraPeloAcessoDoUsuario(t *testing.T) {
	c := novoCenario(t)
	ctx := context.Background()

	if _, err := c.resumoDRE().Consultar(ctx, portsin.ConsultarPeriodoInput{UsuarioID: c.financeiro}); err != nil {
		t.Fatalf("Consultar = %v", err)
	}
	if c.resumos.filtros[0].EmpresaIDs != nil {
		t.Errorf("acesso total deveria consultar sem filtro de empresa: %v", c.resumos.filtros[0].EmpresaIDs)
	}

	if _, err := c.resumoDRE().Consultar(ctx, portsin.ConsultarPeriodoInput{UsuarioID: c.restrito}); err != nil {
		t.Fatalf("Consultar restrito = %v", err)
	}
	if ids := c.resumos.filtros[2].EmpresaIDs; len(ids) != 1 || ids[0] != c.empresaA {
		t.Errorf("acesso restrito = %v, esperado só a empresa A", ids)
	}

	if _, err := c.resumoDRE().Consultar(ctx, portsin.ConsultarPeriodoInput{UsuarioID: c.restrito, Empresa: &c.empresaB}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("empresa fora do acesso = %v, esperado erro de permissão", err)
	}

	resumo, err := c.resumoDRE().Consultar(ctx, portsin.ConsultarPeriodoInput{UsuarioID: c.financeiro, Empresa: &c.empresaB})
	if err != nil {
		t.Fatalf("Consultar com empresa = %v", err)
	}
	if ids := c.resumos.filtros[len(c.resumos.filtros)-1].EmpresaIDs; len(ids) != 1 || ids[0] != c.empresaB {
		t.Errorf("filtro = %v, esperado só a empresa B", ids)
	}
	if len(resumo.Total.Empresas) != 1 || resumo.Total.Empresas[0].EmpresaID != c.empresaB {
		t.Errorf("empresa filtrada deveria aparecer mesmo sem dados: %+v", resumo.Total.Empresas)
	}
}

func TestResumoDREConsultaOIntervaloDoPeriodoEMontaADRE(t *testing.T) {
	c := novoCenario(t)
	setembro := mesDoTeste(t, "2026-09")
	c.resumos.lancamentos = []domaindre.Lancamento{{
		Mes: *setembro, EmpresaID: c.empresaA, Linha: domaindre.LinhaReceitaBruta,
		Item: domaindre.ItemFixo(domaindre.ItemNFVenda), ValorCentavos: 1000, Quantidade: 1,
	}}
	c.resumos.semCusto = []domaindre.ConsumoSemCusto{{Mes: *setembro, EmpresaID: c.empresaA, Quantidade: 3}}

	resumo, err := c.resumoDRE().Consultar(context.Background(), portsin.ConsultarPeriodoInput{
		UsuarioID: c.financeiro, De: mesDoTeste(t, "2026-08"), Ate: setembro,
	})
	if err != nil {
		t.Fatalf("Consultar = %v", err)
	}

	intervalo := c.resumos.filtros[0].Intervalo
	if !intervalo.Inicio.Equal(time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)) ||
		!intervalo.Fim.Equal(time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("intervalo = %s a %s", intervalo.Inicio, intervalo.Fim)
	}
	if len(resumo.Meses) != 2 || resumo.Meses[1].Consolidado.Valor(domaindre.LinhaReceitaBruta) != 1000 {
		t.Errorf("resumo = %+v", resumo.Meses)
	}
	if resumo.Total.Consolidado.QuantidadeSemCusto != 3 {
		t.Errorf("sem custo = %d, esperado 3", resumo.Total.Consolidado.QuantidadeSemCusto)
	}
}

func TestResumoDRERecusaPeriodoInvalidoSemIrAoBanco(t *testing.T) {
	c := novoCenario(t)

	_, err := c.resumoDRE().Consultar(context.Background(), portsin.ConsultarPeriodoInput{
		UsuarioID: c.financeiro, De: mesDoTeste(t, "2024-01"), Ate: mesDoTeste(t, "2026-01"),
	})
	if !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("25 meses = %v, esperado erro de validação", err)
	}
	if len(c.resumos.filtros) != 0 {
		t.Errorf("período inválido não deveria consultar o banco")
	}
}

func TestResumoDREPropagaErroDoBanco(t *testing.T) {
	c := novoCenario(t)
	c.resumos.erro = errBanco

	if _, err := c.resumoDRE().Consultar(context.Background(), portsin.ConsultarPeriodoInput{UsuarioID: c.financeiro}); !errors.Is(err, errBanco) {
		t.Errorf("erro = %v, esperado o erro do banco", err)
	}
	if _, err := c.faturamento().Consultar(context.Background(), portsin.ConsultarPeriodoInput{UsuarioID: c.financeiro}); !errors.Is(err, errBanco) {
		t.Errorf("erro do faturamento = %v, esperado o erro do banco", err)
	}
}

func TestFaturamentoAceitaSessentaMesesEFiltraPeloAcesso(t *testing.T) {
	c := novoCenario(t)
	janeiro := mesDoTeste(t, "2026-01")
	c.resumos.faturamentos = []domaindre.Faturamento{{Mes: *janeiro, EmpresaID: c.empresaA, Quantidade: 2, ValorCentavos: 500}}

	resumo, err := c.faturamento().Consultar(context.Background(), portsin.ConsultarPeriodoInput{
		UsuarioID: c.restrito, De: mesDoTeste(t, "2021-02"), Ate: janeiro,
	})
	if err != nil {
		t.Fatalf("Consultar = %v", err)
	}

	if ids := c.resumos.filtros[0].EmpresaIDs; len(ids) != 1 || ids[0] != c.empresaA {
		t.Errorf("filtro = %v, esperado só a empresa A", ids)
	}
	if len(resumo.Meses) != 60 || resumo.Consolidado.ValorCentavos != 500 || resumo.Consolidado.TicketMedioCentavos() != 250 {
		t.Errorf("faturamento com %d meses e consolidado %+v", len(resumo.Meses), resumo.Consolidado)
	}

	if _, err := c.faturamento().Consultar(context.Background(), portsin.ConsultarPeriodoInput{
		UsuarioID: c.restrito, De: mesDoTeste(t, "2021-01"), Ate: janeiro,
	}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("61 meses = %v, esperado erro de validação", err)
	}
}
