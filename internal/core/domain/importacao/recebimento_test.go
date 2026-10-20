package importacao_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	"github.com/google/uuid"
)

func desembaracada(t *testing.T) *domainimportacao.Importacao {
	t.Helper()

	importacao := embarcada(t)
	if err := importacao.Desembaracar(domainimportacao.DesembaracoInput{
		ChegadaEm: dia(2026, 10, 10), DesembaracadaEm: dia(2026, 10, 12), Declaracao: "26/0001234-5",
	}, agora); err != nil {
		t.Fatalf("Desembaracar: %v", err)
	}

	return importacao
}

func TestReceberGravaQuantidadesECusto(t *testing.T) {
	importacao := desembaracada(t)
	porcelanato, bonificado := importacao.Itens[0], importacao.Itens[1]
	localID := uuid.New()
	cambio := int64(5_000_000)

	calculo, err := importacao.Receber(domainimportacao.RecebimentoInput{
		RecebidaEm: dia(2026, 10, 13), LocalID: localID, CambioMicro: &cambio,
		Recebidas: map[uuid.UUID]int{porcelanato.ID: 900},
	}, agora)
	if err != nil {
		t.Fatalf("Receber: %v", err)
	}

	if importacao.Status != domainimportacao.StatusRecebida || !importacao.RecebidaEm.Equal(dia(2026, 10, 13)) ||
		*importacao.LocalRecebimentoID != localID || importacao.Cambio.Micro() != cambio {
		t.Errorf("importação recebida = %+v", importacao)
	}
	if *porcelanato.QuantidadeRecebida != 900 || *porcelanato.CustoUnitarioCentavos != 2500 {
		t.Errorf("porcelanato = recebido %d, custo %d (esperado 22500,00 ÷ 900 = 25,00)", *porcelanato.QuantidadeRecebida, *porcelanato.CustoUnitarioCentavos)
	}
	if *bonificado.QuantidadeRecebida != 200 || *bonificado.CustoUnitarioCentavos != 0 {
		t.Errorf("bonificado = recebido %d, custo %d", *bonificado.QuantidadeRecebida, *bonificado.CustoUnitarioCentavos)
	}
	if calculo.ValorMercadoriaCentavos != 2250000 {
		t.Errorf("mercadoria = %d", calculo.ValorMercadoriaCentavos)
	}

	if _, err := importacao.Receber(domainimportacao.RecebimentoInput{RecebidaEm: dia(2026, 10, 13), LocalID: localID, CambioMicro: &cambio}, agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("receber de novo: erro = %v", err)
	}
	if err := importacao.Cancelar("tarde demais", agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("cancelar recebida: erro = %v", err)
	}
	if _, err := importacao.LancarCusto(uuid.New(), custoDe(domainimportacao.TipoCustoArmazenagem, 1000, true), agora); err != nil {
		t.Fatalf("custo posterior: %v", err)
	}
	if posterior := importacao.Custos[0]; !posterior.Posterior || posterior.CompoeCusto {
		t.Errorf("custo depois do recebimento = %+v", posterior)
	}
}

func TestReceberValida(t *testing.T) {
	cambio := int64(5_000_000)
	casos := []struct {
		nome  string
		input func(*domainimportacao.Importacao) domainimportacao.RecebimentoInput
	}{
		{"sem local", func(*domainimportacao.Importacao) domainimportacao.RecebimentoInput {
			return domainimportacao.RecebimentoInput{RecebidaEm: dia(2026, 10, 13), CambioMicro: &cambio}
		}},
		{"antes do desembaraço", func(*domainimportacao.Importacao) domainimportacao.RecebimentoInput {
			return domainimportacao.RecebimentoInput{RecebidaEm: dia(2026, 10, 11), LocalID: uuid.New(), CambioMicro: &cambio}
		}},
		{"no futuro", func(*domainimportacao.Importacao) domainimportacao.RecebimentoInput {
			return domainimportacao.RecebimentoInput{RecebidaEm: dia(2026, 10, 15), LocalID: uuid.New(), CambioMicro: &cambio}
		}},
		{"sem câmbio", func(*domainimportacao.Importacao) domainimportacao.RecebimentoInput {
			return domainimportacao.RecebimentoInput{RecebidaEm: dia(2026, 10, 13), LocalID: uuid.New()}
		}},
		{"acima da quantidade", func(i *domainimportacao.Importacao) domainimportacao.RecebimentoInput {
			return domainimportacao.RecebimentoInput{RecebidaEm: dia(2026, 10, 13), LocalID: uuid.New(), CambioMicro: &cambio,
				Recebidas: map[uuid.UUID]int{i.Itens[0].ID: 1001}}
		}},
		{"nada recebido", func(i *domainimportacao.Importacao) domainimportacao.RecebimentoInput {
			return domainimportacao.RecebimentoInput{RecebidaEm: dia(2026, 10, 13), LocalID: uuid.New(), CambioMicro: &cambio,
				Recebidas: map[uuid.UUID]int{i.Itens[0].ID: 0, i.Itens[1].ID: 0}}
		}},
		{"item de outra importação", func(*domainimportacao.Importacao) domainimportacao.RecebimentoInput {
			return domainimportacao.RecebimentoInput{RecebidaEm: dia(2026, 10, 13), LocalID: uuid.New(), CambioMicro: &cambio,
				Recebidas: map[uuid.UUID]int{uuid.New(): 1}}
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			importacao := desembaracada(t)
			if _, err := importacao.Receber(caso.input(importacao), agora); !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("erro = %v", err)
			}
			if importacao.Status != domainimportacao.StatusDesembaracada {
				t.Errorf("status mudou com erro: %q", importacao.Status)
			}
		})
	}

	if _, err := embarcada(t).Receber(domainimportacao.RecebimentoInput{RecebidaEm: dia(2026, 10, 13), LocalID: uuid.New(), CambioMicro: &cambio}, agora); !errors.Is(err, domain.ErrConflito) {
		t.Errorf("receber embarcada: erro = %v", err)
	}
}
