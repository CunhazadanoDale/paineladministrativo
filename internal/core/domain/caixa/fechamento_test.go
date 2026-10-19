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

func previaDe(pendentes, invalidadas int, saldoBanco *int64) *domaincaixa.PreviaFechamento {
	return &domaincaixa.PreviaFechamento{
		ContaBancariaID:      uuid.New(),
		Competencia:          data(2026, 9, 1),
		SaldoInicialCentavos: 100000,
		MovimentoDoMes: domaincaixa.MovimentoDoMes{
			EntradasCentavos: 50000, SaidasCentavos: 20000,
			LancamentosPendentes: pendentes, ConciliacoesInvalidadas: invalidadas, SaldoBancoCentavos: saldoBanco,
		},
	}
}

func centavos(valor int64) *int64 {
	return &valor
}

func TestNovoFechamentoConfereBloqueiosESaldoDoBanco(t *testing.T) {
	usuario := uuid.New()

	fechamento, err := domaincaixa.NovoFechamento(domaincaixa.DadosFechamento{
		Previa: previaDe(0, 0, centavos(130000)), UsuarioID: usuario, Agora: agoraDoTeste,
	})
	if err != nil {
		t.Fatalf("NovoFechamento = %v", err)
	}
	if fechamento.SaldoFinalCentavos != 130000 || fechamento.SaldoBancoCentavos != 130000 || !fechamento.Vigente ||
		!fechamento.Competencia.Equal(data(2026, 9, 1)) || fechamento.UsuarioID != usuario {
		t.Errorf("fechamento = %+v", fechamento)
	}

	if _, err := domaincaixa.NovoFechamento(domaincaixa.DadosFechamento{
		Previa: previaDe(0, 0, nil), SaldoBancoInformado: centavos(130000), UsuarioID: usuario, Agora: agoraDoTeste,
	}); err != nil {
		t.Errorf("saldo do banco informado = %v", err)
	}

	casos := []struct {
		nome      string
		dados     domaincaixa.DadosFechamento
		sentinela error
		trecho    string
	}{
		{"pendências", domaincaixa.DadosFechamento{Previa: previaDe(2, 1, centavos(130000)), UsuarioID: usuario}, domain.ErrConflito,
			"o mês tem 2 lançamento(s) pendente(s) e 1 conciliação(ões) invalidada(s)"},
		{"sem saldo do banco", domaincaixa.DadosFechamento{Previa: previaDe(0, 0, nil), UsuarioID: usuario}, domain.ErrValidacao,
			"informe o saldo do banco"},
		{"saldo diferente", domaincaixa.DadosFechamento{Previa: previaDe(0, 0, centavos(130050)), UsuarioID: usuario}, domain.ErrConflito,
			"saldo calculado (1300,00) diferente do saldo do banco (1300,50); diferença de 0,50"},
		{"informado vale mais que o extrato", domaincaixa.DadosFechamento{
			Previa: previaDe(0, 0, centavos(130000)), SaldoBancoInformado: centavos(1), UsuarioID: usuario,
		}, domain.ErrConflito, "diferente do saldo do banco"},
		{"sem usuário", domaincaixa.DadosFechamento{Previa: previaDe(0, 0, centavos(130000))}, domain.ErrValidacao, "usuário"},
	}
	for _, caso := range casos {
		_, err := domaincaixa.NovoFechamento(caso.dados)
		if !errors.Is(err, caso.sentinela) || !strings.Contains(err.Error(), caso.trecho) {
			t.Errorf("%s: erro = %v, esperado %v com %q", caso.nome, err, caso.sentinela, caso.trecho)
		}
	}
}

func TestReabrirFechamento(t *testing.T) {
	fechamento, err := domaincaixa.NovoFechamento(domaincaixa.DadosFechamento{
		Previa: previaDe(0, 0, centavos(130000)), UsuarioID: uuid.New(), Agora: agoraDoTeste,
	})
	if err != nil {
		t.Fatalf("NovoFechamento = %v", err)
	}

	usuario := uuid.New()
	if err := fechamento.Reabrir(usuario, " ", agoraDoTeste); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("sem motivo = %v", err)
	}
	if err := fechamento.Reabrir(usuario, " tarifa esquecida ", agoraDoTeste); err != nil {
		t.Fatalf("Reabrir = %v", err)
	}
	if fechamento.Vigente || *fechamento.MotivoReabertura != "tarifa esquecida" || *fechamento.ReabertoPor != usuario {
		t.Errorf("reaberto = %+v", fechamento)
	}
	if err := fechamento.Reabrir(usuario, "de novo", agoraDoTeste); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("reabrir de novo = %v", err)
	}
}

