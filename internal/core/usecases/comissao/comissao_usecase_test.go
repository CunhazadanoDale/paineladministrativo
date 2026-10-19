package comissao_test

import (
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/comissao"
	"github.com/google/uuid"
)

func TestApurarGravaUmaComissaoPorEntregaEEhIdempotente(t *testing.T) {
	c := novoCenario(t)
	primeira := c.novaEntrega(c.pedidoDoCarlos, data(2026, 9, 28), domaincomissao.ItemSaido{Quantidade: 10, PrecoUnitarioCentavos: 1500})
	c.novaEntrega(c.pedidoDoCarlos, data(2026, 10, 2), domaincomissao.ItemSaido{Quantidade: 2, PrecoUnitarioCentavos: 1500})

	resultado, err := c.comissoes.Apurar(c.ctx, portsin.ApurarInput{UsuarioID: c.financeiro})
	if err != nil {
		t.Fatalf("Apurar: %v", err)
	}
	if len(resultado.Apuradas) != 2 || resultado.JaExistentes != 0 || len(resultado.Falhas) != 0 || resultado.RestamEntregas {
		t.Fatalf("resultado = %+v", resultado)
	}
	if c.m.comissoes[0].EntregaID != primeira.EntregaID || c.m.comissoes[0].ValorCentavos != 375 || c.m.comissoes[0].Competencia.String() != "2026-09" {
		t.Fatalf("primeira comissão = %+v", c.m.comissoes[0])
	}
	if percentual := c.m.percentuais[c.pedidoDoCarlos.ID]; percentual == nil || percentual.Origem != domaincomissao.OrigemPercentualPadrao {
		t.Fatalf("o padrão deveria ser gravado na primeira apuração: %+v", percentual)
	}
	if c.m.ultimoFiltroEntregas.Size != 501 || c.m.ultimoFiltroEntregas.EmpresaIDs[0] != c.mia || c.m.ultimoFiltroEntregas.SaidaAntesDe != nil {
		t.Fatalf("filtro = %+v", c.m.ultimoFiltroEntregas)
	}

	deNovo, err := c.comissoes.Apurar(c.ctx, portsin.ApurarInput{UsuarioID: c.financeiro})
	if err != nil || len(deNovo.Apuradas) != 0 || len(c.m.comissoes) != 2 {
		t.Fatalf("segunda apuração = %+v, %v", deNovo, err)
	}
}

func TestApurarUsaOPercentualManualDoPedido(t *testing.T) {
	c := novoCenario(t)
	if _, err := c.percentuais.Definir(c.ctx, portsin.DefinirPercentualInput{
		UsuarioID: c.comercial, PedidoID: c.pedidoDoCarlos.ID, PontosBase: 1000,
	}); err != nil {
		t.Fatalf("Definir: %v", err)
	}
	c.novaEntrega(c.pedidoDoCarlos, data(2026, 10, 2), domaincomissao.ItemSaido{Quantidade: 3, PrecoUnitarioCentavos: 1000})

	if _, err := c.comissoes.Apurar(c.ctx, portsin.ApurarInput{UsuarioID: c.administrador}); err != nil {
		t.Fatalf("Apurar: %v", err)
	}
	if c.m.comissoes[0].ValorCentavos != 300 || c.m.comissoes[0].Percentual.PontosBase() != 1000 {
		t.Fatalf("comissão = %+v", c.m.comissoes[0])
	}
}

func TestApurarListaFalhasDeNegocioESegue(t *testing.T) {
	c := novoCenario(t)
	semRepresentanteCadastrado := c.novoPedido(c.mia, ptr(uuid.New()), domainpedido.ModeloProntaEntrega, domainpedido.StatusConfirmado)
	falha := c.novaEntrega(semRepresentanteCadastrado, data(2026, 10, 1), domaincomissao.ItemSaido{Quantidade: 1, PrecoUnitarioCentavos: 100})
	conflito := c.novaEntrega(c.pedidoDoCarlos, data(2026, 10, 1), domaincomissao.ItemSaido{Quantidade: 1, PrecoUnitarioCentavos: 100})
	c.m.falhaNaGravacao[conflito.EntregaID] = domaincomissao.ErroPercentualMudouNaApuracao()
	c.novaEntrega(c.pedidoDoCarlos, data(2026, 10, 2), domaincomissao.ItemSaido{Quantidade: 1, PrecoUnitarioCentavos: 100})

	resultado, err := c.comissoes.Apurar(c.ctx, portsin.ApurarInput{UsuarioID: c.administrador})
	if err != nil {
		t.Fatalf("Apurar: %v", err)
	}
	if len(resultado.Apuradas) != 1 || len(resultado.Falhas) != 2 {
		t.Fatalf("resultado = %+v", resultado)
	}
	if resultado.Falhas[0].EntregaID != falha.EntregaID || resultado.Falhas[1].EntregaID != conflito.EntregaID {
		t.Fatalf("falhas = %+v", resultado.Falhas)
	}
}

