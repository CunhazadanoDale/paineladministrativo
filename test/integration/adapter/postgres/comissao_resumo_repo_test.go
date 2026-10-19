package postgres_test

import (
	"testing"

	portscomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/comissao"
	"github.com/google/uuid"
)

func TestComissaoRepositoryResumeACompetenciaPorRepresentante(t *testing.T) {
	c := novoCenarioFechamentos(t)
	fechada := c.comissaoNaCompetencia(t, c.anterior, 2)
	c.ajusteNaCompetencia(t, c.anterior, -50)
	gravacao := c.gravacao(t, c.saldos(t)[0])
	if err := c.comissoes.Fechar(c.ctx, gravacao); err != nil {
		t.Fatalf("Fechar: %v", err)
	}
	aberta := c.comissaoNaCompetencia(t, c.anterior, 3)
	c.comissaoNaCompetencia(t, c.anterior.Seguinte(), 1)

	linhas, err := c.comissoes.Resumo(c.ctx, portscomissao.ResumoFiltro{Competencia: c.anterior})
	if err != nil {
		t.Fatalf("Resumo: %v", err)
	}
	comissoes := fechada.ValorCentavos + aberta.ValorCentavos
	if len(linhas) != 1 || linhas[0].RepresentanteID != c.carlos.ID || linhas[0].RepresentanteNome != "Carlos" || linhas[0].EmpresaID != c.mia {
		t.Fatalf("linhas = %+v", linhas)
	}
	linha := linhas[0]
	if linha.ComissoesCentavos != comissoes || linha.AjustesCentavos != -50 || linha.EmAbertoCentavos != aberta.ValorCentavos ||
		linha.FechadoCentavos != fechada.ValorCentavos-50 || linha.PagoCentavos != 0 {
		t.Fatalf("valores = %+v", linha)
	}

	if _, err := c.banco.ExecContext(c.ctx, `UPDATE solicitacao SET status = 'pago' WHERE id = $1`, gravacao.Solicitacao.ID); err != nil {
		t.Fatalf("marcar a conta como paga: %v", err)
	}
	linhas, err = c.comissoes.Resumo(c.ctx, portscomissao.ResumoFiltro{Competencia: c.anterior, Empresa: &c.mia, RepresentanteID: &c.carlos.ID})
	if err != nil || len(linhas) != 1 || linhas[0].PagoCentavos != fechada.ValorCentavos-50 {
		t.Fatalf("resumo depois do pagamento = %+v, %v", linhas, err)
	}

	if daRed, err := c.comissoes.Resumo(c.ctx, portscomissao.ResumoFiltro{Competencia: c.anterior, EmpresaIDs: []uuid.UUID{c.red}}); err != nil || len(daRed) != 0 {
		t.Fatalf("resumo da Red = %+v, %v", daRed, err)
	}
	if daBeatriz, err := c.comissoes.Resumo(c.ctx, portscomissao.ResumoFiltro{Competencia: c.anterior, RepresentanteID: &c.beatriz.ID}); err != nil || len(daBeatriz) != 0 {
		t.Fatalf("resumo da Beatriz = %+v, %v", daBeatriz, err)
	}
}
