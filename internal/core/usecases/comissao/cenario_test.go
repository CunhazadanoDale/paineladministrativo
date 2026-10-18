package comissao_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domaincomissao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/comissao"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/comissao"
	comissaousecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/comissao"
	"github.com/google/uuid"
)

type cenario struct {
	ctx         context.Context
	m           *memoria
	percentuais portsin.PercentualUseCase
	comissoes   portsin.ComissaoUseCase
	ajustes     portsin.AjusteUseCase

	mia uuid.UUID
	red uuid.UUID

	administrador  uuid.UUID
	financeiro     uuid.UUID
	comercial      uuid.UUID
	semPerfil      uuid.UUID
	usuarioCarlos  uuid.UUID
	carlos         *domaincliente.Representante
	beatriz        *domaincliente.Representante
	pedidoDoCarlos *domainpedido.Pedido
	pedidoDaRed    *domainpedido.Pedido
}

func novoCenario(t *testing.T) *cenario {
	t.Helper()

	m := novaMemoria()
	c := &cenario{
		ctx: context.Background(),
		m:   m,
		percentuais: comissaousecases.NewPercentualUsecase(
			repoComissoes{m}, repoPedidos{m}, repoRepresentantes{m}, repoEmpresas{m}, repoUsuarios{m}, repoCargos{m},
		),
		comissoes: comissaousecases.NewComissaoUsecase(
			repoComissoes{m}, repoRepresentantes{m}, repoEmpresas{m}, repoUsuarios{m}, repoCargos{m},
		),
		ajustes: comissaousecases.NewAjusteUsecase(
			repoComissoes{m}, repoPedidos{m}, repoRepresentantes{m}, repoEmpresas{m}, repoUsuarios{m}, repoCargos{m},
		),
		mia: uuid.New(),
		red: uuid.New(),
	}

	c.administrador = c.novoUsuario(true, false, false, domainempresa.AcessoTotal())
	c.financeiro = c.novoUsuario(false, true, false, acessoA(c.mia))
	c.comercial = c.novoUsuario(false, false, true, domainempresa.AcessoTotal())
	c.semPerfil = c.novoUsuario(false, false, false, domainempresa.AcessoTotal())
	c.usuarioCarlos = c.novoUsuario(false, false, true, domainempresa.AcessoTotal())

	c.carlos = c.novoRepresentante("Carlos", &c.usuarioCarlos, 250, 400)
	c.beatriz = c.novoRepresentante("Beatriz", nil, 300, 500)
	c.pedidoDoCarlos = c.novoPedido(c.mia, &c.carlos.ID, domainpedido.ModeloProntaEntrega, domainpedido.StatusConfirmado)
	c.pedidoDaRed = c.novoPedido(c.red, &c.beatriz.ID, domainpedido.ModeloImportacao, domainpedido.StatusConfirmado)

	return c
}

func (c *cenario) novoUsuario(administrador, financeiro, comercial bool, acesso domainempresa.Acesso) uuid.UUID {
	cargo := &domainusuarios.Cargo{
		ID: uuid.New(), Nome: "Cargo", Ativo: true,
		Administrador: administrador, Financeiro: financeiro, Comercial: comercial,
	}
	usuario := &domainusuarios.Usuario{ID: uuid.New(), Nome: "Usuário", CargoID: cargo.ID, Ativo: true}
	c.m.cargos[cargo.ID] = cargo
	c.m.usuarios[usuario.ID] = usuario
	c.m.acessos[usuario.ID] = acesso

	return usuario.ID
}

func (c *cenario) novoRepresentante(nome string, usuarioID *uuid.UUID, prontaEntrega, importacao int) *domaincliente.Representante {
	representante := &domaincliente.Representante{
		ID:                    uuid.New(),
		Nome:                  nome,
		UsuarioID:             usuarioID,
		ComissaoProntaEntrega: domaincliente.PercentualDe(prontaEntrega),
		ComissaoImportacao:    domaincliente.PercentualDe(importacao),
		Ativo:                 true,
	}
	c.m.representantes[representante.ID] = representante

	return representante
}

func (c *cenario) novoPedido(empresaID uuid.UUID, representanteID *uuid.UUID, modelo domainpedido.Modelo, status domainpedido.Status) *domainpedido.Pedido {
	pedido := &domainpedido.Pedido{
		ID:              uuid.New(),
		EmpresaID:       empresaID,
		Numero:          len(c.m.pedidos) + 1,
		RepresentanteID: representanteID,
		Modelo:          modelo,
		Status:          status,
		Versao:          1,
	}
	c.m.pedidos[pedido.ID] = pedido

	return pedido
}

func (c *cenario) novaEntrega(pedido *domainpedido.Pedido, saiuEm time.Time, itens ...domaincomissao.ItemSaido) *domaincomissao.EntregaParaApurar {
	entrega := &domaincomissao.EntregaParaApurar{
		EntregaID:       uuid.New(),
		PedidoID:        pedido.ID,
		PedidoNumero:    pedido.Numero,
		EmpresaID:       pedido.EmpresaID,
		RepresentanteID: *pedido.RepresentanteID,
		Modelo:          pedido.Modelo,
		SaiuEm:          saiuEm,
		Percentual:      c.m.percentuais[pedido.ID],
		Itens:           itens,
	}
	c.m.entregas = append(c.m.entregas, entrega)
	c.m.saidas[pedido.ID] = true

	return entrega
}

func data(ano int, mes time.Month, dia int) time.Time {
	return time.Date(ano, mes, dia, 0, 0, 0, 0, time.UTC)
}

func versao(valor int) *int {
	return &valor
}

func esperarErro(t *testing.T, err error, sentinela error, trecho string) {
	t.Helper()

	if !errors.Is(err, sentinela) || !strings.Contains(err.Error(), trecho) {
		t.Fatalf("erro = %v; esperado %v com %q", err, sentinela, trecho)
	}
}

func acessoA(empresas ...uuid.UUID) domainempresa.Acesso {
	return domainempresa.AcessoRestrito(empresas)
}
