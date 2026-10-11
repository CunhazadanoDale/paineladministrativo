package estoque_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	"github.com/google/uuid"
)

func centavosDe(valor int64) *int64 {
	return &valor
}

func TestMovimentarExigeLocalECustoNaEntrada(t *testing.T) {
	c := novoCenario(t)
	mia := c.novaEmpresa(true)
	red := c.novaEmpresa(true)
	galpao := c.novoLocal(t, mia, "Galpão")
	daRed := c.novoLocal(t, red, "Depósito")
	inativo := c.novoLocal(t, mia, "Antigo")
	if _, err := c.local.Desativar(c.ctx, inativo.ID, c.usuarioID); err != nil {
		t.Fatalf("desativar falhou: %v", err)
	}
	produto := c.produtoDeTeste(t)
	administradorDaRed := c.novoUsuarioComPerfil("Diretoria Red", true, false, red)

	casos := []struct {
		nome      string
		input     portsin.MovimentarEstoqueInput
		sentinela error
	}{
		{"sem local", portsin.MovimentarEstoqueInput{Tipo: "entrada", Quantidade: 1, CustoCentavos: centavosDe(100)}, domain.ErrValidacao},
		{"local inexistente", portsin.MovimentarEstoqueInput{LocalID: uuid.New(), Tipo: "entrada", Quantidade: 1, CustoCentavos: centavosDe(100)}, domain.ErrValidacao},
		{"entrada sem custo", portsin.MovimentarEstoqueInput{LocalID: galpao.ID, Tipo: "entrada", Quantidade: 1}, domain.ErrValidacao},
		{"custo negativo", portsin.MovimentarEstoqueInput{LocalID: galpao.ID, Tipo: "entrada", Quantidade: 1, CustoCentavos: centavosDe(-1)}, domain.ErrValidacao},
		{"transferência pelo movimento", portsin.MovimentarEstoqueInput{LocalID: galpao.ID, Tipo: "transferencia", Quantidade: 1}, domain.ErrValidacao},
		{"local inativo", portsin.MovimentarEstoqueInput{LocalID: inativo.ID, Tipo: "entrada", Quantidade: 1, CustoCentavos: centavosDe(100)}, domain.ErrValidacao},
		{"implantação sem saldo não alocado", portsin.MovimentarEstoqueInput{LocalID: galpao.ID, Tipo: "implantacao", Quantidade: 1}, domain.ErrValidacao},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			caso.input.UsuarioID = c.usuarioID
			caso.input.ProdutoID = produto.ID
			if _, err := c.movimento.Movimentar(c.ctx, caso.input); !errors.Is(err, caso.sentinela) {
				t.Errorf("erro %v, esperado %v", err, caso.sentinela)
			}
		})
	}

	_, err := c.movimento.Movimentar(c.ctx, portsin.MovimentarEstoqueInput{
		UsuarioID:     administradorDaRed,
		ProdutoID:     produto.ID,
		LocalID:       galpao.ID,
		Tipo:          "entrada",
		Quantidade:    1,
		CustoCentavos: centavosDe(100),
	})
	if !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("entrada em local fora do acesso = %v, esperado erro de permissão", err)
	}

	movimento, err := c.movimento.Movimentar(c.ctx, portsin.MovimentarEstoqueInput{
		UsuarioID:     administradorDaRed,
		ProdutoID:     produto.ID,
		LocalID:       daRed.ID,
		Tipo:          "entrada",
		Quantidade:    5,
		CustoCentavos: centavosDe(2590),
	})
	if err != nil {
		t.Fatalf("entrada no local do acesso falhou: %v", err)
	}
	if movimento.LocalID == nil || *movimento.LocalID != daRed.ID || movimento.SaldoLocalApos.Quantidade() != 5 || movimento.Custo.Centavos() != 2590 {
		t.Errorf("movimento devolvido = %+v", movimento)
	}
}

func TestImplantacaoESaidaPorFIFOPeloUsecase(t *testing.T) {
	c := novoCenario(t)
	galpao := c.localPadrao(t)
	produto := c.produtoDeTeste(t)
	c.movimentoAnteriorAosLocais(t, produto, 6)

	movimentar := func(tipo string, quantidade int, custo *int64) (*domainestoque.Movimento, error) {
		return c.movimento.Movimentar(c.ctx, portsin.MovimentarEstoqueInput{
			UsuarioID:     c.usuarioID,
			ProdutoID:     produto.ID,
			LocalID:       galpao.ID,
			Tipo:          tipo,
			Quantidade:    quantidade,
			CustoCentavos: custo,
		})
	}

	if _, err := movimentar("implantacao", 7, nil); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("implantação acima do não alocado = %v, esperado erro de validação", err)
	}
	implantacao, err := movimentar("implantacao", 6, centavosDe(80))
	if err != nil {
		t.Fatalf("implantação falhou: %v", err)
	}
	if implantacao.SaldoApos.Quantidade() != 6 || implantacao.SaldoLocalApos.Quantidade() != 6 {
		t.Errorf("implantação mudou o total para %d, local %d", implantacao.SaldoApos.Quantidade(), implantacao.SaldoLocalApos.Quantidade())
	}

	if _, err := movimentar("entrada", 4, centavosDe(120)); err != nil {
		t.Fatalf("entrada falhou: %v", err)
	}
	if _, err := movimentar("saida", 2, centavosDe(1)); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("saída com custo = %v, esperado erro de validação", err)
	}
	if _, err := movimentar("saida", 11, nil); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("saída acima do saldo do local = %v, esperado erro de validação", err)
	}

	saida, err := movimentar("saida", 8, nil)
	if err != nil {
		t.Fatalf("saída falhou: %v", err)
	}
	if len(saida.Consumos) != 2 || saida.Consumos[0].Quantidade != 6 || saida.Consumos[0].Custo.Centavos() != 80 ||
		saida.Consumos[1].Quantidade != 2 || saida.Consumos[1].Custo.Centavos() != 120 {
		t.Errorf("consumos da saída = %+v, esperado 6×80 e 2×120", saida.Consumos)
	}

	saldo, err := c.movimento.Saldo(c.ctx, produto.ID, c.usuarioID)
	if err != nil || saldo.Produto.Saldo.Quantidade() != 2 || saldo.NaoAlocado != 0 || saldo.Locais[0].Quantidade != 2 {
		t.Errorf("saldo final = %+v (%v), esperado 2 no galpão e nada não alocado", saldo, err)
	}
}
