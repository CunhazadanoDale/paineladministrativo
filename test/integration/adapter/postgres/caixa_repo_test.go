package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	domainfinanceiro "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/financeiro"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	portscaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/caixa"
	"github.com/CunhazadanoDale/paineladministrativo.git/test/helpers"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var (
	_ portscaixa.ContaBancariaRepository = (*postgres.ContaBancariaRepository)(nil)
	_ portscaixa.EmpresaRepository       = (*postgres.EmpresaRepository)(nil)
	_ portscaixa.ArquivoRepository       = (*postgres.ArquivoRepository)(nil)
	_ portscaixa.UsuarioRepository       = (*postgres.UsuarioRepository)(nil)
	_ portscaixa.CargoRepository         = (*postgres.CargoRepository)(nil)
)

type cenarioBancoCaixa struct {
	ctx       context.Context
	banco     *sqlx.DB
	aberturas *postgres.CaixaAberturaRepository
	extratos  *postgres.ExtratoRepository
	saldos    *postgres.CaixaSaldoRepository
	arquivos  *postgres.ArquivoRepository
	usuarioID uuid.UUID
	mia       uuid.UUID
	red       uuid.UUID
	sicredi   *domainfinanceiro.ContaBancaria
	itau      *domainfinanceiro.ContaBancaria
}

func novoCenarioCaixa(t *testing.T) *cenarioBancoCaixa {
	t.Helper()

	banco := helpers.BancoDoTeste(t)
	ctx := context.Background()
	empresas := postgres.NewEmpresaRepository(banco)
	contas := postgres.NewContaBancariaRepository(banco)

	cargo := novoCargo("Financeiro do caixa")
	if _, err := postgres.NewCargoRepository(banco).Criar(ctx, cargo); err != nil {
		t.Fatalf("criação do cargo falhou: %v", err)
	}
	usuario := novoUsuario(cargo.ID, "Fábio", "fabio.caixa@exemplo.com")
	if _, err := postgres.NewUsuarioRepository(banco).Criar(ctx, usuario); err != nil {
		t.Fatalf("criação do usuário falhou: %v", err)
	}

	c := &cenarioBancoCaixa{
		ctx:       ctx,
		banco:     banco,
		aberturas: postgres.NewCaixaAberturaRepository(banco),
		extratos:  postgres.NewExtratoRepository(banco),
		saldos:    postgres.NewCaixaSaldoRepository(banco),
		arquivos:  postgres.NewArquivoRepository(banco),
		usuarioID: usuario.ID,
		mia:       novaEmpresaNoBanco(t, empresas, "11.222.333/0001-81", "MIA"),
		red:       novaEmpresaNoBanco(t, empresas, "33.400.004/0001-16", "Red Diamond"),
	}

	c.sicredi = contaDeDominio(t, c.mia, domainfinanceiro.Sicredi, "0812", "54321", "Sicredi MIA")
	c.itau = contaDeDominio(t, c.red, domainfinanceiro.Itau, "1234", "98765", "Itaú Red")
	for _, conta := range []*domainfinanceiro.ContaBancaria{c.sicredi, c.itau} {
		if err := contas.Criar(ctx, conta); err != nil {
			t.Fatalf("criação da conta falhou: %v", err)
		}
	}

	return c
}

