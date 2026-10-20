package postgres_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	portsimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/importacao"
)

func TestRecebimentoRecusaReservaAcimaDoRestante(t *testing.T) {
	c := novoCenarioRecebimento(t)
	pedido := c.pedidoDeImportacao(t)
	importacao := c.importacaoDesembaracada(t)
	if err := c.alocar(t, importacao, domainimportacao.AlocacaoInput{
		ImportacaoItemID: importacao.Itens[0].ID, PedidoItemID: pedido.Itens[0].ID, Quantidade: 10,
	}); err != nil {
		t.Fatalf("alocar: %v", err)
	}
	gravacao := c.gravacaoDoRecebimento(t, c.obterImportacao(t, importacao.ID))

	if _, err := c.banco.Exec(`
		INSERT INTO produto_reserva (produto_id, local_id, quantidade, origem, origem_id, status, usuario_id)
		VALUES ($1, $2, 3, 'pedido_item', $3, 'ativa', $4)
	`, c.produto.ID, c.loja.ID, pedido.Itens[0].ID, c.usuarioID); err != nil {
		t.Fatalf("reserva concorrente: %v", err)
	}

	if err := c.importacoes.Gravar(c.ctx, gravacao); !errors.Is(err, domain.ErrConflito) {
		t.Fatalf("recebimento com reserva concorrente: erro = %v", err)
	}
	if lida := c.obterImportacao(t, importacao.ID); lida.Status != domainimportacao.StatusDesembaracada {
		t.Errorf("importação gravada mesmo com a recusa: %+v", lida)
	}
}

func TestAlocacaoRecusaPedidoQueSaiuDeAtendimento(t *testing.T) {
	c := novoCenarioRecebimento(t)
	pedido := c.pedidoDeImportacao(t)
	importacao := c.importacaoDesembaracada(t)

	if _, err := c.banco.Exec(`UPDATE pedido SET status = 'cancelado', cancelado_motivo = 'teste' WHERE id = $1`, pedido.ID); err != nil {
		t.Fatalf("cancelar pedido: %v", err)
	}
	agora := time.Now().UTC()
	importacao.Alocacoes = []*domainimportacao.Alocacao{{
		ImportacaoItemID: importacao.Itens[0].ID, PedidoItemID: pedido.Itens[0].ID, Quantidade: 5, UsuarioID: c.usuarioID,
		CriadoEm: agora, AtualizadoEm: agora,
	}}
	if err := c.importacoes.Gravar(c.ctx, &portsimportacao.Gravacao{
		Importacao: importacao, VersaoLida: importacao.Versao, StatusAnterior: importacao.Status, SubstituirAlocacoes: true,
	}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("alocação de pedido cancelado passou pelo banco: %v", err)
	}
}

func TestObterLeAImportacaoCompleta(t *testing.T) {
	c := novoCenarioRecebimento(t)
	pedido := c.pedidoDeImportacao(t)
	importacao := c.importacaoDesembaracada(t)
	if err := c.alocar(t, importacao, domainimportacao.AlocacaoInput{
		ImportacaoItemID: importacao.Itens[0].ID, PedidoItemID: pedido.Itens[0].ID, Quantidade: 4,
	}); err != nil {
		t.Fatalf("alocar: %v", err)
	}

	lida := c.obterImportacao(t, importacao.ID)
	if len(lida.Itens) != 2 || len(lida.Alocacoes) != 1 || lida.Parcelas == nil || lida.Custos == nil {
		t.Errorf("importação lida = %+v", lida)
	}
}
