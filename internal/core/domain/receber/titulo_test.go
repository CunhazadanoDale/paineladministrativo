package receber_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainreceber "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/receber"
	"github.com/google/uuid"
)

var agora = time.Date(2026, 10, 15, 15, 0, 0, 0, time.UTC)

func data(ano int, mes time.Month, dia int) time.Time {
	return time.Date(ano, mes, dia, 0, 0, 0, 0, time.UTC)
}

func dadosDeTeste() domainreceber.DadosTitulo {
	categoriaID := uuid.New()

	return domainreceber.DadosTitulo{
		DadosEdicao: domainreceber.DadosEdicao{
			Numero:                "  ALG-10  ",
			Parcela:               1,
			Vencimento:            data(2026, 11, 10),
			ValorCentavos:         150000,
			CategoriaFinanceiraID: &categoriaID,
			Observacao:            "  aluguel do galpão  ",
		},
		ClienteID: uuid.New(),
	}
}

func novoTituloDeTeste(t *testing.T) *domainreceber.Titulo {
	t.Helper()

	titulo, err := domainreceber.NovoTituloManual(uuid.New(), uuid.New(), dadosDeTeste(), agora)
	if err != nil {
		t.Fatalf("NovoTituloManual falhou: %v", err)
	}

	return titulo
}

func esperarErro(t *testing.T, err error, sentinela error, trecho string) {
	t.Helper()

	if !errors.Is(err, sentinela) || !strings.Contains(err.Error(), trecho) {
		t.Fatalf("erro = %v; esperado %v com %q", err, sentinela, trecho)
	}
}

func TestNovoTituloManualNasceEmCarteira(t *testing.T) {
	titulo := novoTituloDeTeste(t)

	if titulo.Status != domainreceber.StatusCarteira || titulo.Origem != domainreceber.OrigemManual || titulo.OrigemID != nil {
		t.Fatalf("título = %+v", titulo)
	}
	if titulo.Numero != "ALG-10" || titulo.Observacao != "aluguel do galpão" || titulo.Versao != 1 {
		t.Fatalf("textos ou versão = %q, %q, %d", titulo.Numero, titulo.Observacao, titulo.Versao)
	}
	if titulo.Competencia.String() != "2026-11" || titulo.SaldoCentavos() != 150000 {
		t.Fatalf("competência ou saldo = %s, %d", titulo.Competencia, titulo.SaldoCentavos())
	}
}

func TestNovoTituloManualComContaNasceEmBanco(t *testing.T) {
	dados := dadosDeTeste()
	contaID := uuid.New()
	dados.ContaBancariaID = &contaID
	competencia, err := domainreceber.NovaCompetencia("2026-10")
	if err != nil {
		t.Fatal(err)
	}
	dados.Competencia = &competencia

	titulo, err := domainreceber.NovoTituloManual(uuid.New(), uuid.New(), dados, agora)
	if err != nil {
		t.Fatalf("NovoTituloManual falhou: %v", err)
	}
	if titulo.Status != domainreceber.StatusEmBanco || *titulo.ContaBancariaID != contaID || titulo.Competencia.String() != "2026-10" {
		t.Fatalf("título = %+v", titulo)
	}
}

