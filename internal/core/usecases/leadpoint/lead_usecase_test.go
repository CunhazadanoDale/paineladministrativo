package leadpoint_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/usecases/leadpoint"
	"github.com/google/uuid"
)

type cenario struct {
	ctx     context.Context
	leads   *repositorioLeads
	etapas  *repositorioEtapas
	usecase *leadpoint.LeadUsecaseImpl
	funilID uuid.UUID
}

func novoCenario() *cenario {
	leads := novoRepositorioLeads()
	etapas := novoRepositorioEtapas()

	return &cenario{
		ctx:     context.Background(),
		leads:   leads,
		etapas:  etapas,
		usecase: leadpoint.NewLeadUsecase(leads, etapas),
		funilID: uuid.New(),
	}
}

func (c *cenario) novaEtapa(funilID uuid.UUID, ativo bool) uuid.UUID {
	id := uuid.New()
	c.etapas.itens[id] = &lead.Etapa{EtapaID: id, Nome: "Etapa", Ordem: len(c.etapas.itens) + 1, FunilID: funilID, Ativo: ativo}

	return id
}

func (c *cenario) novoLead(t *testing.T, etapaID uuid.UUID) uuid.UUID {
	t.Helper()

	id, err := c.usecase.Create(c.ctx, &lead.Lead{Nome: "Ana Souza", Email: "ana@exemplo.com", EtapaID: etapaID})
	if err != nil {
		t.Fatalf("criação do lead falhou: %v", err)
	}

	return id
}

func TestCriarLeadValidaCadastro(t *testing.T) {
	c := novoCenario()
	etapaID := c.novaEtapa(c.funilID, true)

	casos := []struct {
		nome string
		lead lead.Lead
	}{
		{"sem nome", lead.Lead{Nome: "  ", EtapaID: etapaID}},
		{"nome longo", lead.Lead{Nome: strings.Repeat("a", 201), EtapaID: etapaID}},
		{"email inválido", lead.Lead{Nome: "Ana", Email: "ana.exemplo.com", EtapaID: etapaID}},
		{"telefone longo", lead.Lead{Nome: "Ana", Telefone: strings.Repeat("9", 41), EtapaID: etapaID}},
		{"origem longa", lead.Lead{Nome: "Ana", Origem: strings.Repeat("x", 81), EtapaID: etapaID}},
		{"sem etapa", lead.Lead{Nome: "Ana"}},
		{"etapa inexistente", lead.Lead{Nome: "Ana", EtapaID: uuid.New()}},
		{"etapa inativa", lead.Lead{Nome: "Ana", EtapaID: c.novaEtapa(c.funilID, false)}},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			item := caso.lead
			if _, err := c.usecase.Create(c.ctx, &item); !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("erro %v, esperado erro de validação", err)
			}
		})
	}
}

func TestCriarLeadNormalizaEAceitaNomeComAcentosNoLimite(t *testing.T) {
	c := novoCenario()
	etapaID := c.novaEtapa(c.funilID, true)

	id, err := c.usecase.Create(c.ctx, &lead.Lead{
		Nome:    strings.Repeat("ç", 200),
		Email:   "  Ana@Exemplo.com ",
		EtapaID: etapaID,
	})
	if err != nil {
		t.Fatalf("criação falhou: %v", err)
	}

	salvo := c.leads.itens[id]
	if salvo.Email != "ana@exemplo.com" || !salvo.Ativo {
		t.Errorf("lead salvo %+v, esperado email normalizado e ativo", salvo)
	}
}

func TestAtualizarLeadPreservaEtapaECriacao(t *testing.T) {
	c := novoCenario()
	etapaID := c.novaEtapa(c.funilID, true)
	id := c.novoLead(t, etapaID)
	criadoEm := c.leads.itens[id].CriadoEm

	if err := c.usecase.Update(c.ctx, &lead.Lead{ID: id, Nome: "Ana Lima", Ativo: true}); err != nil {
		t.Fatalf("atualização falhou: %v", err)
	}

	salvo := c.leads.itens[id]
	if salvo.Nome != "Ana Lima" || salvo.EtapaID != etapaID || !salvo.CriadoEm.Equal(criadoEm) {
		t.Errorf("lead salvo %+v, esperado nome novo com etapa e criação preservadas", salvo)
	}
}

