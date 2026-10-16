package receber_test

import (
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	"github.com/google/uuid"
)

func dadosDoRecebimento(valor int64) domainreceber.DadosRecebimento {
	conta := uuid.New()

	return domainreceber.DadosRecebimento{
		ContaBancariaID: &conta,
		RecebidoEm:      data(2026, 10, 15),
		ValorCentavos:   valor,
	}
}

func TestReceberParcialETotal(t *testing.T) {
	titulo := novoTituloDeTeste(t)
	usuario := uuid.New()

	parcial, historico, err := titulo.Receber(dadosDoRecebimento(50000), usuario, agora)
	if err != nil {
		t.Fatalf("Receber falhou: %v", err)
	}
	if parcial.Tipo != domainreceber.TipoRecebimentoConta || parcial.StatusTituloAnterior != domainreceber.StatusCarteira || parcial.TituloID != titulo.ID {
		t.Fatalf("recebimento = %+v", parcial)
	}
	if titulo.Status != domainreceber.StatusCarteira || titulo.SaldoCentavos() != 100000 {
		t.Fatalf("título depois da parcial = %+v", titulo)
	}
	if historico.Evento != domainreceber.EventoRecebido || historico.ParaStatus != domainreceber.StatusCarteira ||
		historico.Descricao != "conta em 2026-10-15: valor 50000, juros 0, desconto 0 centavos" {
		t.Fatalf("histórico = %+v", historico)
	}

	dados := dadosDoRecebimento(90000)
	dados.JurosCentavos = 1200
	dados.DescontoCentavos = 10000
	total, historico, err := titulo.Receber(dados, usuario, agora)
	if err != nil {
		t.Fatalf("Receber total falhou: %v", err)
	}
	if titulo.Status != domainreceber.StatusRecebido || titulo.SaldoCentavos() != 0 || historico.ParaStatus != domainreceber.StatusRecebido {
		t.Fatalf("título quitado = %+v", titulo)
	}
	if total.AbatidoCentavos() != 100000 || total.EntradaNoCaixaCentavos() != 91200 {
		t.Fatalf("abatido = %d, caixa = %d", total.AbatidoCentavos(), total.EntradaNoCaixaCentavos())
	}

	_, _, err = titulo.Receber(dadosDoRecebimento(1), usuario, agora)
	esperarErro(t, err, domain.ErrConflito, "recebe baixa")
}

func TestReceberValida(t *testing.T) {
	casos := map[string]struct {
		alterar func(*domainreceber.DadosRecebimento)
		trecho  string
	}{
		"tipo inválido":     {func(d *domainreceber.DadosRecebimento) { d.Tipo = "cheque" }, "conta ou fomento"},
		"sem conta":         {func(d *domainreceber.DadosRecebimento) { d.ContaBancariaID = nil }, "conta bancária do recebimento é obrigatória"},
		"fomento com conta": {func(d *domainreceber.DadosRecebimento) { d.Tipo = "fomento" }, "não tem conta bancária"},
		"sem data":          {func(d *domainreceber.DadosRecebimento) { d.RecebidoEm = time.Time{} }, "data do recebimento é obrigatória"},
		"data futura":       {func(d *domainreceber.DadosRecebimento) { d.RecebidoEm = data(2026, 10, 16) }, "no futuro"},
		"valor zero":        {func(d *domainreceber.DadosRecebimento) { d.ValorCentavos = 0 }, "maior que zero"},
		"juros negativos":   {func(d *domainreceber.DadosRecebimento) { d.JurosCentavos = -1 }, "juros"},
		"desconto negativo": {func(d *domainreceber.DadosRecebimento) { d.DescontoCentavos = -1 }, "desconto"},
		"passa do saldo":    {func(d *domainreceber.DadosRecebimento) { d.DescontoCentavos = 1 }, "passa do saldo do título (150000 centavos)"},
	}

	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			titulo := novoTituloDeTeste(t)
			dados := dadosDoRecebimento(150000)
			caso.alterar(&dados)

			_, _, err := titulo.Receber(dados, uuid.New(), agora)
			esperarErro(t, err, domain.ErrValidacao, caso.trecho)
			if titulo.RecebidoCentavos != 0 {
				t.Fatalf("título alterado depois do erro: %+v", titulo)
			}
		})
	}
}

