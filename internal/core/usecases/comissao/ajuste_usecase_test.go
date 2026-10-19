package comissao_test

import (
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/comissao"
	"github.com/google/uuid"
)

func (c *cenario) dadosDeAjuste() domaincomissao.DadosAjuste {
	return domaincomissao.DadosAjuste{
		EmpresaID:       c.mia,
		RepresentanteID: c.carlos.ID,
		PedidoID:        &c.pedidoDoCarlos.ID,
		Competencia:     domaincomissao.CompetenciaDe(domaincomissao.Hoje(time.Now().UTC())).String(),
		ValorCentavos:   -375,
		Motivo:          "devolução de uma peça",
	}
}

func TestCriarAjusteGravaComOPedidoDoRepresentante(t *testing.T) {
	c := novoCenario(t)

	ajuste, err := c.ajustes.Criar(c.ctx, portsin.CriarAjusteInput{UsuarioID: c.financeiro, Dados: c.dadosDeAjuste()})
	if err != nil {
		t.Fatalf("Criar: %v", err)
	}
	if len(c.m.ajustes) != 1 || c.m.ajustes[0] != ajuste || ajuste.ValorCentavos != -375 || ajuste.UsuarioID != c.financeiro {
		t.Fatalf("ajuste = %+v", ajuste)
	}

	dados := c.dadosDeAjuste()
	dados.PedidoID = nil
	dados.RepresentanteID = c.beatriz.ID
	if _, err := c.ajustes.Criar(c.ctx, portsin.CriarAjusteInput{UsuarioID: c.administrador, Dados: dados}); err != nil {
		t.Fatalf("ajuste sem pedido: %v", err)
	}
}

func TestCriarAjusteRecusaReferenciasInvalidas(t *testing.T) {
	c := novoCenario(t)
	semRepresentante := c.novoPedido(c.mia, nil, domainpedido.ModeloProntaEntrega, domainpedido.StatusConfirmado)
	inexistente := uuid.New()

	casos := []struct {
		nome      string
		usuario   uuid.UUID
		ajustar   func(*domaincomissao.DadosAjuste)
		sentinela error
		trecho    string
	}{
		{"comercial", c.comercial, func(*domaincomissao.DadosAjuste) {}, domain.ErrPermissao, "apurar comissões"},
		{"outra empresa", c.financeiro, func(d *domaincomissao.DadosAjuste) { d.EmpresaID = c.red; d.PedidoID = nil }, domain.ErrPermissao, "sem acesso a esta empresa"},
		{"representante inexistente", c.financeiro, func(d *domaincomissao.DadosAjuste) { d.RepresentanteID = uuid.New(); d.PedidoID = nil }, domain.ErrValidacao, "representante do ajuste não encontrado"},
		{"pedido inexistente", c.financeiro, func(d *domaincomissao.DadosAjuste) { d.PedidoID = &inexistente }, domain.ErrValidacao, "pedido do ajuste não encontrado"},
		{"pedido de outro representante", c.financeiro, func(d *domaincomissao.DadosAjuste) { d.RepresentanteID = c.beatriz.ID }, domain.ErrValidacao, "mesmo representante"},
		{"pedido de outra empresa", c.administrador, func(d *domaincomissao.DadosAjuste) { d.PedidoID = &c.pedidoDaRed.ID; d.RepresentanteID = c.beatriz.ID }, domain.ErrValidacao, "mesma empresa"},
		{"pedido sem representante", c.financeiro, func(d *domaincomissao.DadosAjuste) { d.PedidoID = &semRepresentante.ID }, domain.ErrValidacao, "mesmo representante"},
		{"valor zero", c.financeiro, func(d *domaincomissao.DadosAjuste) { d.ValorCentavos = 0 }, domain.ErrValidacao, "não pode ser zero"},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			dados := c.dadosDeAjuste()
			caso.ajustar(&dados)

			_, err := c.ajustes.Criar(c.ctx, portsin.CriarAjusteInput{UsuarioID: caso.usuario, Dados: dados})
			esperarErro(t, err, caso.sentinela, caso.trecho)
		})
	}
	if len(c.m.ajustes) != 0 {
		t.Fatalf("nenhum ajuste deveria ser gravado: %+v", c.m.ajustes)
	}
}

func TestListarAjustesRestringeORepresentanteAosProprios(t *testing.T) {
	c := novoCenario(t)

	if _, err := c.ajustes.Listar(c.ctx, portsin.ListarAjustesInput{UsuarioID: c.usuarioCarlos, Competencia: "2026-10"}); err != nil {
		t.Fatalf("Listar: %v", err)
	}
	filtro := c.m.ultimoFiltroAjustes
	if filtro.RepresentanteID == nil || *filtro.RepresentanteID != c.carlos.ID || filtro.Competencia.String() != "2026-10" {
		t.Fatalf("filtro do representante = %+v", filtro)
	}

	_, err := c.ajustes.Listar(c.ctx, portsin.ListarAjustesInput{UsuarioID: c.usuarioCarlos, RepresentanteID: &c.beatriz.ID})
	esperarErro(t, err, domain.ErrPermissao, "outro representante")

	if _, err := c.ajustes.Listar(c.ctx, portsin.ListarAjustesInput{UsuarioID: c.financeiro, PedidoID: &c.pedidoDoCarlos.ID}); err != nil {
		t.Fatalf("Listar do financeiro: %v", err)
	}
	if c.m.ultimoFiltroAjustes.EmpresaIDs[0] != c.mia || *c.m.ultimoFiltroAjustes.PedidoID != c.pedidoDoCarlos.ID {
		t.Fatalf("filtro do financeiro = %+v", c.m.ultimoFiltroAjustes)
	}

	_, err = c.ajustes.Listar(c.ctx, portsin.ListarAjustesInput{UsuarioID: c.comercial})
	esperarErro(t, err, domain.ErrPermissao, "ver comissões")

	_, err = c.ajustes.Listar(c.ctx, portsin.ListarAjustesInput{UsuarioID: c.financeiro, Competencia: "x"})
	esperarErro(t, err, domain.ErrValidacao, "AAAA-MM")
}

func TestFinanceiroQueTambemERepresentanteNaoMexeNaPropriaComissao(t *testing.T) {
	c := novoCenario(t)
	financeiroRepresentante := c.novoUsuario(false, true, false, acessoA(c.mia))
	c.beatriz.UsuarioID = &financeiroRepresentante

	dados := c.dadosDeAjuste()
	dados.RepresentanteID = c.beatriz.ID
	dados.PedidoID = nil
	_, err := c.ajustes.Criar(c.ctx, portsin.CriarAjusteInput{UsuarioID: financeiroRepresentante, Dados: dados})
	esperarErro(t, err, domain.ErrPermissao, "representante não apura, ajusta nem fecha")

	_, err = c.comissoes.Apurar(c.ctx, portsin.ApurarInput{UsuarioID: financeiroRepresentante})
	esperarErro(t, err, domain.ErrPermissao, "representante não apura, ajusta nem fecha")

	if _, err := c.comissoes.Listar(c.ctx, portsin.ListarComissoesInput{UsuarioID: financeiroRepresentante}); err != nil {
		t.Fatalf("Listar: %v", err)
	}
	if filtro := c.m.ultimoFiltro; filtro.RepresentanteID == nil || *filtro.RepresentanteID != c.beatriz.ID {
		t.Fatalf("o financeiro que é representante deveria ver só as próprias: %+v", filtro)
	}
}
