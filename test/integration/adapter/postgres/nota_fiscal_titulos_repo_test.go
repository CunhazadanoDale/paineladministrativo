package postgres_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/notafiscal"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	portsnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/notafiscal"
)

var _ portsnotafiscal.TituloReceberRepository = (*postgres.TituloReceberRepository)(nil)

func (c *cenarioBancoNotas) geracaoDaNota(t *testing.T, nota *domainnotafiscal.NotaFiscal) *domainreceber.Geracao {
	t.Helper()

	geracao, err := domainreceber.NovaGeracao(
		domainreceber.OrigemNotaFiscalDuplicata, nota.EmpresaID, c.clienteID, nil, nota.ParcelasDosTitulos(), c.usuarioID, time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("NovaGeracao = %v", err)
	}

	return geracao
}

func esperarErroDaGeracao(t *testing.T, err error, sentinela error, trecho string) {
	t.Helper()

	if !errors.Is(err, sentinela) || !strings.Contains(err.Error(), trecho) {
		t.Errorf("erro = %v, esperado %v com %q", err, sentinela, trecho)
	}
}

func TestTitulosDaNotaFiscalPelaGeracaoDoM9(t *testing.T) {
	c := novoCenarioNotas(t)
	titulos := postgres.NewTituloReceberRepository(c.banco)
	nota := c.criarNota(t, 80, c.mia, time.Now().UTC())

	resultado, err := titulos.GerarDeOrigem(c.ctx, c.geracaoDaNota(t, nota))
	if err != nil || len(resultado.Gerados) != 2 || len(resultado.JaExistentes) != 0 {
		t.Fatalf("gerar = %+v, %v", resultado, err)
	}

	var gravados []struct {
		Numero  string `db:"numero"`
		Parcela int    `db:"parcela"`
		Valor   int64  `db:"valor_centavos"`
		Status  string `db:"status"`
	}
	if err := c.banco.SelectContext(c.ctx, &gravados, `
		SELECT numero, parcela, valor_centavos, status FROM titulo_receber
		WHERE origem = 'nota_fiscal_duplicata' ORDER BY parcela
	`); err != nil {
		t.Fatalf("ler títulos = %v", err)
	}
	if len(gravados) != 2 || gravados[0].Numero != "NF 1-80" || gravados[0].Parcela != 1 || gravados[0].Valor != 54425 || gravados[0].Status != "carteira" {
		t.Errorf("títulos gravados = %+v", gravados)
	}

	repetido, err := titulos.GerarDeOrigem(c.ctx, c.geracaoDaNota(t, nota))
	if err != nil || len(repetido.Gerados) != 0 || len(repetido.JaExistentes) != 2 {
		t.Errorf("gerar de novo = %+v, %v", repetido, err)
	}

	versao := nota.Versao
	nota.Status = domainnotafiscal.StatusCancelada
	if err := c.notas.Atualizar(c.ctx, &portsnotafiscal.Alteracao{
		Nota: nota, VersaoLida: versao, StatusAnterior: domainnotafiscal.StatusAutorizada,
	}); err != nil {
		t.Fatalf("cancelar a nota = %v", err)
	}

	_, err = titulos.GerarDeOrigem(c.ctx, c.geracaoDaNota(t, nota))
	esperarErroDaGeracao(t, err, domain.ErrConflito, "origem cancelada")

	cancelamento, err := domainreceber.NovoCancelamentoDeOrigem(
		domainreceber.OrigemNotaFiscalDuplicata, nota.DuplicataIDs(), nota.MotivoDoCancelamento(), c.usuarioID, time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("NovoCancelamentoDeOrigem = %v", err)
	}
	efeito, err := titulos.CancelarPorOrigem(c.ctx, cancelamento)
	if err != nil || len(efeito.Cancelados) != 2 || len(efeito.Marcados) != 0 {
		t.Errorf("cancelar por origem = %+v, %v", efeito, err)
	}
}

func TestGeracaoDoM9RecusaNotaQueNaoEhVenda(t *testing.T) {
	c := novoCenarioNotas(t)
	titulos := postgres.NewTituloReceberRepository(c.banco)

	remessa := c.novaNota(90, c.mia, time.Now().UTC())
	remessa.ArquivoID = c.novoArquivo(t)
	remessa.Natureza = domainnotafiscal.NaturezaRemessa
	if err := c.notas.Criar(c.ctx, remessa); err != nil {
		t.Fatalf("criar remessa = %v", err)
	}
	_, err := titulos.GerarDeOrigem(c.ctx, c.geracaoDaNota(t, remessa))
	esperarErroDaGeracao(t, err, domain.ErrValidacao, "só nota fiscal de venda")

	transferencia := c.novaNota(91, c.mia, time.Now().UTC())
	transferencia.ArquivoID = c.novoArquivo(t)
	transferencia.ClienteID = nil
	transferencia.EmpresaDestinatariaID = &c.red
	transferencia.EntreEmpresas = true
	transferencia.Natureza = domainnotafiscal.NaturezaTransferencia
	if err := c.notas.Criar(c.ctx, transferencia); err != nil {
		t.Fatalf("criar transferência = %v", err)
	}
	_, err = titulos.GerarDeOrigem(c.ctx, c.geracaoDaNota(t, transferencia))
	esperarErroDaGeracao(t, err, domain.ErrValidacao, "entre empresas")

	var quantidade int
	if err := c.banco.GetContext(c.ctx, &quantidade, `SELECT count(*) FROM titulo_receber`); err != nil || quantidade != 0 {
		t.Errorf("títulos gravados = %d, %v", quantidade, err)
	}
}
