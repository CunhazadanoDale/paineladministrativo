package postgres_test

import (
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	portsimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/importacao"
	"github.com/google/uuid"
)

func TestConsultasDeAlertasEAtendimentos(t *testing.T) {
	c := novoCenarioRecebimento(t)
	consultas := postgres.NewImportacaoConsultaRepository(c.banco)
	pedido := c.pedidoDeImportacao(t)
	importacao := c.importacaoDesembaracada(t)
	if err := c.alocar(t, importacao, domainimportacao.AlocacaoInput{
		ImportacaoItemID: importacao.Itens[0].ID, PedidoItemID: pedido.Itens[0].ID, Quantidade: 6,
	}); err != nil {
		t.Fatalf("alocar: %v", err)
	}

	alocacoes, err := consultas.AlocacoesAbertas(c.ctx, portsimportacao.AlertasFiltro{})
	if err != nil || len(alocacoes) != 1 || alocacoes[0].ImportacaoID != importacao.ID || alocacoes[0].PedidoID != pedido.ID ||
		alocacoes[0].Quantidade != 6 || alocacoes[0].ChegadaPrevista == nil || alocacoes[0].PedidoNumero != pedido.Numero {
		t.Fatalf("alocações abertas = %+v (%v)", alocacoes, err)
	}
	if daRed, err := consultas.AlocacoesAbertas(c.ctx, portsimportacao.AlertasFiltro{EmpresaIDs: []uuid.UUID{c.red}}); err != nil || len(daRed) != 0 {
		t.Errorf("alocações da Red = %+v (%v)", daRed, err)
	}

	pedidos, err := consultas.PedidosComFalta(c.ctx, portsimportacao.AlertasFiltro{Empresa: &c.mia})
	if err != nil || len(pedidos) != 2 {
		t.Fatalf("pedidos com falta = %+v (%v)", pedidos, err)
	}
	for _, item := range pedidos {
		switch item.PedidoItemID {
		case pedido.Itens[0].ID:
			if item.Restante != 10 || item.Alocado != 6 || item.SemCobertura() != 4 {
				t.Errorf("item alocado = %+v", item)
			}
		case pedido.Itens[1].ID:
			if item.Restante != 4 || item.Alocado != 0 || item.SemCobertura() != 4 {
				t.Errorf("item sem alocação = %+v", item)
			}
		}
	}

	empresa, err := consultas.EmpresaDoPedido(c.ctx, pedido.ID)
	if err != nil || empresa == nil || *empresa != c.mia {
		t.Errorf("empresa do pedido = %v (%v)", empresa, err)
	}
	if inexistente, err := consultas.EmpresaDoPedido(c.ctx, uuid.New()); err != nil || inexistente != nil {
		t.Errorf("pedido inexistente = %v (%v)", inexistente, err)
	}

	if err := c.importacoes.Gravar(c.ctx, c.gravacaoDoRecebimento(t, c.obterImportacao(t, importacao.ID))); err != nil {
		t.Fatalf("gravar recebimento: %v", err)
	}
	atendimentos, err := consultas.AtendimentosDoPedido(c.ctx, pedido.ID)
	if err != nil || len(atendimentos) != 1 || atendimentos[0].ImportacaoStatus != domainimportacao.StatusRecebida ||
		atendimentos[0].RecebidaEm == nil || *atendimentos[0].Reservada != 6 {
		t.Errorf("atendimentos do pedido = %+v (%v)", atendimentos, err)
	}
	if abertas, _ := consultas.AlocacoesAbertas(c.ctx, portsimportacao.AlertasFiltro{}); len(abertas) != 0 {
		t.Errorf("alocação de importação recebida continua aberta: %+v", abertas)
	}
	pedidos, _ = consultas.PedidosComFalta(c.ctx, portsimportacao.AlertasFiltro{})
	for _, item := range pedidos {
		if item.PedidoItemID == pedido.Itens[0].ID && (item.Reservado != 6 || item.Alocado != 0) {
			t.Errorf("item depois do recebimento = %+v", item)
		}
	}
}

func (c *cenarioBancoRecebimento) obterImportacao(t *testing.T, id uuid.UUID) *domainimportacao.Importacao {
	t.Helper()

	importacao, err := c.importacoes.Obter(c.ctx, id)
	if err != nil || importacao == nil {
		t.Fatalf("obter a importação = %+v (%v)", importacao, err)
	}

	return importacao
}