func (c *cenarioBancoCaixa) abrir(t *testing.T, contaID uuid.UUID, corte time.Time, saldo int64) *domaincaixa.Abertura {
	t.Helper()

	abertura, err := domaincaixa.NovaAbertura(domaincaixa.DadosAbertura{
		ContaBancariaID: contaID, DataCorte: corte, SaldoCentavos: saldo, UsuarioID: c.usuarioID,
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("NovaAbertura = %v", err)
	}
	if err := c.aberturas.Criar(c.ctx, abertura); err != nil {
		t.Fatalf("Criar abertura = %v", err)
	}

	return abertura
}

func (c *cenarioBancoCaixa) novoArquivoOFX(t *testing.T) uuid.UUID {
	t.Helper()

	id := uuid.New()
	if _, err := c.arquivos.Criar(c.ctx, &domainsolicitacao.Arquivo{
		ID:             id,
		ProprietarioID: c.usuarioID,
		Nome:           "extrato.ofx",
		Chave:          c.usuarioID.String() + "/" + id.String() + "/extrato.ofx",
		ContentType:    "application/x-ofx",
		Tamanho:        1024,
		CriadoEm:       time.Now().UTC(),
	}); err != nil {
		t.Fatalf("criação do arquivo falhou: %v", err)
	}

	return id
}

func (c *cenarioBancoCaixa) preparar(t *testing.T, abertura *domaincaixa.Abertura, arquivoID uuid.UUID, saldo *domaincaixa.SaldoInformado, transacoes ...domaincaixa.TransacaoOFX) *domaincaixa.ExtratoPreparado {
	t.Helper()

	preparado, err := domaincaixa.PrepararImportacao(domaincaixa.DadosImportacao{
		Extrato:         &domaincaixa.ExtratoOFX{Saldo: saldo, Transacoes: transacoes},
		ContaBancariaID: abertura.ContaBancariaID,
		ArquivoID:       arquivoID,
		UsuarioID:       c.usuarioID,
		Abertura:        abertura,
		Agora:           time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("PrepararImportacao = %v", err)
	}

	return preparado
}

func (c *cenarioBancoCaixa) importar(t *testing.T, abertura *domaincaixa.Abertura, saldo *domaincaixa.SaldoInformado, transacoes ...domaincaixa.TransacaoOFX) *domaincaixa.ExtratoPreparado {
	t.Helper()

	preparado := c.preparar(t, abertura, c.novoArquivoOFX(t), saldo, transacoes...)
	if err := c.extratos.Importar(c.ctx, preparado, abertura.Versao); err != nil {
		t.Fatalf("Importar = %v", err)
	}

	return preparado
}

func diaDoCaixa(deslocamento int) time.Time {
	return domaincaixa.Hoje(time.Now().UTC()).AddDate(0, 0, deslocamento)
}

func transacaoDoCaixa(fitid string, deslocamento int, valor int64, descricao string) domaincaixa.TransacaoOFX {
	return domaincaixa.TransacaoOFX{FITID: fitid, Data: diaDoCaixa(deslocamento), ValorCentavos: valor, Descricao: descricao}
}

func TestCaixaAberturaCriaAtualizaEConfereSituacao(t *testing.T) {
	c := novoCenarioCaixa(t)

	abertura := c.abrir(t, c.sicredi.ID, diaDoCaixa(-30), -2500)
	salva, err := c.aberturas.Obter(c.ctx, c.sicredi.ID)
	if err != nil || salva == nil || !salva.DataCorte.Equal(diaDoCaixa(-30)) || salva.SaldoCentavos != -2500 || salva.Versao != 1 {
		t.Fatalf("Obter = %+v, %v", salva, err)
	}
	if ausente, err := c.aberturas.Obter(c.ctx, c.itau.ID); err != nil || ausente != nil {
		t.Errorf("abertura inexistente = %+v, %v", ausente, err)
	}

	repetida := *abertura
	if err := c.aberturas.Criar(c.ctx, &repetida); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("segunda abertura = %v, esperado conflito", err)
	}

	if err := abertura.Redefinir(diaDoCaixa(-10), 900, c.usuarioID, time.Now().UTC()); err != nil {
		t.Fatalf("Redefinir = %v", err)
	}
	if err := c.aberturas.Atualizar(c.ctx, abertura); err != nil || abertura.Versao != 2 {
		t.Fatalf("Atualizar = %v, versão %d", err, abertura.Versao)
	}

	velha := *abertura
	velha.Versao = 1
	if err := c.aberturas.Atualizar(c.ctx, &velha); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("atualização com versão velha = %v, esperado conflito", err)
	}

	if _, err := c.banco.ExecContext(c.ctx, `UPDATE conta_bancaria SET ativa = FALSE WHERE id = $1`, c.sicredi.ID); err != nil {
		t.Fatalf("desativar a conta falhou: %v", err)
	}
	if err := c.aberturas.Atualizar(c.ctx, abertura); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("abertura em conta inativa = %v, esperado validação", err)
	}

	if _, err := c.banco.ExecContext(c.ctx, `UPDATE empresa SET ativa = FALSE WHERE id = $1`, c.red); err != nil {
		t.Fatalf("desativar a empresa falhou: %v", err)
	}
	nova, err := domaincaixa.NovaAbertura(domaincaixa.DadosAbertura{
		ContaBancariaID: c.itau.ID, DataCorte: diaDoCaixa(-3), UsuarioID: c.usuarioID,
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("NovaAbertura = %v", err)
	}
	if err := c.aberturas.Criar(c.ctx, nova); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("abertura em empresa inativa = %v, esperado validação", err)
	}
}

