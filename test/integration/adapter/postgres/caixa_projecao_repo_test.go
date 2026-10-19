package postgres_test

import (
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	portscaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/caixa"
	"github.com/google/uuid"
)

func TestProjecaoLeSaldoPrevistosERecorrencias(t *testing.T) {
	c := novoCenarioConciliacao(t)
	projecoes := postgres.NewCaixaProjecaoRepository(c.banco)
	c.inserirConta(t, c.empresaID, "88888", "Sem abertura")

	c.importarLinhas(t,
		transacaoNoDia("REC", diaDoTitulo(10, 5), 1000),
		transacaoNoDia("TAR", diaDoTitulo(10, 6), -300),
	)
	paga := c.aprovadaClassificada(t)
	if _, err := c.pagar(t, paga, c.contaID, nil); err != nil {
		t.Fatalf("pagar = %v", err)
	}
	aberta := c.aprovadaClassificada(t)
	c.novoTitulo(t, "NF-1", diaDoTitulo(11, 10))

	partida, err := projecoes.SaldoDePartida(c.ctx, []uuid.UUID{c.empresaID})
	if err != nil {
		t.Fatalf("SaldoDePartida = %v", err)
	}
	if partida.ExtratoCentavos != 700 || partida.EmTransitoCentavos != -148500 || partida.ContasConsideradas != 1 || partida.ContasSemAbertura != 1 {
		t.Errorf("saldo de partida = %+v", partida)
	}
	if deOutra, _ := projecoes.SaldoDePartida(c.ctx, []uuid.UUID{uuid.New()}); deOutra.ContasConsideradas != 0 || deOutra.ExtratoCentavos != 0 {
		t.Errorf("saldo de outra empresa = %+v", deOutra)
	}

	ate := diaDoTitulo(12, 31)
	titulos, err := projecoes.TitulosAReceber(c.ctx, []uuid.UUID{c.empresaID}, ate)
	if err != nil || len(titulos) != 1 || titulos[0].ValorCentavos != 150000 || !titulos[0].Data.Equal(diaDoTitulo(11, 10)) {
		t.Errorf("títulos a receber = %+v, %v", titulos, err)
	}
	if antes, _ := projecoes.TitulosAReceber(c.ctx, nil, diaDoTitulo(11, 9)); len(antes) != 0 {
		t.Errorf("títulos até 09/11 = %+v", antes)
	}

	contas, err := projecoes.ContasAPagar(c.ctx, []uuid.UUID{c.empresaID}, time.Now().UTC().AddDate(1, 0, 0))
	if err != nil || len(contas) != 1 || contas[0].ValorCentavos != aberta.Valor.Centavos() {
		t.Errorf("contas a pagar = %+v, %v", contas, err)
	}

	recorrencia := uuid.New()
	if _, err := c.banco.ExecContext(c.ctx, `
		UPDATE solicitacao SET origem = 'recorrencia', origem_id = $1, competencia = '2026-10-01' WHERE id = $2
	`, recorrencia, aberta.ID); err != nil {
		t.Fatalf("marcar a recorrência falhou: %v", err)
	}
	geradas, err := projecoes.RecorrenciasGeradas(c.ctx, []portscaixa.RecorrenciaNaCompetencia{
		{RecorrenciaID: recorrencia, Competencia: diaDoTitulo(10, 1)},
		{RecorrenciaID: recorrencia, Competencia: diaDoTitulo(11, 1)},
	})
	if err != nil || len(geradas) != 1 || !geradas[portscaixa.RecorrenciaNaCompetencia{RecorrenciaID: recorrencia, Competencia: diaDoTitulo(10, 1)}] {
		t.Errorf("recorrências geradas = %+v, %v", geradas, err)
	}

	diario, err := projecoes.MovimentoDiario(c.ctx, portscaixa.MovimentoDiarioFiltro{
		EmpresaIDs: []uuid.UUID{c.empresaID}, De: diaDoTitulo(10, 1), Ate: diaDoTitulo(10, 31),
	})
	if err != nil || len(diario) != 2 {
		t.Fatalf("movimento diário = %+v, %v", diario, err)
	}
	if diario[0].EntradasCentavos != 1000 || diario[0].SaldoCentavos != 1000 || diario[1].SaidasCentavos != 300 || diario[1].SaldoCentavos != 700 {
		t.Errorf("dias = %+v, %+v", diario[0], diario[1])
	}

	soDia6, err := projecoes.MovimentoDiario(c.ctx, portscaixa.MovimentoDiarioFiltro{
		ContaBancariaID: &c.contaID, De: diaDoTitulo(10, 6), Ate: diaDoTitulo(10, 6),
	})
	if err != nil || len(soDia6) != 1 || soDia6[0].SaldoCentavos != 700 || soDia6[0].ContaBancariaID != c.contaID {
		t.Errorf("movimento do dia 6 = %+v, %v", soDia6, err)
	}
}
