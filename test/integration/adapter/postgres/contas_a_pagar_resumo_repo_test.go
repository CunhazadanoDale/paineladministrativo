package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/solicitacao"
	"github.com/google/uuid"
)

func dataDoResumo(ano int, mes time.Month, dia int) time.Time {
	return time.Date(ano, mes, dia, 0, 0, 0, 0, time.UTC)
}

func (c *cenarioBancoClassificacao) solicitacaoDoResumo(t *testing.T, empresaID uuid.UUID, status domainsolicitacao.Status, prazo time.Time, valor int64, estimado bool) {
	t.Helper()

	solicitacao := c.novaSolicitacaoNaEmpresa(t, empresaID)
	solicitacao.Status = status
	solicitacao.Prazo = domainsolicitacao.PrazoPagamentoDe(prazo)
	solicitacao.Valor = domainsolicitacao.ValorDe(valor)
	solicitacao.Estimado = estimado
	if err := c.gravar(t, solicitacao); err != nil {
		t.Fatalf("criação falhou: %v", err)
	}
}

func TestResumoDeContasAPagarAgrupaPorSemanaEEmpresa(t *testing.T) {
	c := cenarioClassificacao(t)
	ctx := context.Background()
	red := inserirEmpresa(t, c.banco, "11.444.777/0001-61", "Red Diamond")
	mia := c.empresaID

	c.solicitacaoDoResumo(t, mia, domainsolicitacao.StatusPendenteAprovacao, dataDoResumo(2026, time.October, 15), 100, true)
	c.solicitacaoDoResumo(t, mia, domainsolicitacao.StatusAprovado, dataDoResumo(2026, time.October, 16), 200, false)
	c.solicitacaoDoResumo(t, mia, domainsolicitacao.StatusAprovado, dataDoResumo(2026, time.October, 13), 50, false)
	c.solicitacaoDoResumo(t, mia, domainsolicitacao.StatusAprovado, dataDoResumo(2026, time.October, 20), 300, false)
	c.solicitacaoDoResumo(t, mia, domainsolicitacao.StatusPago, dataDoResumo(2026, time.October, 15), 5000, false)
	c.solicitacaoDoResumo(t, red, domainsolicitacao.StatusPendenteAprovacao, dataDoResumo(2026, time.October, 15), 70, false)
	c.solicitacaoDoResumo(t, mia, domainsolicitacao.StatusAprovado, dataDoResumo(2026, time.September, 1), 999, false)
	c.solicitacaoDoResumo(t, mia, domainsolicitacao.StatusAprovado, dataDoResumo(2027, time.January, 1), 7777, false)

	repo := postgres.NewContasAPagarResumoRepository(c.banco)
	filtro := portsout.FiltroResumo{
		De:   dataDoResumo(2026, time.October, 12),
		Ate:  dataDoResumo(2026, time.October, 25),
		Hoje: dataDoResumo(2026, time.October, 14),
	}

	semanas, err := repo.ResumoSemanal(ctx, filtro)
	if err != nil {
		t.Fatalf("resumo falhou: %v", err)
	}
	if len(semanas) != 3 {
		t.Fatalf("linhas = %+v, esperado três", semanas)
	}

	porChave := map[string]domainsolicitacao.LinhaResumoSemanal{}
	for _, linha := range semanas {
		porChave[linha.Semana.Format("2006-01-02")+"/"+linha.EmpresaID.String()] = linha
	}

	primeira := porChave["2026-10-12/"+mia.String()]
	if primeira.Previsto != (domainsolicitacao.TotalResumo{Quantidade: 1, ValorCentavos: 100}) ||
		primeira.APagar != (domainsolicitacao.TotalResumo{Quantidade: 1, ValorCentavos: 200}) ||
		primeira.Vencido != (domainsolicitacao.TotalResumo{Quantidade: 1, ValorCentavos: 50}) ||
		primeira.EstimadoCentavos != 100 {
		t.Errorf("semana 12/10 da MIA = %+v", primeira)
	}
	if daRed := porChave["2026-10-12/"+red.String()]; daRed.Previsto != (domainsolicitacao.TotalResumo{Quantidade: 1, ValorCentavos: 70}) {
		t.Errorf("semana 12/10 da Red = %+v", daRed)
	}
	if segunda := porChave["2026-10-19/"+mia.String()]; segunda.APagar != (domainsolicitacao.TotalResumo{Quantidade: 1, ValorCentavos: 300}) ||
		segunda.Previsto.Quantidade != 0 || segunda.Vencido.Quantidade != 0 {
		t.Errorf("semana 19/10 da MIA = %+v", segunda)
	}

	atrasadas, err := repo.Atrasadas(ctx, filtro)
	if err != nil || len(atrasadas) != 1 || atrasadas[0].EmpresaID != mia ||
		atrasadas[0].Vencido != (domainsolicitacao.TotalResumo{Quantidade: 1, ValorCentavos: 999}) {
		t.Errorf("atrasadas = %+v, %v", atrasadas, err)
	}

	filtro.EmpresaIDs = []uuid.UUID{red}
	soDaRed, err := repo.ResumoSemanal(ctx, filtro)
	if err != nil || len(soDaRed) != 1 || soDaRed[0].EmpresaID != red {
		t.Errorf("resumo só da Red = %+v, %v", soDaRed, err)
	}
	if semAtraso, err := repo.Atrasadas(ctx, filtro); err != nil || len(semAtraso) != 0 {
		t.Errorf("atrasadas da Red = %+v, %v", semAtraso, err)
	}
}

func TestContarSolicitacoesUsaOsMesmosFiltrosDaListagem(t *testing.T) {
	c := cenarioClassificacao(t)
	ctx := context.Background()
	red := inserirEmpresa(t, c.banco, "11.444.777/0001-61", "Red Diamond")
	mia := c.empresaID

	c.solicitacaoDoResumo(t, mia, domainsolicitacao.StatusAprovado, dataDoResumo(2026, time.October, 13), 50, false)
	c.solicitacaoDoResumo(t, mia, domainsolicitacao.StatusAprovado, dataDoResumo(2026, time.October, 20), 300, false)
	c.solicitacaoDoResumo(t, mia, domainsolicitacao.StatusPendenteAprovacao, dataDoResumo(2026, time.October, 1), 10, false)
	c.solicitacaoDoResumo(t, red, domainsolicitacao.StatusAprovado, dataDoResumo(2026, time.October, 1), 70, false)

	hoje := dataDoResumo(2026, time.October, 14)
	casos := map[string]struct {
		filtro   portsout.SolicitacaoFiltro
		esperado int
	}{
		"aprovadas da MIA":   {portsout.SolicitacaoFiltro{Status: domainsolicitacao.StatusAprovado, EmpresaIDs: []uuid.UUID{mia}}, 2},
		"aprovadas de todas": {portsout.SolicitacaoFiltro{Status: domainsolicitacao.StatusAprovado}, 3},
		"vencidas da MIA":    {portsout.SolicitacaoFiltro{VencidasEm: &hoje, EmpresaIDs: []uuid.UUID{mia}}, 2},
		"sem acesso":         {portsout.SolicitacaoFiltro{EmpresaIDs: []uuid.UUID{}}, 0},
	}

	for nome, caso := range casos {
		total, err := c.solicitacoes.Contar(ctx, caso.filtro)
		if err != nil || total != caso.esperado {
			t.Errorf("%s = %d, %v; esperado %d", nome, total, err, caso.esperado)
		}
	}
}
