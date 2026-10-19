package caixa_test

import (
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/caixa"
	"github.com/google/uuid"
)

func extratoDoSicredi() string {
	return extratoOFX("748", "0812", "54321-0",
		transacaoOFX("A", -40, "-10.00", "antes do corte"),
		transacaoOFX("B", -5, "-250.00", "BOLETO FORNECEDOR"),
		transacaoOFX("C", -3, "1200.50", "TED RECEBIDA"),
	)
}

func TestImportarExtratoReconheceAContaEIgnoraORepetido(t *testing.T) {
	c := novoCenario(t)
	c.abrirConta(t, c.sicrediMIA, 30, 100000)

	relatorio, err := c.extratos.Importar(c.ctx, portsin.ImportarExtratoInput{
		UsuarioID: c.financeiroMIA,
		ArquivoID: c.novoArquivo(c.financeiroMIA, "application/x-ofx", extratoDoSicredi()),
	})
	if err != nil {
		t.Fatalf("Importar = %v", err)
	}

	importacao := relatorio.Importacao
	if relatorio.Simulacao || relatorio.Total != 3 || importacao.ContaBancariaID != c.sicrediMIA {
		t.Errorf("relatório = %+v", relatorio)
	}
	if importacao.LinhasNovas != 2 || importacao.LinhasRepetidas != 0 || importacao.LinhasIgnoradas != 1 {
		t.Errorf("contagem = %+v", importacao)
	}
	if len(relatorio.Ignoradas) != 1 || relatorio.Ignoradas[0].Motivo != domaincaixa.MotivoAnteriorAoCorte {
		t.Errorf("ignoradas = %+v", relatorio.Ignoradas)
	}
	if importacao.SaldoBancoCentavos == nil || *importacao.SaldoBancoCentavos != 100000 {
		t.Errorf("saldo do banco = %v", importacao.SaldoBancoCentavos)
	}

	maisUm := extratoOFX("748", "812", "543210",
		transacaoOFX("B", -5, "-250.00", "BOLETO FORNECEDOR"),
		transacaoOFX("C", -3, "1200.50", "TED RECEBIDA"),
		transacaoOFX("D", -1, "-12.90", "TARIFA"),
	)

	simulado, err := c.extratos.Importar(c.ctx, portsin.ImportarExtratoInput{
		UsuarioID: c.financeiroMIA, ArquivoID: c.novoArquivo(c.financeiroMIA, "application/x-ofx", maisUm), Simular: true,
	})
	if err != nil {
		t.Fatalf("Importar simulando = %v", err)
	}
	if !simulado.Simulacao || simulado.Importacao.LinhasNovas != 1 || simulado.Importacao.LinhasRepetidas != 2 {
		t.Errorf("simulação = %+v", simulado.Importacao)
	}
	if len(c.extratosRepo.importacoes) != 1 || len(c.extratosRepo.lancamentos) != 2 {
		t.Fatal("a simulação gravou dados")
	}

	segunda, err := c.extratos.Importar(c.ctx, portsin.ImportarExtratoInput{
		UsuarioID: c.financeiroMIA, ArquivoID: c.novoArquivo(c.financeiroMIA, "application/x-ofx", maisUm),
		ContaBancariaID: &c.sicrediMIA,
	})
	if err != nil {
		t.Fatalf("segunda importação = %v", err)
	}
	if segunda.Importacao.LinhasNovas != 1 || segunda.Importacao.LinhasRepetidas != 2 || len(c.extratosRepo.lancamentos) != 3 {
		t.Errorf("segunda importação = %+v", segunda.Importacao)
	}

	lancamentos, err := c.extratos.ListarLancamentos(c.ctx, portsin.ListarLancamentosInput{
		UsuarioID: c.financeiroMIA, ContaBancariaID: c.sicrediMIA, De: timePtr(dia(-4)),
	})
	if err != nil || len(lancamentos) != 2 {
		t.Errorf("lançamentos desde 4 dias atrás = %d, %v", len(lancamentos), err)
	}
}

