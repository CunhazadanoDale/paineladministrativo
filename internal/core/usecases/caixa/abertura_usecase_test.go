package caixa_test

import (
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
	"github.com/google/uuid"
)

func TestDefinirAberturaCriaEAlteraComVersao(t *testing.T) {
	c := novoCenario(t)

	abertura, err := c.aberturas.Definir(c.ctx, portsin.DefinirAberturaInput{
		UsuarioID: c.financeiroMIA, ContaBancariaID: c.sicrediMIA, DataCorte: dia(-31), SaldoCentavos: 150000,
	})
	if err != nil {
		t.Fatalf("Definir = %v", err)
	}
	if !abertura.DataCorte.Equal(dia(-31)) || abertura.SaldoCentavos != 150000 || abertura.Versao != 1 || abertura.UsuarioID != c.financeiroMIA {
		t.Errorf("abertura criada = %+v", abertura)
	}

	_, err = c.aberturas.Definir(c.ctx, portsin.DefinirAberturaInput{
		UsuarioID: c.financeiroMIA, ContaBancariaID: c.sicrediMIA, DataCorte: dia(-10), SaldoCentavos: 1,
	})
	esperarErro(t, err, domain.ErrConflito, "envie a versão para alterar")

	_, err = c.aberturas.Definir(c.ctx, portsin.DefinirAberturaInput{
		UsuarioID: c.financeiroMIA, ContaBancariaID: c.sicrediMIA, DataCorte: dia(-10), SaldoCentavos: 1, Versao: inteiro(2),
	})
	esperarErro(t, err, domain.ErrConflito, "alterado por outra pessoa")

	alterada, err := c.aberturas.Definir(c.ctx, portsin.DefinirAberturaInput{
		UsuarioID: c.administrador, ContaBancariaID: c.sicrediMIA, DataCorte: dia(-10), SaldoCentavos: -500, Versao: inteiro(1),
	})
	if err != nil {
		t.Fatalf("Definir com versão = %v", err)
	}
	if !alterada.DataCorte.Equal(dia(-10)) || alterada.SaldoCentavos != -500 || alterada.Versao != 2 || alterada.UsuarioID != c.administrador {
		t.Errorf("abertura alterada = %+v", alterada)
	}

	obtida, err := c.aberturas.Obter(c.ctx, c.sicrediMIA, c.financeiroMIA)
	if err != nil || obtida.Versao != 2 {
		t.Errorf("Obter = %+v, %v", obtida, err)
	}

	_, err = c.aberturas.Definir(c.ctx, portsin.DefinirAberturaInput{
		UsuarioID: c.financeiroMIA, ContaBancariaID: c.sicrediMIA, DataCorte: dia(0), SaldoCentavos: 1, Versao: inteiro(2),
	})
	esperarErro(t, err, domain.ErrValidacao, "anterior a hoje")
}

func TestDefinirAberturaRespeitaPerfilEmpresaESituacao(t *testing.T) {
	c := novoCenario(t)
	input := func(usuarioID, contaID uuid.UUID) portsin.DefinirAberturaInput {
		return portsin.DefinirAberturaInput{UsuarioID: usuarioID, ContaBancariaID: contaID, DataCorte: dia(-5), SaldoCentavos: 10}
	}

	_, err := c.aberturas.Definir(c.ctx, input(c.comum, c.sicrediMIA))
	esperarErro(t, err, domain.ErrPermissao, "perfil sem permissão para o financeiro")

	_, err = c.aberturas.Definir(c.ctx, input(c.financeiroRed, c.sicrediMIA))
	esperarErro(t, err, domain.ErrPermissao, "perfil sem acesso a esta empresa")

	_, err = c.aberturas.Definir(c.ctx, input(c.financeiroMIA, uuid.New()))
	esperarErro(t, err, domain.ErrNotFound, "conta bancária não encontrada")

	_, err = c.aberturas.Definir(c.ctx, input(c.financeiroMIA, uuid.Nil))
	esperarErro(t, err, domain.ErrValidacao, "conta bancária é obrigatória")

	_, err = c.aberturas.Definir(c.ctx, input(c.financeiroMIA, c.contaInativa))
	esperarErro(t, err, domain.ErrValidacao, "conta bancária inativa")

	_, err = c.aberturas.Definir(c.ctx, input(c.financeiroMIA, c.contaEmpresaInativa))
	esperarErro(t, err, domain.ErrValidacao, "empresa inativa")

	_, err = c.aberturas.Obter(c.ctx, c.itauRed, c.financeiroRed)
	esperarErro(t, err, domain.ErrNotFound, "conta sem saldo inicial")

	_, err = c.aberturas.Obter(c.ctx, c.itauRed, c.financeiroMIA)
	esperarErro(t, err, domain.ErrPermissao, "perfil sem acesso a esta empresa")
}
