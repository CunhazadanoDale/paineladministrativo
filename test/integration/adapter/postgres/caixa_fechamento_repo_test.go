package postgres_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	portscaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/caixa"
	"github.com/google/uuid"
)

type cenarioBancoFechamento struct {
	*cenarioBancoConciliacao
	fechamentos *postgres.CaixaFechamentoRepository
	lancamentos *postgres.LancamentoCaixaRepository
	contaMes    uuid.UUID
	aberturaMes *domaincaixa.Abertura
}

func novoCenarioFechamento(t *testing.T) *cenarioBancoFechamento {
	t.Helper()

	base := novoCenarioConciliacao(t)
	c := &cenarioBancoFechamento{
		cenarioBancoConciliacao: base,
		fechamentos:             postgres.NewCaixaFechamentoRepository(base.banco),
		lancamentos:             postgres.NewLancamentoCaixaRepository(base.banco),
		contaMes:                base.inserirConta(t, base.empresaID, "77777", "Movimento"),
	}

	abertura, err := domaincaixa.NovaAbertura(domaincaixa.DadosAbertura{
		ContaBancariaID: c.contaMes, DataCorte: diaDoTitulo(8, 31), SaldoCentavos: 100000, UsuarioID: c.solicitanteID,
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("NovaAbertura = %v", err)
	}
	if err := postgres.NewCaixaAberturaRepository(c.banco).Criar(c.ctx, abertura); err != nil {
		t.Fatalf("criar abertura = %v", err)
	}
	c.aberturaMes = abertura

	return c
}

func (c *cenarioBancoFechamento) prepararNaConta(t *testing.T, saldo *domaincaixa.SaldoInformado, transacoes ...domaincaixa.TransacaoOFX) *domaincaixa.ExtratoPreparado {
	t.Helper()

	preparado, err := domaincaixa.PrepararImportacao(domaincaixa.DadosImportacao{
		Extrato:         &domaincaixa.ExtratoOFX{Saldo: saldo, Transacoes: transacoes},
		ContaBancariaID: c.contaMes, ArquivoID: c.novoArquivoDoCaixa(t), UsuarioID: c.solicitanteID,
		Abertura: c.aberturaMes, Agora: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("PrepararImportacao = %v", err)
	}

	return preparado
}

func (c *cenarioBancoFechamento) classificar(t *testing.T, linhaID, categoriaID uuid.UUID) *domaincaixa.Classificacao {
	t.Helper()

	classificacao, err := domaincaixa.NovaClassificacao(domaincaixa.DadosClassificacao{
		Lancamento: c.lancamento(t, linhaID), EmpresaID: c.empresaID, Categoria: c.categoria(t, categoriaID),
		UsuarioID: c.solicitanteID, Agora: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("NovaClassificacao = %v", err)
	}
	if err := c.lancamentos.Classificar(c.ctx, classificacao); err != nil {
		t.Fatalf("Classificar = %v", err)
	}

	return classificacao
}

func (c *cenarioBancoFechamento) fechamentoDeSetembro(movimento *domaincaixa.MovimentoDoMes) *domaincaixa.Fechamento {
	final := c.aberturaMes.SaldoCentavos + movimento.EntradasCentavos - movimento.SaidasCentavos

	return &domaincaixa.Fechamento{
		ID: uuid.New(), ContaBancariaID: c.contaMes, Competencia: diaDoTitulo(9, 1),
		SaldoInicialCentavos: c.aberturaMes.SaldoCentavos, EntradasCentavos: movimento.EntradasCentavos,
		SaidasCentavos: movimento.SaidasCentavos, SaldoFinalCentavos: final, SaldoBancoCentavos: final,
		Vigente: true, UsuarioID: c.solicitanteID, CriadoEm: time.Now().UTC(),
	}
}

func TestFechamentoDoMesComTravas(t *testing.T) {
	c := novoCenarioFechamento(t)

	preparado := c.prepararNaConta(t, &domaincaixa.SaldoInformado{Centavos: 104000, Em: diaDoTitulo(9, 30)},
		transacaoNoDia("TAR", diaDoTitulo(9, 10), -1000),
		transacaoNoDia("REN", diaDoTitulo(9, 20), 5000),
		transacaoNoDia("OUT", diaDoTitulo(10, 5), -200),
	)
	if err := c.extratos.Importar(c.ctx, preparado, c.aberturaMes.Versao); err != nil {
		t.Fatalf("Importar = %v", err)
	}

	movimento, err := c.fechamentos.MovimentoDoMes(c.ctx, c.contaMes, diaDoTitulo(9, 1), c.aberturaMes.DataCorte)
	if err != nil {
		t.Fatalf("MovimentoDoMes = %v", err)
	}
	if movimento.EntradasCentavos != 5000 || movimento.SaidasCentavos != 1000 || movimento.LancamentosPendentes != 2 ||
		movimento.ConciliacoesInvalidadas != 0 || movimento.SaldoBancoCentavos == nil || *movimento.SaldoBancoCentavos != 104000 {
		t.Fatalf("movimento de setembro = %+v", movimento)
	}

	gravacao := &portscaixa.GravacaoFechamento{Fechamento: c.fechamentoDeSetembro(movimento), VersaoDaAbertura: c.aberturaMes.Versao}
	if err := c.fechamentos.Fechar(c.ctx, gravacao); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("fechar com pendências = %v", err)
	}

	tarifa := c.classificar(t, preparado.Lancamentos[0].ID, c.despesaID)
	c.classificar(t, preparado.Lancamentos[1].ID, c.receitaID)

	if err := c.fechamentos.Fechar(c.ctx, &portscaixa.GravacaoFechamento{
		Fechamento: c.fechamentoDeSetembro(movimento), VersaoDaAbertura: c.aberturaMes.Versao + 1,
	}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("fechar com abertura alterada = %v", err)
	}

	setembro := c.fechamentoDeSetembro(movimento)
	if err := c.fechamentos.Fechar(c.ctx, &portscaixa.GravacaoFechamento{Fechamento: setembro, VersaoDaAbertura: c.aberturaMes.Versao}); err != nil {
		t.Fatalf("Fechar = %v", err)
	}
	if err := c.fechamentos.Fechar(c.ctx, &portscaixa.GravacaoFechamento{
		Fechamento: c.fechamentoDeSetembro(movimento), VersaoDaAbertura: c.aberturaMes.Versao,
	}); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("fechar setembro de novo = %v", err)
	}

	ultimo, err := c.fechamentos.Ultimo(c.ctx, c.contaMes)
	if err != nil || ultimo == nil || ultimo.ID != setembro.ID || ultimo.SaldoFinalCentavos != 104000 || !ultimo.Competencia.Equal(diaDoTitulo(9, 1)) {
		t.Errorf("Ultimo = %+v, %v", ultimo, err)
	}
	fechados, err := c.extratos.MesesFechados(c.ctx, c.contaMes)
	if err != nil || len(fechados) != 1 || !fechados[0].Equal(diaDoTitulo(9, 1)) {
		t.Errorf("MesesFechados = %+v, %v", fechados, err)
	}
	listados, err := c.fechamentos.Listar(c.ctx, portscaixa.FechamentoFiltro{EmpresaIDs: []uuid.UUID{c.empresaID}, Limite: 10})
	if err != nil || len(listados) != 1 {
		t.Errorf("Listar = %+v, %v", listados, err)
	}

	conciliacao, _ := c.conciliacoes.Obter(c.ctx, tarifa.Conciliacao.ID)
	if err := conciliacao.Desfazer(c.solicitanteID, "categoria errada", time.Now().UTC()); err != nil {
		t.Fatalf("Desfazer no domínio = %v", err)
	}
	if err := c.conciliacoes.Desfazer(c.ctx, conciliacao); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("desfazer em mês fechado = %v", err)
	}

	tardia := c.prepararNaConta(t, nil, transacaoNoDia("LATE", diaDoTitulo(9, 25), -50))
	if err := c.extratos.Importar(c.ctx, tardia, c.aberturaMes.Versao); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("importar linha de mês fechado = %v", err)
	}

	abertura, _ := postgres.NewCaixaAberturaRepository(c.banco).Obter(c.ctx, c.contaMes)
	if err := abertura.Redefinir(diaDoTitulo(8, 30), 1, c.solicitanteID, time.Now().UTC()); err != nil {
		t.Fatalf("Redefinir = %v", err)
	}
	if err := postgres.NewCaixaAberturaRepository(c.banco).Atualizar(c.ctx, abertura); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("mudar abertura com mês fechado = %v", err)
	}

	if _, err := c.banco.ExecContext(c.ctx, `
		INSERT INTO caixa_fechamento (conta_bancaria_id, competencia, saldo_inicial_centavos, entradas_centavos, saidas_centavos,
			saldo_final_centavos, saldo_banco_centavos, usuario_id)
		VALUES ($1, '2026-10-01', 104000, 0, 0, 104000, 104000, $2)
	`, c.contaMes, c.solicitanteID); err != nil {
		t.Fatalf("fechamento de outubro falhou: %v", err)
	}
	if err := setembro.Reabrir(c.solicitanteID, "tarifa errada", time.Now().UTC()); err != nil {
		t.Fatalf("Reabrir no domínio = %v", err)
	}
	if err := c.fechamentos.Reabrir(c.ctx, setembro); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("reabrir com mês seguinte fechado = %v", err)
	}
	if _, err := c.banco.ExecContext(c.ctx, `DELETE FROM caixa_fechamento WHERE competencia = '2026-10-01'`); err != nil {
		t.Fatalf("apagar outubro falhou: %v", err)
	}

	if err := c.fechamentos.Reabrir(c.ctx, setembro); err != nil {
		t.Fatalf("Reabrir = %v", err)
	}
	if err := c.fechamentos.Reabrir(c.ctx, setembro); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("reabrir de novo = %v", err)
	}
	reaberto, _ := c.fechamentos.Obter(c.ctx, setembro.ID)
	if reaberto.Vigente || reaberto.MotivoReabertura == nil || *reaberto.MotivoReabertura != "tarifa errada" {
		t.Errorf("reaberto = %+v", reaberto)
	}
	if err := c.conciliacoes.Desfazer(c.ctx, conciliacao); err != nil {
		t.Errorf("desfazer depois de reabrir = %v", err)
	}
	if semFechamento, _ := c.fechamentos.Ultimo(c.ctx, c.contaMes); semFechamento != nil {
		t.Errorf("último vigente depois de reabrir = %+v", semFechamento)
	}
}
