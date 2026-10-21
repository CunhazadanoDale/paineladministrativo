package postgres_test

import (
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	domaindivida "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/divida"
	portsdivida "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/divida"
	"github.com/google/uuid"
)

func TestDividaParcelasEmAbertoDescontaOPagoEOCancelado(t *testing.T) {
	c := novoCenarioDividas(t)
	mia, geracao := c.nova(t, c.mia)
	red, _ := c.nova(t, c.red)
	if _, err := c.banco.Exec(`UPDATE solicitacao SET status = 'pago' WHERE id = $1`, geracao.Contas[0].Solicitacao.ID); err != nil {
		t.Fatalf("pagamento de teste falhou: %v", err)
	}
	if _, err := c.banco.Exec(`UPDATE divida SET status = 'cancelada', motivo = 'teste' WHERE id = $1`, red.ID); err != nil {
		t.Fatalf("cancelamento de teste falhou: %v", err)
	}
	consultas := postgres.NewDividaConsultaRepository(c.banco)

	parcelas, err := consultas.ParcelasEmAberto(c.ctx, nil)
	if err != nil {
		t.Fatalf("ParcelasEmAberto: %v", err)
	}
	if len(parcelas) != 3 {
		t.Fatalf("parcelas em aberto = %+v", parcelas)
	}
	primeira := parcelas[0]
	if primeira.DividaID != mia.ID || primeira.EmpresaID != c.mia || primeira.CredorTipo != domaindivida.CredorBanco ||
		primeira.AmortizacaoCentavos != 0 || primeira.JurosCentavos != 3000 || !primeira.Vencimento.Equal(c.hoje) {
		t.Errorf("primeira parcela em aberto = %+v", primeira)
	}
	resumo := domaindivida.NovoResumo(parcelas, c.hoje)
	if resumo.SaldoDevedorCentavos != 200000 || resumo.Dividas != 1 || resumo.AVencer.JurosCentavos != 5000 {
		t.Errorf("resumo = %+v", resumo)
	}

	daRed, err := consultas.ParcelasEmAberto(c.ctx, []uuid.UUID{c.red})
	if err != nil || len(daRed) != 0 {
		t.Errorf("parcelas da Red cancelada = %+v, %v", daRed, err)
	}
}

func TestDividaListaFiltraQuitadaEVencidaComAsParcelas(t *testing.T) {
	c := novoCenarioDividas(t)
	divida, geracao := c.nova(t, c.mia)
	verdadeiro, falso := true, false
	amanha := c.hoje.AddDate(0, 0, 1)

	lista, err := c.dividas.Listar(c.ctx, portsdivida.DividaFiltro{Vencidas: &verdadeiro, Hoje: amanha})
	if err != nil || len(lista) != 1 || lista[0].ID != divida.ID || len(lista[0].Parcelas) != 3 ||
		lista[0].Parcelas[0].ContaJuros == nil || lista[0].VencidoCentavos(amanha) != 103000 {
		t.Fatalf("vencidas amanhã = %+v, %v", lista, err)
	}
	lista, err = c.dividas.Listar(c.ctx, portsdivida.DividaFiltro{Vencidas: &verdadeiro, Hoje: c.hoje})
	if err != nil || len(lista) != 0 {
		t.Fatalf("vencidas hoje = %+v, %v", lista, err)
	}
	lista, err = c.dividas.Listar(c.ctx, portsdivida.DividaFiltro{Quitada: &verdadeiro})
	if err != nil || len(lista) != 0 {
		t.Fatalf("quitadas = %+v, %v", lista, err)
	}

	if _, err := c.banco.Exec(`
		UPDATE divida_parcela SET status = 'cancelada' WHERE divida_id = $1 AND numero > 1
	`, divida.ID); err != nil {
		t.Fatalf("ajuste de teste falhou: %v", err)
	}
	for _, conta := range geracao.Contas {
		if _, err := c.banco.Exec(`UPDATE solicitacao SET status = 'pago' WHERE id = $1`, conta.Solicitacao.ID); err != nil {
			t.Fatalf("pagamento de teste falhou: %v", err)
		}
	}
	lista, err = c.dividas.Listar(c.ctx, portsdivida.DividaFiltro{Quitada: &verdadeiro})
	if err != nil || len(lista) != 1 || !lista[0].Quitada() || lista[0].SaldoDevedor() != 0 {
		t.Fatalf("quitada = %+v, %v", lista, err)
	}
	lista, err = c.dividas.Listar(c.ctx, portsdivida.DividaFiltro{Quitada: &falso})
	if err != nil || len(lista) != 0 {
		t.Fatalf("não quitadas = %+v, %v", lista, err)
	}
}

func TestDividaComparaDatasSemDependerDoFusoDoBanco(t *testing.T) {
	c := novoCenarioDividas(t)
	divida, _ := c.nova(t, c.mia)
	if _, err := c.banco.Exec(`DO $$ BEGIN EXECUTE format('ALTER DATABASE %I SET timezone = %L', current_database(), 'America/Sao_Paulo'); END $$`); err != nil {
		t.Fatalf("troca do fuso falhou: %v", err)
	}
	c.banco.SetMaxIdleConns(0)
	var fuso string
	if err := c.banco.Get(&fuso, `SHOW timezone`); err != nil || fuso != "America/Sao_Paulo" {
		t.Fatalf("fuso da sessão = %q, %v", fuso, err)
	}

	segunda := c.hoje.AddDate(0, 2, 0)
	ids, err := c.dividas.ComParcelasAGerar(c.ctx, segunda, nil)
	if err != nil || len(ids) != 1 || ids[0] != divida.ID {
		t.Fatalf("parcela que vence no limite = %v, %v", ids, err)
	}

	verdadeiro := true
	lista, err := c.dividas.Listar(c.ctx, portsdivida.DividaFiltro{Vencidas: &verdadeiro, Hoje: c.hoje})
	if err != nil || len(lista) != 0 {
		t.Fatalf("parcela que vence hoje apareceu vencida = %+v, %v", lista, err)
	}
	lista, err = c.dividas.Listar(c.ctx, portsdivida.DividaFiltro{Vencidas: &verdadeiro, Hoje: c.hoje.AddDate(0, 0, 1)})
	if err != nil || len(lista) != 1 {
		t.Fatalf("parcela de ontem não apareceu vencida = %+v, %v", lista, err)
	}
}