func TestNovoTituloManualValidaOsDados(t *testing.T) {
	casos := map[string]struct {
		alterar func(*domainreceber.DadosTitulo)
		trecho  string
	}{
		"sem cliente":        {func(d *domainreceber.DadosTitulo) { d.ClienteID = uuid.Nil }, "cliente do título é obrigatório"},
		"sem número":         {func(d *domainreceber.DadosTitulo) { d.Numero = "  " }, "número do título é obrigatório"},
		"número longo":       {func(d *domainreceber.DadosTitulo) { d.Numero = strings.Repeat("9", 31) }, "no máximo 30"},
		"parcela zero":       {func(d *domainreceber.DadosTitulo) { d.Parcela = 0 }, "parcela deve estar entre 1 e 999"},
		"parcela grande":     {func(d *domainreceber.DadosTitulo) { d.Parcela = 1000 }, "parcela deve estar entre 1 e 999"},
		"sem vencimento":     {func(d *domainreceber.DadosTitulo) { d.Vencimento = time.Time{} }, "vencimento é obrigatório"},
		"vencimento passado": {func(d *domainreceber.DadosTitulo) { d.Vencimento = data(2026, 10, 14) }, "anterior a hoje"},
		"valor zero":         {func(d *domainreceber.DadosTitulo) { d.ValorCentavos = 0 }, "maior que zero"},
		"sem categoria":      {func(d *domainreceber.DadosTitulo) { d.CategoriaFinanceiraID = nil }, "categoria financeira do título manual"},
		"categoria nula":     {func(d *domainreceber.DadosTitulo) { nula := uuid.Nil; d.CategoriaFinanceiraID = &nula }, "categoria financeira"},
		"observação longa":   {func(d *domainreceber.DadosTitulo) { d.Observacao = strings.Repeat("a", 1001) }, "no máximo 1000"},
	}

	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			dados := dadosDeTeste()
			caso.alterar(&dados)

			_, err := domainreceber.NovoTituloManual(uuid.New(), uuid.New(), dados, agora)
			esperarErro(t, err, domain.ErrValidacao, caso.trecho)
		})
	}
}

func TestVencimentoDeHojeEmBrasiliaNaoEstaNoPassado(t *testing.T) {
	dados := dadosDeTeste()
	dados.Vencimento = data(2026, 10, 15)
	noiteEmBrasilia := time.Date(2026, 10, 16, 1, 0, 0, 0, time.UTC)

	if _, err := domainreceber.NovoTituloManual(uuid.New(), uuid.New(), dados, noiteEmBrasilia); err != nil {
		t.Fatalf("às 22h de Brasília o dia ainda é 15: %v", err)
	}
}

func TestVencidoEDerivadoDaData(t *testing.T) {
	titulo := novoTituloDeTeste(t)

	if titulo.Vencido(data(2026, 11, 10)) {
		t.Fatal("no dia do vencimento o título não está vencido")
	}
	if !titulo.Vencido(data(2026, 11, 11)) {
		t.Fatal("depois do vencimento o título está vencido")
	}

	titulo.RecebidoCentavos = titulo.ValorCentavos
	if titulo.Vencido(data(2026, 11, 11)) {
		t.Fatal("título sem saldo não está vencido")
	}
}

func TestEditarDescreveOsCamposAlterados(t *testing.T) {
	titulo := novoTituloDeTeste(t)
	dados := dadosDeTeste().DadosEdicao
	dados.CategoriaFinanceiraID = titulo.CategoriaFinanceiraID
	dados.ValorCentavos = 200000
	dados.Vencimento = data(2026, 11, 20)

	descricao, err := titulo.Editar(dados, agora)
	if err != nil {
		t.Fatalf("Editar falhou: %v", err)
	}
	if descricao != "alterados: vencimento, valor" {
		t.Fatalf("descrição = %q", descricao)
	}
	if titulo.ValorCentavos != 200000 || !titulo.Vencimento.Equal(data(2026, 11, 20)) {
		t.Fatalf("título = %+v", titulo)
	}

	igual, err := titulo.Editar(dados, agora)
	if err != nil || igual != "" {
		t.Fatalf("editar com os mesmos dados = %q, %v", igual, err)
	}
}

func TestEditarTituloVencidoSemMudarOVencimento(t *testing.T) {
	titulo := novoTituloDeTeste(t)
	depois := time.Date(2026, 12, 1, 15, 0, 0, 0, time.UTC)
	dados := dadosDeTeste().DadosEdicao
	dados.CategoriaFinanceiraID = titulo.CategoriaFinanceiraID
	dados.Observacao = "cobrar na portaria"

	if _, err := titulo.Editar(dados, depois); err != nil {
		t.Fatalf("editar sem mudar o vencimento vencido falhou: %v", err)
	}

	dados.Vencimento = data(2026, 11, 20)
	_, err := titulo.Editar(dados, depois)
	esperarErro(t, err, domain.ErrValidacao, "anterior a hoje")
}