func TestImportarExtratoConfereArquivo(t *testing.T) {
	c := novoCenario(t)
	c.abrirConta(t, c.sicrediMIA, 30, 0)
	conteudo := extratoDoSicredi()

	_, err := c.extratos.Importar(c.ctx, portsin.ImportarExtratoInput{UsuarioID: c.financeiroMIA})
	esperarErro(t, err, domain.ErrValidacao, "arquivo OFX é obrigatório")

	_, err = c.extratos.Importar(c.ctx, portsin.ImportarExtratoInput{UsuarioID: c.financeiroMIA, ArquivoID: uuid.New()})
	esperarErro(t, err, domain.ErrNotFound, "arquivo não encontrado")

	_, err = c.extratos.Importar(c.ctx, portsin.ImportarExtratoInput{
		UsuarioID: c.financeiroMIA, ArquivoID: c.novoArquivo(c.administrador, "application/x-ofx", conteudo),
	})
	esperarErro(t, err, domain.ErrPermissao, "enviado por você")

	_, err = c.extratos.Importar(c.ctx, portsin.ImportarExtratoInput{
		UsuarioID: c.financeiroMIA, ArquivoID: c.novoArquivo(c.financeiroMIA, "application/pdf", conteudo),
	})
	esperarErro(t, err, domain.ErrValidacao, "arquivo deve ser um extrato OFX")

	daSolicitacao := c.novoArquivo(c.financeiroMIA, "application/x-ofx", conteudo)
	c.arquivosRepo.daSolicitacao[daSolicitacao] = true
	_, err = c.extratos.Importar(c.ctx, portsin.ImportarExtratoInput{UsuarioID: c.financeiroMIA, ArquivoID: daSolicitacao})
	esperarErro(t, err, domain.ErrValidacao, "arquivo de uma solicitação")

	arquivo := c.novoArquivo(c.financeiroMIA, "application/x-ofx", conteudo)
	if _, err := c.extratos.Importar(c.ctx, portsin.ImportarExtratoInput{UsuarioID: c.financeiroMIA, ArquivoID: arquivo}); err != nil {
		t.Fatalf("Importar = %v", err)
	}
	_, err = c.extratos.Importar(c.ctx, portsin.ImportarExtratoInput{UsuarioID: c.financeiroMIA, ArquivoID: arquivo})
	esperarErro(t, err, domain.ErrConflito, "arquivo já importado")

	_, err = c.extratos.Importar(c.ctx, portsin.ImportarExtratoInput{
		UsuarioID: c.financeiroMIA, ArquivoID: c.novoArquivo(c.financeiroMIA, "application/x-ofx", "<html></html>"),
	})
	esperarErro(t, err, domain.ErrValidacao, "não é um extrato OFX")
}

func TestImportarExtratoConfereAConta(t *testing.T) {
	c := novoCenario(t)
	c.abrirConta(t, c.itauRed, 30, 0)
	importar := func(usuarioID uuid.UUID, conteudo string, contaID *uuid.UUID) error {
		_, err := c.extratos.Importar(c.ctx, portsin.ImportarExtratoInput{
			UsuarioID: usuarioID, ArquivoID: c.novoArquivo(usuarioID, "application/x-ofx", conteudo), ContaBancariaID: contaID,
		})
		return err
	}
	transacao := transacaoOFX("X", -2, "-1.00", "tarifa")

	esperarErro(t, importar(c.comum, extratoDoSicredi(), nil), domain.ErrPermissao, "perfil sem permissão para o financeiro")
	esperarErro(t, importar(c.financeiroMIA, extratoDoSicredi(), nil), domain.ErrValidacao, "conta sem saldo inicial")
	esperarErro(t, importar(c.financeiroMIA, extratoOFX("341", "1234", "98765-4", transacao), nil), domain.ErrValidacao, "nenhuma conta ativa no seu acesso")
	esperarErro(t, importar(c.financeiroMIA, extratoOFX("341", "1234", "98765-4", transacao), &c.itauRed), domain.ErrPermissao, "perfil sem acesso")
	esperarErro(t, importar(c.financeiroMIA, extratoOFX("341", "1234", "11111-1", transacao), nil), domain.ErrValidacao, "nenhuma conta ativa no seu acesso")
	esperarErro(t, importar(c.financeiroMIA, extratoOFX("", "", "", transacao), nil), domain.ErrValidacao, "não identifica a conta")
	esperarErro(t, importar(c.financeiroMIA, extratoOFX("341", "1234", "98765-4", transacao), &c.sicrediMIA), domain.ErrValidacao,
		"o extrato é de outra conta (banco 341, agência 1234, conta 98765-4)")
	esperarErro(t, importar(c.financeiroMIA, extratoOFX("237", "1", "111", transacao), nil), domain.ErrValidacao, "nenhuma conta ativa no seu acesso")
	esperarErro(t, importar(c.financeiroMIA, extratoOFX("237", "1", "111", transacao), &c.contaInativa), domain.ErrValidacao, "conta bancária inativa")
	esperarErro(t, importar(c.financeiroMIA, extratoOFX("077", "1", "222", transacao), nil), domain.ErrValidacao, "empresa inativa")
	esperarErro(t, importar(c.financeiroMIA, extratoOFX("748", "0812", "543210", transacao), &c.sicrediMIA), domain.ErrValidacao, "conta sem saldo inicial")
	c.novaConta(t, c.mia, "748", "0812", "543210", "", true)
	esperarErro(t, importar(c.financeiroMIA, extratoOFX("748", "0812", "543210", transacao), &c.sicrediMIA), domain.ErrValidacao, "o extrato é de outra conta")

	if err := importar(c.financeiroRed, extratoOFX("", "", "", transacao), &c.itauRed); err != nil {
		t.Errorf("extrato sem conta com conta informada = %v", err)
	}

	c.novaConta(t, c.red, "341", "1234", "98765", "", true)
	esperarErro(t, importar(c.financeiroRed, extratoOFX("341", "1234", "98765-4", transacao), nil), domain.ErrValidacao,
		"mais de uma conta corresponde")
}

