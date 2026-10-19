package caixa_test

import (
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
	"github.com/google/uuid"
)

func competenciaRelativa(meses int) time.Time {
	return domaincaixa.CompetenciaDe(domaincaixa.Hoje(time.Now().UTC())).AddDate(0, meses, 0)
}

func (c *cenario) abrirParaFechar(t *testing.T, contaID uuid.UUID) {
	t.Helper()

	abertura, err := domaincaixa.NovaAbertura(domaincaixa.DadosAbertura{
		ContaBancariaID: contaID, DataCorte: competenciaRelativa(-2).AddDate(0, 0, -1), SaldoCentavos: 100000, UsuarioID: c.administrador,
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("NovaAbertura = %v", err)
	}
	abertura.Versao = 4
	c.aberturasRepo.itens[contaID] = abertura
}

func TestPreviaEFechamentoEmSequencia(t *testing.T) {
	c := novoCenario(t)
	c.abrirParaFechar(t, c.sicrediMIA)
	bancoNoFim := int64(130000)
	c.fechamentosRepo.movimento = domaincaixa.MovimentoDoMes{EntradasCentavos: 50000, SaidasCentavos: 20000, SaldoBancoCentavos: &bancoNoFim}
	semLinha := c.movimento(domaincaixa.MovimentoPagamento, c.sicrediMIA, 0, -10)
	semLinha.Data = competenciaRelativa(-2).AddDate(0, 0, 9)

	previa, err := c.fechamentos.Previa(c.ctx, portsin.PreviaInput{
		UsuarioID: c.financeiroMIA, ContaBancariaID: c.sicrediMIA, Competencia: competenciaRelativa(-2).AddDate(0, 0, 4),
	})
	if err != nil {
		t.Fatalf("Previa = %v", err)
	}
	if !previa.Competencia.Equal(competenciaRelativa(-2)) || previa.SaldoInicialCentavos != 100000 || previa.SaldoFinalCentavos() != 130000 ||
		len(previa.MovimentosSemLinha) != 1 {
		t.Errorf("prévia = %+v", previa)
	}

	_, err = c.fechamentos.Fechar(c.ctx, portsin.FecharInput{UsuarioID: c.financeiroMIA, ContaBancariaID: c.sicrediMIA, Competencia: competenciaRelativa(-1)})
	esperarErro(t, err, domain.ErrConflito, "feche antes o mês de")

	detalhe, err := c.fechamentos.Fechar(c.ctx, portsin.FecharInput{
		UsuarioID: c.financeiroMIA, ContaBancariaID: c.sicrediMIA, Competencia: competenciaRelativa(-2),
	})
	if err != nil {
		t.Fatalf("Fechar = %v", err)
	}
	if detalhe.Fechamento.SaldoFinalCentavos != 130000 || len(detalhe.MovimentosSemLinha) != 1 {
		t.Errorf("fechamento = %+v", detalhe)
	}
	gravacao := c.fechamentosRepo.gravacoes[0]
	if gravacao.UltimoID != nil || gravacao.VersaoDaAbertura != 4 {
		t.Errorf("gravação = %+v", gravacao)
	}

	_, err = c.fechamentos.Fechar(c.ctx, portsin.FecharInput{UsuarioID: c.financeiroMIA, ContaBancariaID: c.sicrediMIA, Competencia: competenciaRelativa(-2)})
	esperarErro(t, err, domain.ErrConflito, "mês já fechado")

	c.fechamentosRepo.movimento = domaincaixa.MovimentoDoMes{EntradasCentavos: 1000}
	_, err = c.fechamentos.Fechar(c.ctx, portsin.FecharInput{UsuarioID: c.financeiroMIA, ContaBancariaID: c.sicrediMIA, Competencia: competenciaRelativa(-1)})
	esperarErro(t, err, domain.ErrValidacao, "informe o saldo do banco")
	saldoInformado := int64(131000)
	segundo, err := c.fechamentos.Fechar(c.ctx, portsin.FecharInput{
		UsuarioID: c.financeiroMIA, ContaBancariaID: c.sicrediMIA, Competencia: competenciaRelativa(-1), SaldoBancoCentavos: &saldoInformado,
	})
	if err != nil {
		t.Fatalf("segundo fechamento = %v", err)
	}
	if segundo.Fechamento.SaldoInicialCentavos != 130000 || c.fechamentosRepo.gravacoes[1].UltimoID == nil ||
		*c.fechamentosRepo.gravacoes[1].UltimoID != detalhe.Fechamento.ID {
		t.Errorf("segundo fechamento = %+v", segundo.Fechamento)
	}

	_, err = c.fechamentos.Fechar(c.ctx, portsin.FecharInput{UsuarioID: c.financeiroMIA, ContaBancariaID: c.sicrediMIA, Competencia: competenciaRelativa(0)})
	esperarErro(t, err, domain.ErrValidacao, "só é possível fechar mês encerrado")
	if _, err := c.fechamentos.Previa(c.ctx, portsin.PreviaInput{
		UsuarioID: c.financeiroMIA, ContaBancariaID: c.sicrediMIA, Competencia: competenciaRelativa(0),
	}); err != nil {
		t.Errorf("prévia do mês corrente = %v", err)
	}
}

func TestFechamentoBloqueadoEPermissoes(t *testing.T) {
	c := novoCenario(t)
	c.abrirParaFechar(t, c.sicrediMIA)
	banco := int64(100000)
	c.fechamentosRepo.movimento = domaincaixa.MovimentoDoMes{LancamentosPendentes: 3, SaldoBancoCentavos: &banco}
	input := portsin.FecharInput{UsuarioID: c.financeiroMIA, ContaBancariaID: c.sicrediMIA, Competencia: competenciaRelativa(-2)}

	_, err := c.fechamentos.Fechar(c.ctx, input)
	esperarErro(t, err, domain.ErrConflito, "3 lançamento(s) pendente(s)")

	input.UsuarioID = c.financeiroRed
	_, err = c.fechamentos.Fechar(c.ctx, input)
	esperarErro(t, err, domain.ErrPermissao, "perfil sem acesso")
	input.UsuarioID = c.comum
	_, err = c.fechamentos.Fechar(c.ctx, input)
	esperarErro(t, err, domain.ErrPermissao, "perfil sem permissão para o financeiro")

	_, err = c.fechamentos.Previa(c.ctx, portsin.PreviaInput{UsuarioID: c.financeiroRed, ContaBancariaID: c.itauRed, Competencia: competenciaRelativa(-1)})
	esperarErro(t, err, domain.ErrValidacao, "conta sem saldo inicial")
	_, err = c.fechamentos.Previa(c.ctx, portsin.PreviaInput{UsuarioID: c.financeiroMIA, ContaBancariaID: c.sicrediMIA})
	esperarErro(t, err, domain.ErrValidacao, "competência é obrigatória")
}

func TestReabrirObterEListarFechamentos(t *testing.T) {
	c := novoCenario(t)
	c.abrirParaFechar(t, c.sicrediMIA)
	banco := int64(100000)
	c.fechamentosRepo.movimento = domaincaixa.MovimentoDoMes{SaldoBancoCentavos: &banco}

	detalhe, err := c.fechamentos.Fechar(c.ctx, portsin.FecharInput{
		UsuarioID: c.financeiroMIA, ContaBancariaID: c.sicrediMIA, Competencia: competenciaRelativa(-2),
	})
	if err != nil {
		t.Fatalf("Fechar = %v", err)
	}
	id := detalhe.Fechamento.ID

	c.fechamentosRepo.movimento = domaincaixa.MovimentoDoMes{ConciliacoesInvalidadas: 1}
	obtido, err := c.fechamentos.Obter(c.ctx, id, c.financeiroMIA)
	if err != nil || obtido.ConciliacoesInvalidadas != 1 || obtido.Fechamento.ID != id {
		t.Errorf("Obter = %+v, %v", obtido, err)
	}
	_, err = c.fechamentos.Obter(c.ctx, id, c.financeiroRed)
	esperarErro(t, err, domain.ErrPermissao, "perfil sem acesso")
	_, err = c.fechamentos.Obter(c.ctx, uuid.New(), c.financeiroMIA)
	esperarErro(t, err, domain.ErrNotFound, "fechamento não encontrado")

	if _, err := c.fechamentos.Listar(c.ctx, portsin.ListarFechamentosInput{
		UsuarioID: c.financeiroRed, Competencia: timePtr(competenciaRelativa(-2).AddDate(0, 0, 10)),
	}); err != nil {
		t.Fatalf("Listar = %v", err)
	}
	filtro := c.fechamentosRepo.filtros[0]
	if len(filtro.EmpresaIDs) != 1 || filtro.EmpresaIDs[0] != c.red || !filtro.Competencia.Equal(competenciaRelativa(-2)) {
		t.Errorf("filtro = %+v", filtro)
	}

	_, err = c.fechamentos.Reabrir(c.ctx, portsin.ReabrirInput{UsuarioID: c.financeiroMIA, FechamentoID: id, Motivo: "tarifa esquecida"})
	esperarErro(t, err, domain.ErrPermissao, "perfil sem permissão")
	_, err = c.fechamentos.Reabrir(c.ctx, portsin.ReabrirInput{UsuarioID: c.administrador, FechamentoID: id})
	esperarErro(t, err, domain.ErrValidacao, "motivo")
	reaberto, err := c.fechamentos.Reabrir(c.ctx, portsin.ReabrirInput{UsuarioID: c.administrador, FechamentoID: id, Motivo: "tarifa esquecida"})
	if err != nil || reaberto.Vigente {
		t.Fatalf("Reabrir = %+v, %v", reaberto, err)
	}
	_, err = c.fechamentos.Reabrir(c.ctx, portsin.ReabrirInput{UsuarioID: c.administrador, FechamentoID: id, Motivo: "de novo"})
	esperarErro(t, err, domain.ErrConflito, "já reaberto")

	c.fechamentosRepo.movimento = domaincaixa.MovimentoDoMes{}
	if _, err := c.fechamentos.Fechar(c.ctx, portsin.FecharInput{
		UsuarioID: c.financeiroMIA, ContaBancariaID: c.sicrediMIA, Competencia: competenciaRelativa(-2), SaldoBancoCentavos: &banco,
	}); err != nil {
		t.Errorf("fechar de novo depois de reabrir = %v", err)
	}
}

func TestImportacaoRecebeOsMesesFechados(t *testing.T) {
	c := novoCenario(t)
	c.abrirConta(t, c.sicrediMIA, 30, 0)
	c.extratosRepo.mesesFechados = []time.Time{domaincaixa.CompetenciaDe(dia(-5))}

	relatorio, err := c.extratos.Importar(c.ctx, portsin.ImportarExtratoInput{
		UsuarioID: c.financeiroMIA,
		ArquivoID: c.novoArquivo(c.financeiroMIA, "application/x-ofx", extratoOFX("748", "0812", "54321-0", transacaoOFX("A", -5, "-1.00", "x"))),
		Simular:   true,
	})
	if err != nil {
		t.Fatalf("Importar = %v", err)
	}
	if len(relatorio.Ignoradas) != 1 || relatorio.Ignoradas[0].Motivo != domaincaixa.MotivoMesFechado {
		t.Errorf("ignoradas = %+v", relatorio.Ignoradas)
	}
}