func TestExtratoImportaSemDuplicarLancamentos(t *testing.T) {
	c := novoCenarioCaixa(t)
	abertura := c.abrir(t, c.sicredi.ID, diaDoCaixa(-30), 0)

	primeira := c.importar(t, abertura, nil,
		transacaoDoCaixa("A", -5, -25000, "BOLETO"),
		transacaoDoCaixa("B", -3, 120050, "TED"),
	)
	if primeira.Importacao.LinhasNovas != 2 || primeira.Importacao.LinhasRepetidas != 0 {
		t.Errorf("primeira importação = %+v", primeira.Importacao)
	}

	arquivoDaSegunda := c.novoArquivoOFX(t)
	segunda := c.preparar(t, abertura, arquivoDaSegunda, nil,
		transacaoDoCaixa("B", -3, 120050, "TED"),
		transacaoDoCaixa("C", -1, -1290, "TARIFA"),
		transacaoDoCaixa("D", -40, -1, "antes do corte"),
	)
	existentes, err := c.extratos.ContarExistentes(c.ctx, c.sicredi.ID, segunda.Identificadores())
	if err != nil || existentes != 1 {
		t.Errorf("ContarExistentes = %d, %v", existentes, err)
	}
	if err := c.extratos.Importar(c.ctx, segunda, abertura.Versao); err != nil {
		t.Fatalf("segunda importação = %v", err)
	}
	if segunda.Importacao.LinhasNovas != 1 || segunda.Importacao.LinhasRepetidas != 1 || segunda.Importacao.LinhasIgnoradas != 1 {
		t.Errorf("segunda importação = %+v", segunda.Importacao)
	}

	salva, err := c.extratos.ObterImportacao(c.ctx, segunda.Importacao.ID)
	if err != nil || salva == nil || salva.LinhasNovas != 1 || salva.LinhasRepetidas != 1 || salva.LinhasIgnoradas != 1 {
		t.Errorf("importação gravada = %+v, %v", salva, err)
	}
	if importado, err := c.extratos.ArquivoImportado(c.ctx, arquivoDaSegunda); err != nil || !importado {
		t.Errorf("ArquivoImportado = %v, %v", importado, err)
	}

	repetida := c.preparar(t, abertura, arquivoDaSegunda, nil, transacaoDoCaixa("Z", -1, 1, "z"))
	if err := c.extratos.Importar(c.ctx, repetida, abertura.Versao); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("mesmo arquivo = %v, esperado conflito", err)
	}
	if err := c.extratos.Importar(c.ctx, c.preparar(t, abertura, c.novoArquivoOFX(t), nil, transacaoDoCaixa("E", -1, 5, "E")), abertura.Versao+1); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("abertura alterada = %v, esperado conflito", err)
	}

	lancamentos, err := c.extratos.ListarLancamentos(c.ctx, portscaixa.LancamentoFiltro{
		PaginacaoFiltro: domain.PaginacaoFiltro{Page: 1, Size: 10},
		ContaBancariaID: c.sicredi.ID,
	})
	if err != nil || len(lancamentos) != 3 {
		t.Fatalf("ListarLancamentos = %d, %v", len(lancamentos), err)
	}
	if lancamentos[0].Descricao != "BOLETO" || lancamentos[2].Descricao != "TARIFA" || !lancamentos[0].Data.Equal(diaDoCaixa(-5)) {
		t.Errorf("ordem dos lançamentos = %s, %s, %s", lancamentos[0].Descricao, lancamentos[1].Descricao, lancamentos[2].Descricao)
	}

	de, ate := diaDoCaixa(-4), diaDoCaixa(-2)
	filtrados, err := c.extratos.ListarLancamentos(c.ctx, portscaixa.LancamentoFiltro{
		PaginacaoFiltro: domain.PaginacaoFiltro{Page: 1, Size: 10},
		ContaBancariaID: c.sicredi.ID, De: &de, Ate: &ate,
	})
	if err != nil || len(filtrados) != 1 || filtrados[0].Descricao != "TED" {
		t.Errorf("lançamentos filtrados = %+v, %v", filtrados, err)
	}
}

func TestExtratoDesfazSoAImportacaoMaisRecente(t *testing.T) {
	c := novoCenarioCaixa(t)
	abertura := c.abrir(t, c.sicredi.ID, diaDoCaixa(-30), 0)

	primeira := c.importar(t, abertura, nil, transacaoDoCaixa("A", -5, -100, "A"))
	segunda := c.importar(t, abertura, nil, transacaoDoCaixa("A", -5, -100, "A"), transacaoDoCaixa("B", -4, 200, "B"))

	if err := c.extratos.DesfazerImportacao(c.ctx, primeira.Importacao); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("desfazer a antiga = %v, esperado conflito", err)
	}
	if err := c.extratos.DesfazerImportacao(c.ctx, segunda.Importacao); err != nil {
		t.Fatalf("desfazer a mais recente = %v", err)
	}
	if err := c.extratos.DesfazerImportacao(c.ctx, segunda.Importacao); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("desfazer de novo = %v, esperado não encontrado", err)
	}

	lancamentos, err := c.extratos.ListarLancamentos(c.ctx, portscaixa.LancamentoFiltro{
		PaginacaoFiltro: domain.PaginacaoFiltro{Page: 1, Size: 10}, ContaBancariaID: c.sicredi.ID,
	})
	if err != nil || len(lancamentos) != 1 || lancamentos[0].ImportacaoID != primeira.Importacao.ID {
		t.Errorf("lançamentos depois de desfazer = %+v, %v", lancamentos, err)
	}

	if err := c.extratos.DesfazerImportacao(c.ctx, primeira.Importacao); err != nil {
		t.Errorf("desfazer a que virou a mais recente = %v", err)
	}
	if importado, _ := c.extratos.ArquivoImportado(c.ctx, primeira.Importacao.ArquivoID); importado {
		t.Error("o arquivo da importação desfeita pode ser importado de novo")
	}
}

