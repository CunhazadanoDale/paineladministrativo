package importacao_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/importacao"
	"github.com/google/uuid"
)

func TestHistoricoEscondeValoresDoComercial(t *testing.T) {
	c := novoCenario(t)
	importacao := c.incluirParcela(t, c.criar(t, c.financeiro, c.mia, c.dados()), 900000)
	c.fecharCambio(t, importacao, importacao.Parcelas[0].ID, 780_000)

	doComercial, err := c.importacoes.ListarHistorico(c.ctx, importacao.ID, c.comercial)
	if err != nil {
		t.Fatalf("ListarHistorico do comercial: %v", err)
	}
	for _, linha := range doComercial {
		if linha.Acao.ExpoeValores() && linha.Detalhe != "" {
			t.Errorf("comercial viu o detalhe %q de %q", linha.Detalhe, linha.Acao)
		}
	}

	doFinanceiro, _ := c.importacoes.ListarHistorico(c.ctx, importacao.ID, c.financeiro)
	if ultimo := doFinanceiro[len(doFinanceiro)-1]; ultimo.Acao != domainimportacao.AcaoParcelaCambioFechado || ultimo.Detalhe == "" {
		t.Errorf("financeiro perdeu o detalhe: %+v", ultimo)
	}
	if guardado := c.m.historicos[len(c.m.historicos)-1]; guardado.Detalhe == "" {
		t.Error("o histórico gravado foi alterado pela leitura do comercial")
	}
}

func TestPermissaoAntesDaValidacaoDeCadastros(t *testing.T) {
	c := novoCenario(t)
	importacao := c.incluirParcela(t, c.criar(t, c.financeiro, c.mia, c.dados()), 900000)
	cambio := int64(780_000)

	if _, err := c.financas.FecharCambio(c.ctx, portsin.FecharCambioInput{
		UsuarioID: c.comercial, ImportacaoID: importacao.ID, ParcelaID: importacao.Parcelas[0].ID, Versao: importacao.Versao,
		CambioMicro: &cambio, Vencimento: time.Now(), CategoriaFinanceiraID: uuid.New(),
	}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("comercial com categoria inexistente: erro = %v", err)
	}
	if _, err := c.financas.LancarCusto(c.ctx, portsin.LancarCustoInput{
		UsuarioID: c.financeiroRed, ImportacaoID: importacao.ID, Versao: importacao.Versao, Tipo: "multa",
		FornecedorID: uuid.New(), CategoriaFinanceiraID: uuid.New(),
	}); !errors.Is(err, domain.ErrPermissao) {
		t.Errorf("financeiro de outra empresa com dados inválidos: erro = %v", err)
	}
}

func TestEstadoAntesDaValidacaoNoRecebimentoENaAlocacao(t *testing.T) {
	c := novoCenario(t)
	importacao := c.criar(t, c.financeiro, c.mia, c.dados())
	cambio := int64(780_000)

	if _, err := c.recebimento.Receber(c.ctx, portsin.ReceberInput{
		UsuarioID: c.financeiro, ImportacaoID: importacao.ID, Versao: importacao.Versao,
		RecebidaEm: domainimportacao.Hoje(time.Now()), LocalID: c.galpaoRed.ID, CambioMicro: &cambio,
	}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("receber aberta com local de outra empresa: erro = %v", err)
	}

	visao, err := c.importacoes.Cancelar(c.ctx, portsin.CancelarImportacaoInput{
		UsuarioID: c.financeiro, ImportacaoID: importacao.ID, Versao: importacao.Versao, Motivo: "teste",
	})
	if err != nil {
		t.Fatalf("Cancelar: %v", err)
	}
	if _, err := c.recebimento.DefinirAlocacoes(c.ctx, portsin.DefinirAlocacoesInput{
		UsuarioID: c.comercial, ImportacaoID: importacao.ID, Versao: visao.Importacao.Versao,
		Alocacoes: []portsin.AlocacaoInput{{ImportacaoItemID: importacao.Itens[0].ID, PedidoItemID: uuid.New(), Quantidade: 1}},
	}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("alocar importação cancelada: erro = %v", err)
	}
}