func TestEditarSoTituloManualEmCarteiraSemRecebimento(t *testing.T) {
	dados := dadosDeTeste().DadosEdicao

	gerado := novoTituloDeTeste(t)
	origemID := uuid.New()
	gerado.Origem, gerado.OrigemID = domainreceber.OrigemNotaFiscalDuplicata, &origemID
	_, err := gerado.Editar(dados, agora)
	esperarErro(t, err, domain.ErrConflito, "só o título manual")

	emBanco := novoTituloDeTeste(t)
	if err := emBanco.EnviarAoBanco(uuid.New(), agora); err != nil {
		t.Fatal(err)
	}
	_, err = emBanco.Editar(dados, agora)
	esperarErro(t, err, domain.ErrConflito, "em carteira")

	parcial := novoTituloDeTeste(t)
	parcial.RecebidoCentavos = 100
	_, err = parcial.Editar(dados, agora)
	esperarErro(t, err, domain.ErrConflito, "com recebimento")
}

func TestProrrogar(t *testing.T) {
	titulo := novoTituloDeTeste(t)

	_, err := titulo.Prorrogar(data(2026, 11, 10), "cliente pediu", agora)
	esperarErro(t, err, domain.ErrValidacao, "posterior ao atual")
	_, err = titulo.Prorrogar(data(2026, 11, 20), " ", agora)
	esperarErro(t, err, domain.ErrValidacao, "motivo da prorrogação é obrigatório")

	descricao, err := titulo.Prorrogar(data(2026, 11, 20), "cliente pediu", agora)
	if err != nil {
		t.Fatalf("Prorrogar falhou: %v", err)
	}
	if descricao != "vencimento de 2026-11-10 para 2026-11-20: cliente pediu" || !titulo.Vencimento.Equal(data(2026, 11, 20)) {
		t.Fatalf("descrição = %q, vencimento = %s", descricao, titulo.Vencimento)
	}

	vencido := novoTituloDeTeste(t)
	_, err = vencido.Prorrogar(data(2026, 11, 12), "atraso", time.Date(2026, 11, 15, 15, 0, 0, 0, time.UTC))
	esperarErro(t, err, domain.ErrValidacao, "anterior a hoje")

	vencido.Status = domainreceber.StatusAntecipado
	_, err = vencido.Prorrogar(data(2026, 12, 1), "atraso", agora)
	esperarErro(t, err, domain.ErrConflito, "carteira ou em banco")
}

func TestEnviarERetirarDoBanco(t *testing.T) {
	titulo := novoTituloDeTeste(t)
	contaID := uuid.New()

	esperarErro(t, titulo.EnviarAoBanco(uuid.Nil, agora), domain.ErrValidacao, "conta bancária de cobrança")
	if err := titulo.EnviarAoBanco(contaID, agora); err != nil {
		t.Fatalf("EnviarAoBanco falhou: %v", err)
	}
	if titulo.Status != domainreceber.StatusEmBanco || *titulo.ContaBancariaID != contaID {
		t.Fatalf("título = %+v", titulo)
	}
	esperarErro(t, titulo.EnviarAoBanco(contaID, agora), domain.ErrConflito, "em_banco")

	if err := titulo.RetirarDoBanco(agora); err != nil {
		t.Fatalf("RetirarDoBanco falhou: %v", err)
	}
	if titulo.Status != domainreceber.StatusCarteira || titulo.ContaBancariaID != nil {
		t.Fatalf("título = %+v", titulo)
	}
	esperarErro(t, titulo.RetirarDoBanco(agora), domain.ErrConflito, "carteira")

	momento := agora
	titulo.OrigemCanceladaEm = &momento
	esperarErro(t, titulo.EnviarAoBanco(contaID, agora), domain.ErrConflito, "origem do título foi cancelada")
}