func TestReceberTituloAntecipado(t *testing.T) {
	titulo := novoTituloDeTeste(t)
	titulo.Status = domainreceber.StatusAntecipado

	_, _, err := titulo.Receber(dadosDoRecebimento(150000), uuid.New(), agora)
	esperarErro(t, err, domain.ErrConflito, "repasse ao fomento")

	liquidado, _, err := titulo.Receber(domainreceber.DadosRecebimento{
		Tipo: "fomento", RecebidoEm: data(2026, 10, 15), ValorCentavos: 150000,
	}, uuid.New(), agora)
	if err != nil {
		t.Fatalf("liquidação no fomento falhou: %v", err)
	}
	if titulo.Status != domainreceber.StatusRecebido || liquidado.EntradaNoCaixaCentavos() != 0 || liquidado.StatusTituloAnterior != domainreceber.StatusAntecipado {
		t.Fatalf("título = %+v, recebimento = %+v", titulo, liquidado)
	}

	carteira := novoTituloDeTeste(t)
	_, _, err = carteira.Receber(domainreceber.DadosRecebimento{Tipo: "fomento", RecebidoEm: data(2026, 10, 15), ValorCentavos: 10}, uuid.New(), agora)
	esperarErro(t, err, domain.ErrConflito, "só o título antecipado")
}

func TestEstornarRecebimento(t *testing.T) {
	titulo := novoTituloDeTeste(t)
	if err := titulo.EnviarAoBanco(uuid.New(), agora); err != nil {
		t.Fatal(err)
	}
	recebimento, _, err := titulo.Receber(dadosDoRecebimento(150000), uuid.New(), agora)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = titulo.EstornarRecebimento(recebimento, " ", data(2026, 10, 15), uuid.New(), agora)
	esperarErro(t, err, domain.ErrValidacao, "motivo do estorno é obrigatório")
	_, _, err = titulo.EstornarRecebimento(recebimento, "cheque devolvido", data(2026, 10, 14), uuid.New(), agora)
	esperarErro(t, err, domain.ErrValidacao, "anterior à do recebimento")
	_, _, err = titulo.EstornarRecebimento(recebimento, "cheque devolvido", data(2026, 10, 16), uuid.New(), agora)
	esperarErro(t, err, domain.ErrValidacao, "no futuro")

	estorno, historico, err := titulo.EstornarRecebimento(recebimento, "cheque devolvido", data(2026, 10, 15), uuid.New(), agora)
	if err != nil {
		t.Fatalf("EstornarRecebimento falhou: %v", err)
	}
	if titulo.Status != domainreceber.StatusEmBanco || titulo.SaldoCentavos() != 150000 || !recebimento.Estornado || recebimento.Estorno != estorno {
		t.Fatalf("título = %+v, recebimento = %+v", titulo, recebimento)
	}
	if historico.Evento != domainreceber.EventoRecebimentoEstornado || *historico.DeStatus != domainreceber.StatusRecebido || historico.ParaStatus != domainreceber.StatusEmBanco {
		t.Fatalf("histórico = %+v", historico)
	}

	_, _, err = titulo.EstornarRecebimento(recebimento, "de novo", data(2026, 10, 15), uuid.New(), agora)
	esperarErro(t, err, domain.ErrConflito, "já foi estornado")

	daEntrada := &domainreceber.Recebimento{ID: uuid.New(), TituloID: titulo.ID, RegistrouEntradaPedido: true}
	_, _, err = titulo.EstornarRecebimento(daEntrada, "x", data(2026, 10, 15), uuid.New(), agora)
	esperarErro(t, err, domain.ErrConflito, "entrada paga do pedido")

	deOutro := &domainreceber.Recebimento{ID: uuid.New(), TituloID: uuid.New()}
	_, _, err = titulo.EstornarRecebimento(deOutro, "x", data(2026, 10, 15), uuid.New(), agora)
	esperarErro(t, err, domain.ErrNotFound, "recebimento não encontrado")
}

func TestEstornarRecebimentoParcialMantemOStatus(t *testing.T) {
	titulo := novoTituloDeTeste(t)
	recebimento, _, err := titulo.Receber(dadosDoRecebimento(1000), uuid.New(), agora)
	if err != nil {
		t.Fatal(err)
	}

	if _, _, err := titulo.EstornarRecebimento(recebimento, "lançado errado", data(2026, 10, 15), uuid.New(), agora); err != nil {
		t.Fatalf("estorno falhou: %v", err)
	}
	if titulo.Status != domainreceber.StatusCarteira || titulo.RecebidoCentavos != 0 {
		t.Fatalf("título = %+v", titulo)
	}
}