func TestApurarDevolveErroDeInfraestrutura(t *testing.T) {
	c := novoCenario(t)
	entrega := c.novaEntrega(c.pedidoDoCarlos, data(2026, 10, 1), domaincomissao.ItemSaido{Quantidade: 1, PrecoUnitarioCentavos: 100})
	c.m.falhaNaGravacao[entrega.EntregaID] = errBancoFora

	if _, err := c.comissoes.Apurar(c.ctx, portsin.ApurarInput{UsuarioID: c.administrador}); err != errBancoFora {
		t.Fatalf("erro = %v", err)
	}
}

func TestApurarAteACompetenciaFiltraPelaSaida(t *testing.T) {
	c := novoCenario(t)

	if _, err := c.comissoes.Apurar(c.ctx, portsin.ApurarInput{UsuarioID: c.financeiro, Empresa: &c.mia, Competencia: "2026-09"}); err != nil {
		t.Fatalf("Apurar: %v", err)
	}
	filtro := c.m.ultimoFiltroEntregas
	if filtro.SaidaAntesDe == nil || !filtro.SaidaAntesDe.Equal(data(2026, 10, 1)) || *filtro.Empresa != c.mia {
		t.Fatalf("filtro = %+v", filtro)
	}

	_, err := c.comissoes.Apurar(c.ctx, portsin.ApurarInput{UsuarioID: c.financeiro, Competencia: "09/2026"})
	esperarErro(t, err, domain.ErrValidacao, "AAAA-MM")

	_, err = c.comissoes.Apurar(c.ctx, portsin.ApurarInput{UsuarioID: c.financeiro, Empresa: &c.red})
	esperarErro(t, err, domain.ErrPermissao, "sem acesso a esta empresa")

	_, err = c.comissoes.Apurar(c.ctx, portsin.ApurarInput{UsuarioID: c.comercial})
	esperarErro(t, err, domain.ErrPermissao, "apurar comissões")
}

func TestPendenciasNormalizaAPaginacao(t *testing.T) {
	c := novoCenario(t)
	c.novaEntrega(c.pedidoDoCarlos, data(2026, 10, 1), domaincomissao.ItemSaido{Quantidade: 1, PrecoUnitarioCentavos: 100})

	pendentes, err := c.comissoes.Pendencias(c.ctx, portsin.PendenciasInput{
		UsuarioID: c.financeiro, Filtro: domain.PaginacaoFiltro{Page: 0, Size: 500},
	})
	if err != nil || len(pendentes) != 1 {
		t.Fatalf("pendências = %+v, %v", pendentes, err)
	}
	if c.m.ultimoFiltroEntregas.Page != 1 || c.m.ultimoFiltroEntregas.Size != domain.TamanhoPaginaMaximo {
		t.Fatalf("paginação = %+v", c.m.ultimoFiltroEntregas.PaginacaoFiltro)
	}

	_, err = c.comissoes.Pendencias(c.ctx, portsin.PendenciasInput{UsuarioID: c.usuarioCarlos})
	esperarErro(t, err, domain.ErrPermissao, "apurar comissões")
}

func TestListarComissoesRestringeORepresentanteAsProprias(t *testing.T) {
	c := novoCenario(t)

	if _, err := c.comissoes.Listar(c.ctx, portsin.ListarComissoesInput{UsuarioID: c.usuarioCarlos, Competencia: "2026-10"}); err != nil {
		t.Fatalf("Listar: %v", err)
	}
	filtro := c.m.ultimoFiltro
	if filtro.RepresentanteID == nil || *filtro.RepresentanteID != c.carlos.ID || filtro.Competencia.String() != "2026-10" {
		t.Fatalf("filtro do representante = %+v", filtro)
	}

	_, err := c.comissoes.Listar(c.ctx, portsin.ListarComissoesInput{UsuarioID: c.usuarioCarlos, RepresentanteID: &c.beatriz.ID})
	esperarErro(t, err, domain.ErrPermissao, "outro representante")

	if _, err := c.comissoes.Listar(c.ctx, portsin.ListarComissoesInput{UsuarioID: c.financeiro, RepresentanteID: &c.beatriz.ID}); err != nil {
		t.Fatalf("Listar do financeiro: %v", err)
	}
	if *c.m.ultimoFiltro.RepresentanteID != c.beatriz.ID || c.m.ultimoFiltro.EmpresaIDs[0] != c.mia {
		t.Fatalf("filtro do financeiro = %+v", c.m.ultimoFiltro)
	}

	_, err = c.comissoes.Listar(c.ctx, portsin.ListarComissoesInput{UsuarioID: c.comercial})
	esperarErro(t, err, domain.ErrPermissao, "ver comissões")
}

func ptr(id uuid.UUID) *uuid.UUID {
	return &id
}
