package caixa_test

import (
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincaixa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/caixa"
	"github.com/google/uuid"
)

var agoraDoTeste = time.Date(2026, 10, 16, 2, 30, 0, 0, time.UTC)

func TestNovaAberturaValidaDataDeCorte(t *testing.T) {
	contaID, usuarioID := uuid.New(), uuid.New()

	abertura, err := domaincaixa.NovaAbertura(domaincaixa.DadosAbertura{
		ContaBancariaID: contaID,
		DataCorte:       time.Date(2026, 9, 30, 15, 0, 0, 0, time.UTC),
		SaldoCentavos:   -2500,
		UsuarioID:       usuarioID,
	}, agoraDoTeste)
	if err != nil {
		t.Fatalf("NovaAbertura = %v", err)
	}
	if !abertura.DataCorte.Equal(data(2026, 9, 30)) || abertura.SaldoCentavos != -2500 || abertura.Versao != 1 {
		t.Errorf("abertura = %+v", abertura)
	}
	if abertura.Abrange(data(2026, 9, 30)) || !abertura.Abrange(data(2026, 10, 1)) {
		t.Error("a abertura vale do dia seguinte ao corte em diante")
	}

	invalidas := map[string]domaincaixa.DadosAbertura{
		"sem conta":        {DataCorte: data(2026, 9, 30), UsuarioID: usuarioID},
		"sem usuário":      {ContaBancariaID: contaID, DataCorte: data(2026, 9, 30)},
		"sem data":         {ContaBancariaID: contaID, UsuarioID: usuarioID},
		"hoje em Brasília": {ContaBancariaID: contaID, UsuarioID: usuarioID, DataCorte: data(2026, 10, 15)},
		"no futuro":        {ContaBancariaID: contaID, UsuarioID: usuarioID, DataCorte: data(2026, 11, 1)},
	}
	for nome, dados := range invalidas {
		if _, err := domaincaixa.NovaAbertura(dados, agoraDoTeste); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("%s: erro = %v, esperado erro de validação", nome, err)
		}
	}

	if _, err := domaincaixa.NovaAbertura(domaincaixa.DadosAbertura{
		ContaBancariaID: contaID, UsuarioID: usuarioID, DataCorte: data(2026, 10, 14),
	}, agoraDoTeste); err != nil {
		t.Errorf("ontem em Brasília = %v", err)
	}
}

func TestAberturaRedefinidaComVersao(t *testing.T) {
	abertura, err := domaincaixa.NovaAbertura(domaincaixa.DadosAbertura{
		ContaBancariaID: uuid.New(), DataCorte: data(2026, 9, 30), SaldoCentavos: 100, UsuarioID: uuid.New(),
	}, agoraDoTeste)
	if err != nil {
		t.Fatalf("NovaAbertura = %v", err)
	}

	if err := abertura.ConferirVersao(0); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("versão ausente = %v", err)
	}
	if err := abertura.ConferirVersao(2); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("versão divergente = %v", err)
	}
	if err := abertura.ConferirVersao(1); err != nil {
		t.Errorf("versão atual = %v", err)
	}

	outro := uuid.New()
	depois := agoraDoTeste.Add(time.Hour)
	if err := abertura.Redefinir(data(2026, 8, 31), 900, outro, depois); err != nil {
		t.Fatalf("Redefinir = %v", err)
	}
	if !abertura.DataCorte.Equal(data(2026, 8, 31)) || abertura.SaldoCentavos != 900 || abertura.UsuarioID != outro || !abertura.AtualizadoEm.Equal(depois) {
		t.Errorf("redefinida = %+v", abertura)
	}
	if err := abertura.Redefinir(data(2026, 10, 20), 1, outro, depois); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("corte no futuro = %v", err)
	}
}