func TestDesfazerEListarImportacoes(t *testing.T) {
	c := novoCenario(t)
	c.abrirConta(t, c.sicrediMIA, 30, 0)
	c.abrirConta(t, c.itauRed, 30, 0)

	mia, err := c.extratos.Importar(c.ctx, portsin.ImportarExtratoInput{
		UsuarioID: c.financeiroMIA, ArquivoID: c.novoArquivo(c.financeiroMIA, "application/x-ofx", extratoDoSicredi()),
	})
	if err != nil {
		t.Fatalf("Importar MIA = %v", err)
	}
	if _, err := c.extratos.Importar(c.ctx, portsin.ImportarExtratoInput{
		UsuarioID: c.financeiroRed,
		ArquivoID: c.novoArquivo(c.financeiroRed, "application/x-ofx", extratoOFX("341", "1234", "98765-4", transacaoOFX("R", -1, "5.00", "pix"))),
	}); err != nil {
		t.Fatalf("Importar Red = %v", err)
	}

	daRed, err := c.extratos.ListarImportacoes(c.ctx, portsin.ListarImportacoesInput{UsuarioID: c.financeiroRed})
	if err != nil || len(daRed) != 1 || daRed[0].ContaBancariaID != c.itauRed {
		t.Errorf("importações da Red = %+v, %v", daRed, err)
	}
	_, err = c.extratos.ListarImportacoes(c.ctx, portsin.ListarImportacoesInput{UsuarioID: c.financeiroRed, ContaBancariaID: &c.sicrediMIA})
	esperarErro(t, err, domain.ErrPermissao, "perfil sem acesso")
	_, err = c.extratos.ListarImportacoes(c.ctx, portsin.ListarImportacoesInput{UsuarioID: c.financeiroRed, Empresa: &c.mia})
	esperarErro(t, err, domain.ErrPermissao, "perfil sem acesso")

	esperarErro(t, c.extratos.DesfazerImportacao(c.ctx, mia.Importacao.ID, c.financeiroMIA), domain.ErrPermissao, "perfil sem permissão")
	esperarErro(t, c.extratos.DesfazerImportacao(c.ctx, uuid.New(), c.administrador), domain.ErrNotFound, "importação de extrato não encontrada")
	if err := c.extratos.DesfazerImportacao(c.ctx, mia.Importacao.ID, c.administrador); err != nil {
		t.Fatalf("DesfazerImportacao = %v", err)
	}

	restantes, err := c.extratos.ListarImportacoes(c.ctx, portsin.ListarImportacoesInput{UsuarioID: c.administrador})
	if err != nil || len(restantes) != 1 || restantes[0].ContaBancariaID != c.itauRed {
		t.Errorf("importações depois de desfazer = %+v, %v", restantes, err)
	}
}

func TestListarLancamentosValidaFiltros(t *testing.T) {
	c := novoCenario(t)

	_, err := c.extratos.ListarLancamentos(c.ctx, portsin.ListarLancamentosInput{UsuarioID: c.financeiroMIA})
	esperarErro(t, err, domain.ErrValidacao, "conta bancária é obrigatória")

	_, err = c.extratos.ListarLancamentos(c.ctx, portsin.ListarLancamentosInput{
		UsuarioID: c.financeiroMIA, ContaBancariaID: c.sicrediMIA, De: timePtr(dia(-1)), Ate: timePtr(dia(-2)),
	})
	esperarErro(t, err, domain.ErrValidacao, "data final anterior à inicial")

	_, err = c.extratos.ListarLancamentos(c.ctx, portsin.ListarLancamentosInput{UsuarioID: c.financeiroRed, ContaBancariaID: c.sicrediMIA})
	esperarErro(t, err, domain.ErrPermissao, "perfil sem acesso")
}

func TestSaldosRespeitamOAcesso(t *testing.T) {
	c := novoCenario(t)

	saldos, err := c.saldos.Listar(c.ctx, c.financeiroRed, nil)
	if err != nil || len(saldos) != 1 || saldos[0].ContaBancariaID != c.itauRed {
		t.Errorf("saldos da Red = %+v, %v", saldos, err)
	}

	if _, err := c.saldos.Listar(c.ctx, c.administrador, nil); err != nil || c.saldosRepo.filtros[1] != nil {
		t.Errorf("administrador deveria ver todas as empresas: filtro %v, erro %v", c.saldosRepo.filtros[1], err)
	}

	_, err = c.saldos.Listar(c.ctx, c.financeiroRed, &c.mia)
	esperarErro(t, err, domain.ErrPermissao, "perfil sem acesso")

	_, err = c.saldos.Listar(c.ctx, c.comum, nil)
	esperarErro(t, err, domain.ErrPermissao, "perfil sem permissão para o financeiro")
}