func TestRegistrarEnvioBoleto(t *testing.T) {
	titulo := novoTituloDeTeste(t)

	esperarErro(t, titulo.RegistrarEnvioBoleto("email", agora, agora), domain.ErrConflito, "em banco")

	if err := titulo.EnviarAoBanco(uuid.New(), agora); err != nil {
		t.Fatal(err)
	}
	esperarErro(t, titulo.RegistrarEnvioBoleto("email", agora.Add(time.Hour), agora), domain.ErrValidacao, "futuro")

	enviadoEm := agora.Add(-time.Hour)
	if err := titulo.RegistrarEnvioBoleto("WhatsApp", enviadoEm, agora); err != nil {
		t.Fatalf("RegistrarEnvioBoleto falhou: %v", err)
	}
	if *titulo.BoletoCanal != domainreceber.CanalBoletoWhatsapp || !titulo.BoletoEnviadoEm.Equal(enviadoEm) {
		t.Fatalf("boleto = %v, %v", titulo.BoletoCanal, titulo.BoletoEnviadoEm)
	}
}

func TestCancelar(t *testing.T) {
	titulo := novoTituloDeTeste(t)

	esperarErro(t, titulo.Cancelar("  ", agora), domain.ErrValidacao, "motivo do cancelamento é obrigatório")

	titulo.RecebidoCentavos = 10
	esperarErro(t, titulo.Cancelar("lançado errado", agora), domain.ErrConflito, "com recebimento")

	titulo.RecebidoCentavos = 0
	if err := titulo.Cancelar("lançado errado", agora); err != nil {
		t.Fatalf("Cancelar falhou: %v", err)
	}
	if titulo.Status != domainreceber.StatusCancelado || *titulo.CanceladoMotivo != "lançado errado" {
		t.Fatalf("título = %+v", titulo)
	}
	esperarErro(t, titulo.Cancelar("de novo", agora), domain.ErrConflito, "cancelado")
}

func TestConferirVersao(t *testing.T) {
	titulo := novoTituloDeTeste(t)

	esperarErro(t, titulo.ConferirVersao(0), domain.ErrValidacao, "versão do título é obrigatória")
	esperarErro(t, titulo.ConferirVersao(2), domain.ErrConflito, "alterado por outra pessoa")
	if err := titulo.ConferirVersao(1); err != nil {
		t.Fatalf("ConferirVersao(1) = %v", err)
	}
}

func TestEnumsDoTitulo(t *testing.T) {
	if origem, err := domainreceber.NovaOrigem(" Pedido "); err != nil || origem != domainreceber.OrigemPedido || !origem.Gerada() {
		t.Fatalf("NovaOrigem = %q, %v", origem, err)
	}
	if _, err := domainreceber.NovaOrigem("boleto"); !errors.Is(err, domain.ErrValidacao) {
		t.Fatalf("origem inválida = %v", err)
	}
	if status, err := domainreceber.NovoStatus("EM_BANCO"); err != nil || status != domainreceber.StatusEmBanco {
		t.Fatalf("NovoStatus = %q, %v", status, err)
	}
	if _, err := domainreceber.NovoStatus("vencido"); !errors.Is(err, domain.ErrValidacao) {
		t.Fatalf("vencido não é status gravado: %v", err)
	}
	if canal, err := domainreceber.NovoCanalBoleto("Email"); err != nil || canal != domainreceber.CanalBoletoEmail {
		t.Fatalf("NovoCanalBoleto = %q, %v", canal, err)
	}
	esperarErro(t, func() error { _, err := domainreceber.NovoCanalBoleto(""); return err }(), domain.ErrValidacao, "obrigatório")
	esperarErro(t, func() error { _, err := domainreceber.NovoCanalBoleto("fax"); return err }(), domain.ErrValidacao, "deve ser")
	esperarErro(t, func() error { _, err := domainreceber.NovaCompetencia("10/2026"); return err }(), domain.ErrValidacao, "AAAA-MM")
}