func TestSequenciaDosFechamentos(t *testing.T) {
	abertura := aberturaEm(t, uuid.New(), data(2026, 7, 31))
	hoje := data(2026, 10, 15)

	if primeira := abertura.PrimeiraCompetencia(); !primeira.Equal(data(2026, 8, 1)) {
		t.Errorf("primeira competência = %v", primeira)
	}
	if meio := aberturaEm(t, uuid.New(), data(2026, 8, 14)).PrimeiraCompetencia(); !meio.Equal(data(2026, 8, 1)) {
		t.Errorf("corte no meio do mês = %v", meio)
	}

	if err := domaincaixa.ConferirCompetencia(data(2026, 8, 20), abertura, nil, hoje, true); err != nil {
		t.Errorf("primeiro mês = %v", err)
	}
	agosto := &domaincaixa.Fechamento{Competencia: data(2026, 8, 1), SaldoFinalCentavos: 777, Vigente: true}
	if !domaincaixa.ProximaCompetencia(abertura, agosto).Equal(data(2026, 9, 1)) ||
		domaincaixa.SaldoInicialDaCompetencia(abertura, agosto) != 777 || domaincaixa.SaldoInicialDaCompetencia(abertura, nil) != 1000 {
		t.Error("sequência depois de agosto")
	}

	casos := []struct {
		nome        string
		competencia time.Time
		ultimo      *domaincaixa.Fechamento
		sentinela   error
		trecho      string
	}{
		{"antes do saldo inicial", data(2026, 7, 1), nil, domain.ErrValidacao, "anterior ao saldo inicial"},
		{"mês corrente", data(2026, 10, 1), &domaincaixa.Fechamento{Competencia: data(2026, 9, 1)}, domain.ErrValidacao, "mês encerrado"},
		{"pulando mês", data(2026, 9, 1), nil, domain.ErrConflito, "feche antes o mês de 08/2026"},
		{"já fechado", data(2026, 8, 1), agosto, domain.ErrConflito, "mês já fechado"},
	}
	for _, caso := range casos {
		err := domaincaixa.ConferirCompetencia(caso.competencia, abertura, caso.ultimo, hoje, true)
		if !errors.Is(err, caso.sentinela) || !strings.Contains(err.Error(), caso.trecho) {
			t.Errorf("%s: erro = %v, esperado %v com %q", caso.nome, err, caso.sentinela, caso.trecho)
		}
	}

	if err := domaincaixa.ConferirCompetencia(data(2026, 10, 1), abertura, &domaincaixa.Fechamento{Competencia: data(2026, 9, 1)}, hoje, false); err != nil {
		t.Errorf("prévia do mês corrente = %v", err)
	}
	if err := domaincaixa.ConferirCompetencia(data(2026, 8, 1), nil, nil, hoje, true); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("sem abertura = %v", err)
	}
	if fim := domaincaixa.FimDaCompetencia(data(2026, 2, 10)); !fim.Equal(data(2026, 2, 28)) {
		t.Errorf("fim de fevereiro = %v", fim)
	}
}

func TestImportacaoIgnoraMesFechado(t *testing.T) {
	abertura := aberturaEm(t, uuid.New(), data(2026, 8, 31))

	preparado, err := domaincaixa.PrepararImportacao(domaincaixa.DadosImportacao{
		Extrato: &domaincaixa.ExtratoOFX{Transacoes: []domaincaixa.TransacaoOFX{
			{FITID: "A", Data: data(2026, 9, 30), ValorCentavos: -10},
			{FITID: "B", Data: data(2026, 10, 1), ValorCentavos: -20},
		}},
		ContaBancariaID: abertura.ContaBancariaID, ArquivoID: uuid.New(), UsuarioID: uuid.New(), Abertura: abertura,
		MesesFechados: []time.Time{data(2026, 9, 1)}, Agora: agoraDoTeste,
	})
	if err != nil {
		t.Fatalf("PrepararImportacao = %v", err)
	}
	if len(preparado.Lancamentos) != 1 || preparado.Lancamentos[0].ValorCentavos != -20 {
		t.Errorf("lançamentos = %+v", preparado.Lancamentos)
	}
	if len(preparado.Ignoradas) != 1 || preparado.Ignoradas[0].Motivo != domaincaixa.MotivoMesFechado {
		t.Errorf("ignoradas = %+v", preparado.Ignoradas)
	}
}
