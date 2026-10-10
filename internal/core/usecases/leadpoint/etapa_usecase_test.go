package leadpoint_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/leadpoint"
	"github.com/google/uuid"
)

func TestReordenarEtapasExigeTodasAsEtapasSemRepeticao(t *testing.T) {
	c := novoCenario()
	usecase := leadpoint.NewEtapaUsecase(c.etapas)
	primeira := c.etapas.itens[c.novaEtapa(c.funilID, true)]
	segunda := c.etapas.itens[c.novaEtapa(c.funilID, true)]
	terceira := c.etapas.itens[c.novaEtapa(c.funilID, true)]

	casos := []struct {
		nome   string
		etapas []*lead.Etapa
	}{
		{"lista parcial", []*lead.Etapa{segunda, primeira}},
		{"etapa repetida", []*lead.Etapa{segunda, segunda, primeira}},
		{"etapa desconhecida", []*lead.Etapa{segunda, primeira, {EtapaID: uuid.New(), FunilID: c.funilID}}},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			if err := usecase.Reordenar(c.ctx, c.funilID, caso.etapas); !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("erro %v, esperado erro de validação", err)
			}
		})
	}

	if err := usecase.Reordenar(c.ctx, c.funilID, []*lead.Etapa{terceira, primeira, segunda}); err != nil {
		t.Fatalf("reordenação completa falhou: %v", err)
	}
	if c.etapas.itens[terceira.EtapaID].Ordem != 1 || c.etapas.itens[segunda.EtapaID].Ordem != 3 {
		t.Error("reordenação completa não aplicou a nova ordem")
	}
}
