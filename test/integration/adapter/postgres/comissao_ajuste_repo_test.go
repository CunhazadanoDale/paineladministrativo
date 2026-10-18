package postgres_test

import (
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	portscomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/comissao"
	"github.com/google/uuid"
)

func (c *cenarioBancoComissoes) novoAjuste(t *testing.T, dados domaincomissao.DadosAjuste) *domaincomissao.Ajuste {
	t.Helper()

	ajuste, err := domaincomissao.NovoAjuste(dados, c.usuarioID, time.Now().UTC())
	if err != nil {
		t.Fatalf("NovoAjuste: %v", err)
	}

	return ajuste
}

func (c *cenarioBancoComissoes) ajustesListados(t *testing.T, filtro portscomissao.AjusteFiltro) []*domaincomissao.Ajuste {
	t.Helper()

	ajustes, err := c.comissoes.ListarAjustes(c.ctx, filtro)
	if err != nil {
		t.Fatalf("ListarAjustes: %v", err)
	}

	return ajustes
}

func TestComissaoRepositoryCriaEListaAjustes(t *testing.T) {
	c := novoCenarioComissoes(t)
	pedido := c.pedidoConfirmado(t, c.carlos)
	competencia := domaincomissao.CompetenciaDe(domaincomissao.Hoje(time.Now().UTC()))

	doPedido := c.novoAjuste(t, domaincomissao.DadosAjuste{
		EmpresaID: c.mia, RepresentanteID: c.carlos.ID, PedidoID: &pedido.ID,
		Competencia: competencia.String(), ValorCentavos: -375, Motivo: "devolução de uma peça",
	})
	if err := c.comissoes.CriarAjuste(c.ctx, doPedido); err != nil {
		t.Fatalf("CriarAjuste: %v", err)
	}
	avulso := c.novoAjuste(t, domaincomissao.DadosAjuste{
		EmpresaID: c.red, RepresentanteID: c.beatriz.ID,
		Competencia: competencia.String(), ValorCentavos: 10000, Motivo: "prêmio de meta",
	})
	if err := c.comissoes.CriarAjuste(c.ctx, avulso); err != nil {
		t.Fatalf("CriarAjuste avulso: %v", err)
	}

	doCarlos := c.ajustesListados(t, portscomissao.AjusteFiltro{RepresentanteID: &c.carlos.ID, Competencia: &competencia})
	if len(doCarlos) != 1 || doCarlos[0].ID != doPedido.ID || doCarlos[0].ValorCentavos != -375 || doCarlos[0].PedidoID == nil ||
		*doCarlos[0].PedidoID != pedido.ID || doCarlos[0].Motivo != "devolução de uma peça" || doCarlos[0].Competencia != competencia {
		t.Fatalf("ajustes do Carlos = %+v", doCarlos)
	}
	if daRed := c.ajustesListados(t, portscomissao.AjusteFiltro{EmpresaIDs: []uuid.UUID{c.red}}); len(daRed) != 1 || daRed[0].PedidoID != nil {
		t.Fatalf("ajustes da Red = %+v", daRed)
	}
	if doPedidoListado := c.ajustesListados(t, portscomissao.AjusteFiltro{PedidoID: &pedido.ID, Empresa: &c.mia}); len(doPedidoListado) != 1 {
		t.Fatalf("ajustes do pedido = %+v", doPedidoListado)
	}
	if todos := c.ajustesListados(t, portscomissao.AjusteFiltro{}); len(todos) != 2 {
		t.Fatalf("todos os ajustes = %+v", todos)
	}
}

func TestComissaoRepositoryRecusaAjusteComPedidoDeOutroRepresentante(t *testing.T) {
	c := novoCenarioComissoes(t)
	pedido := c.pedidoConfirmado(t, c.carlos)
	competencia := domaincomissao.CompetenciaDe(domaincomissao.Hoje(time.Now().UTC())).String()

	daBeatriz := c.novoAjuste(t, domaincomissao.DadosAjuste{
		EmpresaID: c.mia, RepresentanteID: c.beatriz.ID, PedidoID: &pedido.ID,
		Competencia: competencia, ValorCentavos: 100, Motivo: "teste",
	})
	esperarErro(t, c.comissoes.CriarAjuste(c.ctx, daBeatriz), domain.ErrValidacao, "mesmo representante")

	inexistente := uuid.New()
	semPedido := c.novoAjuste(t, domaincomissao.DadosAjuste{
		EmpresaID: c.mia, RepresentanteID: c.carlos.ID, PedidoID: &inexistente,
		Competencia: competencia, ValorCentavos: 100, Motivo: "teste",
	})
	esperarErro(t, c.comissoes.CriarAjuste(c.ctx, semPedido), domain.ErrValidacao, "pedido do ajuste não encontrado")

	semRepresentante := c.novoAjuste(t, domaincomissao.DadosAjuste{
		EmpresaID: c.mia, RepresentanteID: uuid.New(), Competencia: competencia, ValorCentavos: 100, Motivo: "teste",
	})
	esperarErro(t, c.comissoes.CriarAjuste(c.ctx, semRepresentante), domain.ErrValidacao, "registro relacionado não encontrado")

	if todos := c.ajustesListados(t, portscomissao.AjusteFiltro{}); len(todos) != 0 {
		t.Fatalf("nenhum ajuste deveria ser gravado: %+v", todos)
	}
}
