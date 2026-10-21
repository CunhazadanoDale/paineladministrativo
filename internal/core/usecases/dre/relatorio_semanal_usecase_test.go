package dre_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	domaindre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/dre"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/dre"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/dre"
	usecasedre "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/dre"
	"github.com/google/uuid"
)

var (
	_ portsout.RelatorioSemanalRepository = (*repositorioRelatorios)(nil)
	_ portsout.SaldoRepository            = (*repositorioSaldos)(nil)
)

type repositorioRelatorios struct {
	filtros       []portsout.FiltroSemanal
	limiteEstoque int
	erro          error
	notas         []domaindre.NotasEmitidas
}

func (r *repositorioRelatorios) NotasEmitidas(_ context.Context, filtro portsout.FiltroSemanal) ([]domaindre.NotasEmitidas, error) {
	r.filtros = append(r.filtros, filtro)

	return r.notas, r.erro
}

func (r *repositorioRelatorios) PagamentosFeitos(_ context.Context, filtro portsout.FiltroSemanal) ([]domaindre.PagamentosFeitos, error) {
	r.filtros = append(r.filtros, filtro)

	return nil, nil
}

func (r *repositorioRelatorios) RecebimentosFeitos(_ context.Context, filtro portsout.FiltroSemanal) ([]domaindre.RecebimentosFeitos, error) {
	r.filtros = append(r.filtros, filtro)

	return nil, nil
}

func (r *repositorioRelatorios) TitulosVencidos(_ context.Context, filtro portsout.FiltroSemanal) (domaindre.Vencidos[domaindre.TituloVencido], error) {
	r.filtros = append(r.filtros, filtro)

	return domaindre.Vencidos[domaindre.TituloVencido]{}, nil
}

func (r *repositorioRelatorios) ContasVencidas(_ context.Context, filtro portsout.FiltroSemanal) (domaindre.Vencidos[domaindre.ContaVencida], error) {
	r.filtros = append(r.filtros, filtro)

	return domaindre.Vencidos[domaindre.ContaVencida]{}, nil
}

func (r *repositorioRelatorios) EstoqueBaixo(_ context.Context, limite int) (domaindre.EstoqueBaixo, error) {
	r.limiteEstoque = limite

	return domaindre.EstoqueBaixo{Total: 7}, nil
}

type repositorioSaldos struct {
	empresas [][]uuid.UUID
	erro     error
}

func (r *repositorioSaldos) Saldos(_ context.Context, empresaIDs []uuid.UUID) ([]*domaincaixa.SaldoConta, error) {
	r.empresas = append(r.empresas, empresaIDs)

	return []*domaincaixa.SaldoConta{{EmpresaID: uuid.New()}}, r.erro
}

func (c *cenario) relatorioSemanal(relatorios *repositorioRelatorios, saldos *repositorioSaldos) *usecasedre.RelatorioSemanalUsecaseImpl {
	return usecasedre.NewRelatorioSemanalUsecase(relatorios, saldos, c.usuarios, c.cargos, c.empresas)
}

func TestRelatorioSemanalExigeFinanceiroEAcessoAEmpresa(t *testing.T) {
	c := novoCenario(t)
	relatorios, saldos := &repositorioRelatorios{}, &repositorioSaldos{}
	usecase := c.relatorioSemanal(relatorios, saldos)

	if _, err := usecase.Consultar(context.Background(), portsin.ConsultarSemanaInput{UsuarioID: c.comercial}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("comercial = %v, esperado erro de permissão", err)
	}
	if _, err := usecase.Consultar(context.Background(), portsin.ConsultarSemanaInput{UsuarioID: c.restrito, Empresa: &c.empresaB}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("empresa fora do acesso = %v, esperado erro de permissão", err)
	}
	if len(relatorios.filtros) != 0 || len(saldos.empresas) != 0 {
		t.Errorf("consulta recusada não deveria ir ao banco")
	}
}

func TestRelatorioSemanalUsaASemanaEOEscopoDoUsuario(t *testing.T) {
	c := novoCenario(t)
	relatorios, saldos := &repositorioRelatorios{}, &repositorioSaldos{}
	relatorios.notas = []domaindre.NotasEmitidas{{EmpresaID: c.empresaA, Natureza: "venda", Total: domaindre.TotalSemanal{Quantidade: 1, ValorCentavos: 100}}}
	quinta := time.Date(2026, time.October, 15, 0, 0, 0, 0, time.UTC)

	relatorio, err := c.relatorioSemanal(relatorios, saldos).Consultar(context.Background(), portsin.ConsultarSemanaInput{
		UsuarioID: c.restrito, Semana: &quinta,
	})
	if err != nil {
		t.Fatalf("Consultar = %v", err)
	}

	if len(relatorios.filtros) != 5 {
		t.Fatalf("consultas = %d, esperado 5", len(relatorios.filtros))
	}
	filtro := relatorios.filtros[0]
	if !filtro.Intervalo.Inicio.Equal(time.Date(2026, time.October, 12, 0, 0, 0, 0, time.UTC)) ||
		!filtro.Intervalo.Fim.Equal(time.Date(2026, time.October, 19, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("intervalo = %s a %s", filtro.Intervalo.Inicio, filtro.Intervalo.Fim)
	}
	if len(filtro.EmpresaIDs) != 1 || filtro.EmpresaIDs[0] != c.empresaA || filtro.Limite != domaindre.LimiteDasListasDoRelatorio {
		t.Errorf("filtro = %+v", filtro)
	}
	if !filtro.Hoje.Equal(domaindre.Hoje(time.Now())) {
		t.Errorf("hoje = %s, esperado o dia de Brasília", filtro.Hoje)
	}
	if relatorios.limiteEstoque != domaindre.LimiteDasListasDoRelatorio {
		t.Errorf("limite do estoque = %d", relatorios.limiteEstoque)
	}
	if len(saldos.empresas) != 1 || len(saldos.empresas[0]) != 1 || saldos.empresas[0][0] != c.empresaA {
		t.Errorf("saldos consultados para %v, esperado só a empresa A", saldos.empresas)
	}
	if relatorio.TotalNotasVenda.ValorCentavos != 100 || relatorio.EstoqueBaixo.Total != 7 || len(relatorio.Saldos.Contas) != 1 {
		t.Errorf("relatório = %+v", relatorio)
	}
}

func TestRelatorioSemanalRecusaSemanaFuturaEPropagaErros(t *testing.T) {
	c := novoCenario(t)
	futura := time.Now().AddDate(0, 0, 14)

	_, err := c.relatorioSemanal(&repositorioRelatorios{}, &repositorioSaldos{}).Consultar(context.Background(), portsin.ConsultarSemanaInput{
		UsuarioID: c.financeiro, Semana: &futura,
	})
	if !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("semana futura = %v, esperado erro de validação", err)
	}

	if _, err := c.relatorioSemanal(&repositorioRelatorios{erro: errBanco}, &repositorioSaldos{}).Consultar(context.Background(), portsin.ConsultarSemanaInput{UsuarioID: c.financeiro}); !errors.Is(err, errBanco) {
		t.Errorf("erro do relatório = %v", err)
	}
	if _, err := c.relatorioSemanal(&repositorioRelatorios{}, &repositorioSaldos{erro: errBanco}).Consultar(context.Background(), portsin.ConsultarSemanaInput{UsuarioID: c.financeiro}); !errors.Is(err, errBanco) {
		t.Errorf("erro dos saldos = %v", err)
	}
}
