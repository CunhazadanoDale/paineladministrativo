package lead_test

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainlead "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	leadusecases "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/lead"
	"github.com/google/uuid"
)

func TestReordenarEtapasExigeTodasAsEtapasSemRepeticao(t *testing.T) {
	c := novoCenario()
	usecase := leadusecases.NewEtapaUsecase(c.etapas)
	primeira := c.etapas.itens[c.novaEtapa(c.funilID, true)]
	segunda := c.etapas.itens[c.novaEtapa(c.funilID, true)]
	terceira := c.etapas.itens[c.novaEtapa(c.funilID, true)]

	casos := []struct {
		nome   string
		etapas []*domainlead.Etapa
	}{
		{"lista parcial", []*domainlead.Etapa{segunda, primeira}},
		{"etapa repetida", []*domainlead.Etapa{segunda, segunda, primeira}},
		{"etapa desconhecida", []*domainlead.Etapa{segunda, primeira, {EtapaID: uuid.New(), FunilID: c.funilID}}},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			if err := usecase.Reordenar(c.ctx, c.funilID, caso.etapas); !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("erro %v, esperado erro de validação", err)
			}
		})
	}

	if err := usecase.Reordenar(c.ctx, c.funilID, []*domainlead.Etapa{terceira, primeira, segunda}); err != nil {
		t.Fatalf("reordenação completa falhou: %v", err)
	}
	if c.etapas.itens[terceira.EtapaID].Ordem != 1 || c.etapas.itens[segunda.EtapaID].Ordem != 3 {
		t.Error("reordenação completa não aplicou a nova ordem")
	}
}
