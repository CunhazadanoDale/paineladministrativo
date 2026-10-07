package solicitacao_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
)

func TestTransicoesDeStatus(t *testing.T) {
	casos := []struct {
		de        domainsolicitacao.Status
		para      domainsolicitacao.Status
		permitido bool
	}{
		{domainsolicitacao.StatusPendenteAprovacao, domainsolicitacao.StatusAprovado, true},
		{domainsolicitacao.StatusPendenteAprovacao, domainsolicitacao.StatusRejeitado, true},
		{domainsolicitacao.StatusPendenteAprovacao, domainsolicitacao.StatusCancelado, true},
		{domainsolicitacao.StatusPendenteAprovacao, domainsolicitacao.StatusPago, false},
		{domainsolicitacao.StatusAprovado, domainsolicitacao.StatusPago, true},
		{domainsolicitacao.StatusAprovado, domainsolicitacao.StatusCancelado, true},
		{domainsolicitacao.StatusAprovado, domainsolicitacao.StatusPendenteAprovacao, false},
		{domainsolicitacao.StatusAprovado, domainsolicitacao.StatusRejeitado, false},
		{domainsolicitacao.StatusPago, domainsolicitacao.StatusCancelado, false},
		{domainsolicitacao.StatusPago, domainsolicitacao.StatusAprovado, false},
		{domainsolicitacao.StatusRejeitado, domainsolicitacao.StatusAprovado, false},
		{domainsolicitacao.StatusRejeitado, domainsolicitacao.StatusCancelado, false},
		{domainsolicitacao.StatusCancelado, domainsolicitacao.StatusPendenteAprovacao, false},
	}

	for _, caso := range casos {
		if obtido := caso.de.PodeTransicionarPara(caso.para); obtido != caso.permitido {
			t.Errorf("%s -> %s = %t, esperado %t", caso.de, caso.para, obtido, caso.permitido)
		}
	}
}

func TestNovoStatusRejeitaValorDesconhecido(t *testing.T) {
	if _, err := domainsolicitacao.NovoStatus("aguardando_pagamento"); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("NovoStatus inesperado: %v", err)
	}

	status, err := domainsolicitacao.NovoStatus("pago")
	if err != nil {
		t.Fatalf("NovoStatus(pago) falhou: %v", err)
	}
	if status != domainsolicitacao.StatusPago {
		t.Errorf("status = %q, esperado %q", status, domainsolicitacao.StatusPago)
	}
}
