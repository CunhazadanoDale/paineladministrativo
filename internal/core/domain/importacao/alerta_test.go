package importacao_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	"github.com/google/uuid"
)

func TestNovaFolga(t *testing.T) {
	if folga, err := domainimportacao.NovaFolga(nil); err != nil || folga != 10 {
		t.Errorf("padrão = %d (%v)", folga, err)
	}
	for _, valor := range []int{0, 90} {
		if folga, err := domainimportacao.NovaFolga(&valor); err != nil || folga != valor {
			t.Errorf("folga %d = %d (%v)", valor, folga, err)
		}
	}
	for _, valor := range []int{-1, 91} {
		if _, err := domainimportacao.NovaFolga(&valor); !errors.Is(err, domain.ErrValidacao) {
			t.Errorf("folga %d: erro = %v", valor, err)
		}
	}
}

func TestClassificarAlertas(t *testing.T) {
	chegada := dia(2026, 11, 20)
	prazoApertado := dia(2026, 11, 25)
	prazoFolgado := dia(2026, 12, 31)
	exato := dia(2026, 11, 30)

	emRisco := domainimportacao.AlocacaoAberta{PedidoItemID: uuid.New(), ChegadaPrevista: &chegada, PrazoLimite: &prazoApertado}
	tranquila := domainimportacao.AlocacaoAberta{PedidoItemID: uuid.New(), ChegadaPrevista: &chegada, PrazoLimite: &prazoFolgado}
	noLimite := domainimportacao.AlocacaoAberta{PedidoItemID: uuid.New(), ChegadaPrevista: &chegada, PrazoLimite: &exato}
	semPrazo := domainimportacao.AlocacaoAberta{PedidoItemID: uuid.New(), ChegadaPrevista: &chegada}
	semPrevisao := domainimportacao.AlocacaoAberta{PedidoItemID: uuid.New(), PrazoLimite: &prazoApertado}

	descoberto := domainimportacao.PedidoSemImportacao{PedidoItemID: uuid.New(), Restante: 100, Reservado: 20, Alocado: 50}
	coberto := domainimportacao.PedidoSemImportacao{PedidoItemID: uuid.New(), Restante: 100, Reservado: 60, Alocado: 50}

	alertas := domainimportacao.ClassificarAlertas(10,
		[]domainimportacao.AlocacaoAberta{emRisco, tranquila, noLimite, semPrazo, semPrevisao},
		[]domainimportacao.PedidoSemImportacao{descoberto, coberto})

	if len(alertas.PrazoEmRisco) != 1 || alertas.PrazoEmRisco[0].PedidoItemID != emRisco.PedidoItemID ||
		alertas.PrazoEmRisco[0].DiasDeAtraso != 5 || !alertas.PrazoEmRisco[0].ChegadaComFolga.Equal(dia(2026, 11, 30)) {
		t.Errorf("prazo em risco = %+v", alertas.PrazoEmRisco)
	}
	if len(alertas.SemPrevisao) != 1 || alertas.SemPrevisao[0].PedidoItemID != semPrevisao.PedidoItemID {
		t.Errorf("sem previsão = %+v", alertas.SemPrevisao)
	}
	if len(alertas.SemImportacao) != 1 || alertas.SemImportacao[0].SemCobertura() != 30 {
		t.Errorf("sem importação = %+v", alertas.SemImportacao)
	}

	semFolga := domainimportacao.ClassificarAlertas(0, []domainimportacao.AlocacaoAberta{emRisco}, nil)
	if len(semFolga.PrazoEmRisco) != 0 {
		t.Errorf("sem folga a chegada é antes do prazo: %+v", semFolga.PrazoEmRisco)
	}
}