func TestCaixaSaldosSomamOExtratoDepoisDoCorte(t *testing.T) {
	c := novoCenarioCaixa(t)
	abertura := c.abrir(t, c.sicredi.ID, diaDoCaixa(-30), 100000)

	c.importar(t, abertura,
		&domaincaixa.SaldoInformado{Centavos: 194950, Em: diaDoCaixa(-3)},
		transacaoDoCaixa("A", -5, -25000, "BOLETO"),
		transacaoDoCaixa("B", -3, 120000, "TED"),
		transacaoDoCaixa("C", -1, -1290, "TARIFA"),
	)

	saldos, err := c.saldos.Saldos(c.ctx, nil)
	if err != nil || len(saldos) != 2 {
		t.Fatalf("Saldos = %d, %v", len(saldos), err)
	}

	var sicredi, itau *domaincaixa.SaldoConta
	for _, saldo := range saldos {
		switch saldo.ContaBancariaID {
		case c.sicredi.ID:
			sicredi = saldo
		case c.itau.ID:
			itau = saldo
		}
	}

	if atual := sicredi.SaldoAtualCentavos(); atual == nil || *atual != 193710 {
		t.Errorf("saldo atual = %v", atual)
	}
	if diferenca := sicredi.DiferencaDoBancoCentavos(); diferenca == nil || *diferenca != -50 {
		t.Errorf("diferença com o banco = %v", diferenca)
	}
	if sicredi.UltimoLancamentoEm == nil || !sicredi.UltimoLancamentoEm.Equal(diaDoCaixa(-1)) || sicredi.Banco != domainfinanceiro.Sicredi {
		t.Errorf("saldo do Sicredi = %+v", sicredi)
	}
	if itau.TemAbertura() || itau.SaldoAtualCentavos() != nil || itau.MovimentoCentavos != 0 {
		t.Errorf("conta sem abertura = %+v", itau)
	}

	if err := abertura.Redefinir(diaDoCaixa(-4), 100000, c.usuarioID, time.Now().UTC()); err != nil {
		t.Fatalf("Redefinir = %v", err)
	}
	if err := c.aberturas.Atualizar(c.ctx, abertura); err != nil {
		t.Fatalf("Atualizar = %v", err)
	}
	daMIA, err := c.saldos.Saldos(c.ctx, []uuid.UUID{c.mia})
	if err != nil || len(daMIA) != 1 {
		t.Fatalf("Saldos da MIA = %d, %v", len(daMIA), err)
	}
	if atual := daMIA[0].SaldoAtualCentavos(); atual == nil || *atual != 218710 {
		t.Errorf("saldo depois de mover o corte = %v", atual)
	}

	importacoes, err := c.extratos.ListarImportacoes(c.ctx, portscaixa.ImportacaoFiltro{
		PaginacaoFiltro: domain.PaginacaoFiltro{Page: 1, Size: 10}, EmpresaIDs: []uuid.UUID{c.red},
	})
	if err != nil || len(importacoes) != 0 {
		t.Errorf("importações da Red = %d, %v", len(importacoes), err)
	}
	importacoes, err = c.extratos.ListarImportacoes(c.ctx, portscaixa.ImportacaoFiltro{
		PaginacaoFiltro: domain.PaginacaoFiltro{Page: 1, Size: 10}, ContaBancariaID: &c.sicredi.ID,
	})
	if err != nil || len(importacoes) != 1 || importacoes[0].SaldoBancoEm == nil || !importacoes[0].SaldoBancoEm.Equal(diaDoCaixa(-3)) {
		t.Errorf("importações do Sicredi = %+v, %v", importacoes, err)
	}

	contas, err := c.extratos.ContasDoBanco(c.ctx, domainfinanceiro.Sicredi)
	if err != nil || len(contas) != 1 || contas[0].ID != c.sicredi.ID {
		t.Errorf("ContasDoBanco = %+v, %v", contas, err)
	}
}