func TestAtualizarLeadRecusaTrocaDeEtapa(t *testing.T) {
	c := novoCenario()
	etapaID := c.novaEtapa(c.funilID, true)
	outraID := c.novaEtapa(c.funilID, true)
	id := c.novoLead(t, etapaID)

	if err := c.usecase.Update(c.ctx, &lead.Lead{ID: id, Nome: "Ana", Ativo: true, EtapaID: outraID}); !errors.Is(err, domain.ErrValidacao) {
		t.Errorf("erro %v, esperado erro de validação", err)
	}
	if c.leads.itens[id].EtapaID != etapaID {
		t.Error("o PUT trocou a etapa do lead")
	}
	if err := c.usecase.Update(c.ctx, &lead.Lead{ID: id, Nome: "Ana", Ativo: true, EtapaID: etapaID}); err != nil {
		t.Errorf("atualização com a mesma etapa falhou: %v", err)
	}
}

func TestMoverLeadGravaMovimentacao(t *testing.T) {
	c := novoCenario()
	origem := c.novaEtapa(c.funilID, true)
	destino := c.novaEtapa(c.funilID, true)
	id := c.novoLead(t, origem)

	if err := c.usecase.UpdateEtapa(c.ctx, id, destino); err != nil {
		t.Fatalf("movimentação falhou: %v", err)
	}

	if c.leads.itens[id].EtapaID != destino {
		t.Error("lead não mudou de etapa")
	}
	if len(c.leads.movimentacoes) != 1 || c.leads.movimentacoes[0] != (movimentacao{leadID: id, anterior: origem, atual: destino}) {
		t.Errorf("movimentações %+v, esperada uma de origem para destino", c.leads.movimentacoes)
	}
}

func TestMoverLeadParaAMesmaEtapaNaoGravaHistorico(t *testing.T) {
	c := novoCenario()
	etapaID := c.novaEtapa(c.funilID, true)
	id := c.novoLead(t, etapaID)

	if err := c.usecase.UpdateEtapa(c.ctx, id, etapaID); err != nil {
		t.Fatalf("movimentação falhou: %v", err)
	}
	if len(c.leads.movimentacoes) != 0 {
		t.Errorf("%d movimentações, esperado nenhuma", len(c.leads.movimentacoes))
	}
}

func TestMoverLeadValidaEtapaDeDestino(t *testing.T) {
	c := novoCenario()
	origem := c.novaEtapa(c.funilID, true)
	id := c.novoLead(t, origem)

	casos := []struct {
		nome    string
		destino uuid.UUID
	}{
		{"inexistente", uuid.New()},
		{"inativa", c.novaEtapa(c.funilID, false)},
		{"de outro funil", c.novaEtapa(uuid.New(), true)},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			if err := c.usecase.UpdateEtapa(c.ctx, id, caso.destino); !errors.Is(err, domain.ErrValidacao) {
				t.Errorf("erro %v, esperado erro de validação", err)
			}
		})
	}

	if c.leads.itens[id].EtapaID != origem || len(c.leads.movimentacoes) != 0 {
		t.Error("movimentação inválida alterou o lead")
	}
}

func TestOperacoesComLeadInexistente(t *testing.T) {
	c := novoCenario()
	etapaID := c.novaEtapa(c.funilID, true)
	id := uuid.New()

	if _, err := c.usecase.GetByID(c.ctx, id); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("consulta = %v, esperado não encontrado", err)
	}
	if err := c.usecase.Update(c.ctx, &lead.Lead{ID: id, Nome: "Ana"}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("atualização = %v, esperado não encontrado", err)
	}
	if err := c.usecase.UpdateEtapa(c.ctx, id, etapaID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("movimentação = %v, esperado não encontrado", err)
	}
	if err := c.usecase.Delete(c.ctx, id); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("remoção = %v, esperado não encontrado", err)
	}
}
