package caixa_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	"github.com/google/uuid"
)

func aberturaEm(t *testing.T, contaID uuid.UUID, corte time.Time) *domaincaixa.Abertura {
	t.Helper()

	abertura, err := domaincaixa.NovaAbertura(domaincaixa.DadosAbertura{
		ContaBancariaID: contaID, DataCorte: corte, SaldoCentavos: 1000, UsuarioID: uuid.New(),
	}, agoraDoTeste)
	if err != nil {
		t.Fatalf("NovaAbertura = %v", err)
	}

	return abertura
}

func preparar(t *testing.T, extrato *domaincaixa.ExtratoOFX, abertura *domaincaixa.Abertura) *domaincaixa.ExtratoPreparado {
	t.Helper()

	preparado, err := domaincaixa.PrepararImportacao(domaincaixa.DadosImportacao{
		Extrato:         extrato,
		ContaBancariaID: abertura.ContaBancariaID,
		ArquivoID:       uuid.New(),
		UsuarioID:       uuid.New(),
		Abertura:        abertura,
		Agora:           agoraDoTeste,
	})
	if err != nil {
		t.Fatalf("PrepararImportacao = %v", err)
	}

	return preparado
}

func TestPrepararImportacaoSeparaLinhasIgnoradas(t *testing.T) {
	contaID := uuid.New()
	abertura := aberturaEm(t, contaID, data(2026, 9, 30))
	extrato := &domaincaixa.ExtratoOFX{
		Saldo: &domaincaixa.SaldoInformado{Centavos: 5000, Em: time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)},
		Transacoes: []domaincaixa.TransacaoOFX{
			{FITID: "1", Data: data(2026, 9, 30), ValorCentavos: -100, Descricao: "antes do corte"},
			{FITID: "2", Data: data(2026, 10, 1), ValorCentavos: -200, Descricao: "  tarifa   pacote  ", Documento: " 77 "},
			{FITID: "3", Data: data(2026, 10, 2), ValorCentavos: 0, Descricao: "informativo"},
			{FITID: "4", Data: data(2026, 10, 15), ValorCentavos: 300, Descricao: ""},
			{FITID: "5", Data: data(2026, 10, 16), ValorCentavos: 400, Descricao: "agendado"},
		},
	}

	preparado := preparar(t, extrato, abertura)

	importacao := preparado.Importacao
	if !importacao.PeriodoInicio.Equal(data(2026, 9, 30)) || !importacao.PeriodoFim.Equal(data(2026, 10, 16)) {
		t.Errorf("período = %v a %v", importacao.PeriodoInicio, importacao.PeriodoFim)
	}
	if importacao.SaldoBancoCentavos == nil || *importacao.SaldoBancoCentavos != 5000 || !importacao.SaldoBancoEm.Equal(data(2026, 10, 15)) {
		t.Errorf("saldo do banco = %v em %v", importacao.SaldoBancoCentavos, importacao.SaldoBancoEm)
	}
	if preparado.Total() != 5 || len(preparado.Lancamentos) != 2 || importacao.LinhasIgnoradas != 3 {
		t.Fatalf("total %d, lançamentos %d, ignoradas %d", preparado.Total(), len(preparado.Lancamentos), importacao.LinhasIgnoradas)
	}

	motivos := map[int]domaincaixa.MotivoIgnorada{}
	for _, ignorada := range preparado.Ignoradas {
		motivos[ignorada.Sequencia] = ignorada.Motivo
	}
	esperados := map[int]domaincaixa.MotivoIgnorada{
		1: domaincaixa.MotivoAnteriorAoCorte,
		3: domaincaixa.MotivoValorZero,
		5: domaincaixa.MotivoDataFutura,
	}
	for sequencia, motivo := range esperados {
		if motivos[sequencia] != motivo {
			t.Errorf("linha %d ignorada por %q, esperado %q", sequencia, motivos[sequencia], motivo)
		}
	}

	tarifa, credito := preparado.Lancamentos[0], preparado.Lancamentos[1]
	if tarifa.Sequencia != 2 || tarifa.Descricao != "tarifa pacote" || tarifa.Documento == nil || *tarifa.Documento != "77" || tarifa.Credito() {
		t.Errorf("tarifa = %+v", tarifa)
	}
	if credito.Descricao != "Sem descrição" || credito.Documento != nil || !credito.Credito() {
		t.Errorf("crédito = %+v", credito)
	}
	if tarifa.ImportacaoID != importacao.ID || tarifa.ContaBancariaID != contaID || len(tarifa.Identificador) != 64 {
		t.Errorf("vínculos da tarifa = %+v", tarifa)
	}

	preparado.RegistrarNovas(1)
	if importacao.LinhasNovas != 1 || importacao.LinhasRepetidas != 1 {
		t.Errorf("novas %d, repetidas %d", importacao.LinhasNovas, importacao.LinhasRepetidas)
	}
}

