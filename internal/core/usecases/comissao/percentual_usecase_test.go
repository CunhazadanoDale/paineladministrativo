package comissao_test

import (
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/comissao"
	"github.com/google/uuid"
)

func TestObterPercentualSugereOPadraoDoRepresentanteSemGravar(t *testing.T) {
	c := novoCenario(t)

	percentual, err := c.percentuais.Obter(c.ctx, c.pedidoDoCarlos.ID, c.comercial)
	if err != nil {
		t.Fatalf("Obter: %v", err)
	}
	if percentual.Gravado() || percentual.Origem != domaincomissao.OrigemPercentualPadrao || percentual.Percentual.PontosBase() != 250 {
		t.Fatalf("percentual = %+v", percentual)
	}
	if len(c.m.percentuais) != 0 {
		t.Fatalf("a sugestão não deveria ser gravada: %+v", c.m.percentuais)
	}

	importacao, err := c.percentuais.Obter(c.ctx, c.pedidoDaRed.ID, c.administrador)
	if err != nil || importacao.Percentual.PontosBase() != 500 {
		t.Fatalf("importação = %+v, %v", importacao, err)
	}
}

func TestDefinirPercentualCriaEDepoisExigeVersao(t *testing.T) {
	c := novoCenario(t)

	definido, err := c.percentuais.Definir(c.ctx, portsin.DefinirPercentualInput{
		UsuarioID: c.comercial, PedidoID: c.pedidoDoCarlos.ID, PontosBase: 320,
	})
	if err != nil {
		t.Fatalf("Definir: %v", err)
	}
	if definido.Origem != domaincomissao.OrigemPercentualManual || definido.Versao != 1 || c.m.versaoLida != nil {
		t.Fatalf("definido = %+v, versão lida %v", definido, c.m.versaoLida)
	}

	_, err = c.percentuais.Definir(c.ctx, portsin.DefinirPercentualInput{
		UsuarioID: c.comercial, PedidoID: c.pedidoDoCarlos.ID, PontosBase: 350,
	})
	esperarErro(t, err, domain.ErrValidacao, "versão da comissão do pedido é obrigatória")

	_, err = c.percentuais.Definir(c.ctx, portsin.DefinirPercentualInput{
		UsuarioID: c.comercial, PedidoID: c.pedidoDoCarlos.ID, PontosBase: 350, Versao: versao(7),
	})
	esperarErro(t, err, domain.ErrConflito, "recarregue")

	redefinido, err := c.percentuais.Definir(c.ctx, portsin.DefinirPercentualInput{
		UsuarioID: c.administrador, PedidoID: c.pedidoDoCarlos.ID, PontosBase: 350, Versao: versao(1),
	})
	if err != nil {
		t.Fatalf("Definir de novo: %v", err)
	}
	if redefinido.Versao != 2 || redefinido.Percentual.PontosBase() != 350 || *c.m.versaoLida != 1 {
		t.Fatalf("redefinido = %+v", redefinido)
	}

	lido, err := c.percentuais.Obter(c.ctx, c.pedidoDoCarlos.ID, c.usuarioCarlos)
	if err != nil || !lido.Gravado() || lido.Percentual.PontosBase() != 350 {
		t.Fatalf("lido pelo representante = %+v, %v", lido, err)
	}
}

func TestDefinirPercentualDeOutroRepresentanteSubstituiALinhaAntiga(t *testing.T) {
	c := novoCenario(t)
	antigo, err := domaincomissao.NovoPercentualManual(c.pedidoDoCarlos.ID, c.beatriz.ID, c.comercial, 900, data(2026, 10, 1))
	if err != nil {
		t.Fatalf("NovoPercentualManual: %v", err)
	}
	c.m.percentuais[c.pedidoDoCarlos.ID] = antigo

	definido, err := c.percentuais.Definir(c.ctx, portsin.DefinirPercentualInput{
		UsuarioID: c.comercial, PedidoID: c.pedidoDoCarlos.ID, PontosBase: 200,
	})
	if err != nil {
		t.Fatalf("Definir: %v", err)
	}
	if definido.RepresentanteID != c.carlos.ID || c.m.versaoLida != nil {
		t.Fatalf("definido = %+v", definido)
	}
}

func TestDefinirPercentualRecusaPedidosSemComissaoEditavel(t *testing.T) {
	c := novoCenario(t)
	semRepresentante := c.novoPedido(c.mia, nil, domainpedido.ModeloProntaEntrega, domainpedido.StatusProposta)
	cancelado := c.novoPedido(c.mia, &c.carlos.ID, domainpedido.ModeloProntaEntrega, domainpedido.StatusCancelado)
	comSaida := c.novoPedido(c.mia, &c.carlos.ID, domainpedido.ModeloProntaEntrega, domainpedido.StatusParcialmenteEntregue)
	c.novaEntrega(comSaida, data(2026, 10, 5), domaincomissao.ItemSaido{Quantidade: 1, PrecoUnitarioCentavos: 100})

	casos := []struct {
		nome      string
		pedidoID  uuid.UUID
		sentinela error
		trecho    string
	}{
		{"inexistente", uuid.New(), domain.ErrNotFound, "pedido não encontrado"},
		{"sem representante", semRepresentante.ID, domain.ErrValidacao, "sem representante"},
		{"cancelado", cancelado.ID, domain.ErrConflito, "cancelado"},
		{"depois da saída", comSaida.ID, domain.ErrConflito, "primeira saída"},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			_, err := c.percentuais.Definir(c.ctx, portsin.DefinirPercentualInput{
				UsuarioID: c.comercial, PedidoID: caso.pedidoID, PontosBase: 100,
			})
			esperarErro(t, err, caso.sentinela, caso.trecho)
		})
	}

	_, err := c.percentuais.Definir(c.ctx, portsin.DefinirPercentualInput{
		UsuarioID: c.comercial, PedidoID: c.pedidoDoCarlos.ID, PontosBase: 10001,
	})
	esperarErro(t, err, domain.ErrValidacao, "10000 pontos-base")
}

func TestPermissoesDoPercentual(t *testing.T) {
	c := novoCenario(t)
	financeiroRed := c.novoUsuario(false, true, false, acessoA(c.red))

	_, err := c.percentuais.Definir(c.ctx, portsin.DefinirPercentualInput{
		UsuarioID: c.usuarioCarlos, PedidoID: c.pedidoDoCarlos.ID, PontosBase: 900,
	})
	esperarErro(t, err, domain.ErrPermissao, "representante não define a própria comissão")

	_, err = c.percentuais.Definir(c.ctx, portsin.DefinirPercentualInput{
		UsuarioID: c.financeiro, PedidoID: c.pedidoDoCarlos.ID, PontosBase: 300,
	})
	esperarErro(t, err, domain.ErrPermissao, "definir a comissão")

	_, err = c.percentuais.Obter(c.ctx, c.pedidoDoCarlos.ID, c.semPerfil)
	esperarErro(t, err, domain.ErrPermissao, "ver a comissão")

	_, err = c.percentuais.Obter(c.ctx, c.pedidoDaRed.ID, c.usuarioCarlos)
	esperarErro(t, err, domain.ErrPermissao, "outro representante")

	_, err = c.percentuais.Obter(c.ctx, c.pedidoDoCarlos.ID, financeiroRed)
	esperarErro(t, err, domain.ErrPermissao, "sem acesso a esta empresa")

	if _, err := c.percentuais.Obter(c.ctx, c.pedidoDoCarlos.ID, c.financeiro); err != nil {
		t.Fatalf("financeiro da empresa deveria ler: %v", err)
	}
}