func TestIdentificadoresSaoEstaveisEDistinguemRepeticoes(t *testing.T) {
	abertura := aberturaEm(t, uuid.New(), data(2026, 9, 30))
	extrato := &domaincaixa.ExtratoOFX{Transacoes: []domaincaixa.TransacaoOFX{
		{FITID: "X", Data: data(2026, 10, 1), ValorCentavos: -100, Descricao: "a"},
		{FITID: "X", Data: data(2026, 10, 1), ValorCentavos: -100, Descricao: "b"},
		{FITID: "X", Data: data(2026, 10, 1), ValorCentavos: -101, Descricao: "c"},
		{Data: data(2026, 10, 1), ValorCentavos: -100, Descricao: "sem fitid"},
		{Data: data(2026, 10, 1), ValorCentavos: -100, Descricao: "outra sem fitid"},
	}}

	primeira := preparar(t, extrato, abertura).Identificadores()
	segunda := preparar(t, extrato, abertura).Identificadores()

	vistos := map[string]bool{}
	for i, identificador := range primeira {
		if identificador != segunda[i] {
			t.Errorf("linha %d mudou de identificador entre leituras", i+1)
		}
		if vistos[identificador] {
			t.Errorf("linha %d repete identificador", i+1)
		}
		vistos[identificador] = true
	}
}

func TestPrepararImportacaoRecusaDadosIncompletos(t *testing.T) {
	contaID := uuid.New()
	abertura := aberturaEm(t, contaID, data(2026, 9, 30))
	inicio, fim := data(2026, 10, 10), data(2026, 10, 1)

	casos := map[string]domaincaixa.DadosImportacao{
		"sem extrato":       {ContaBancariaID: contaID, ArquivoID: uuid.New(), UsuarioID: uuid.New(), Abertura: abertura},
		"sem abertura":      {Extrato: &domaincaixa.ExtratoOFX{}, ContaBancariaID: contaID, ArquivoID: uuid.New(), UsuarioID: uuid.New()},
		"abertura de outra": {Extrato: &domaincaixa.ExtratoOFX{}, ContaBancariaID: uuid.New(), ArquivoID: uuid.New(), UsuarioID: uuid.New(), Abertura: abertura},
		"sem período":       {Extrato: &domaincaixa.ExtratoOFX{}, ContaBancariaID: contaID, ArquivoID: uuid.New(), UsuarioID: uuid.New(), Abertura: abertura},
		"período invertido": {Extrato: &domaincaixa.ExtratoOFX{Inicio: &inicio, Fim: &fim}, ContaBancariaID: contaID, ArquivoID: uuid.New(), UsuarioID: uuid.New(), Abertura: abertura},
	}

	for nome, dados := range casos {
		dados.Agora = agoraDoTeste
		if _, err := domaincaixa.PrepararImportacao(dados); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("%s: erro = %v, esperado erro de validação", nome, err)
		}
	}
}

func TestPrepararImportacaoLimitaTextos(t *testing.T) {
	abertura := aberturaEm(t, uuid.New(), data(2026, 9, 30))
	extrato := &domaincaixa.ExtratoOFX{Transacoes: []domaincaixa.TransacaoOFX{{
		FITID: "1", Data: data(2026, 10, 1), ValorCentavos: 1,
		Descricao: strings.Repeat("é", 300), Documento: strings.Repeat("9", 80),
	}}}

	lancamento := preparar(t, extrato, abertura).Lancamentos[0]
	if len([]rune(lancamento.Descricao)) != 255 || len(*lancamento.Documento) != 60 {
		t.Errorf("descrição com %d caracteres, documento com %d", len([]rune(lancamento.Descricao)), len(*lancamento.Documento))
	}
}

func TestSaldoDaConta(t *testing.T) {
	semAbertura := &domaincaixa.SaldoConta{MovimentoCentavos: 10}
	if semAbertura.SaldoAtualCentavos() != nil || semAbertura.DiferencaDoBancoCentavos() != nil {
		t.Error("conta sem abertura não tem saldo nem diferença")
	}

	corte, saldoInicial := data(2026, 9, 30), int64(1000)
	bancoEm, saldoBanco := data(2026, 10, 10), int64(1450)
	saldo := &domaincaixa.SaldoConta{
		DataCorte:                        &corte,
		SaldoInicialCentavos:             &saldoInicial,
		MovimentoCentavos:                700,
		SaldoBancoCentavos:               &saldoBanco,
		SaldoBancoEm:                     &bancoEm,
		MovimentoAteSaldoDoBancoCentavos: 500,
	}
	if atual := saldo.SaldoAtualCentavos(); atual == nil || *atual != 1700 {
		t.Errorf("saldo atual = %v", atual)
	}
	if diferenca := saldo.DiferencaDoBancoCentavos(); diferenca == nil || *diferenca != -50 {
		t.Errorf("diferença = %v", diferenca)
	}

	antes := data(2026, 9, 1)
	saldo.SaldoBancoEm = &antes
	if saldo.DiferencaDoBancoCentavos() != nil {
		t.Error("saldo do banco anterior ao corte não é comparável")
	}
}
