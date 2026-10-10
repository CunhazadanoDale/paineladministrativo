// Package postgres_test cobre os repositórios contra um Postgres de
// verdade: as queries, as transações e as constraints do schema. É a camada
// que não dá para testar com mock.
package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/postgres"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainlead "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	"github.com/CunhazadanoDale/paineladministrativo.git/test/helpers"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// novoLead monta um lead válido para o repositório persistir. Os campos de
// data são preenchidos aqui porque a tabela os exige com valor.
func novoLead(etapaID uuid.UUID, nome string) *domainlead.Lead {
	agora := time.Now().UTC()

	return &domainlead.Lead{
		ID:           uuid.New(),
		Nome:         nome,
		Email:        "ana@exemplo.com",
		Telefone:     "11999999999",
		Origem:       "teste",
		Ativo:        true,
		CriadoEm:     agora,
		AtualizadoEm: agora,
		EtapaID:      etapaID,
	}
}

// cenario devolve um repositório, o banco, o id do funil e as duas etapas
// criadas, já com o banco zerado.
func cenario(t *testing.T) (*postgres.LeadRepository, *sqlx.DB, uuid.UUID, []uuid.UUID) {
	t.Helper()

	banco := helpers.BancoDoTeste(t)

	funil := helpers.CriarFunil(t, banco, "Funil de teste")
	etapas := helpers.CriarEtapas(t, banco, funil, "Novo", "Proposta")

	return postgres.NewLeadRepository(banco), banco, funil, etapas
}

func TestLeadCriaEBuscaPorID(t *testing.T) {
	repo, _, _, etapas := cenario(t)
	ctx := context.Background()

	criado := novoLead(etapas[0], "Ana Souza")
	id, err := repo.Create(ctx, criado)
	if err != nil {
		t.Fatalf("criação do lead falhou: %v", err)
	}
	if id == uuid.Nil {
		t.Fatal("criação devolveu id zero")
	}

	salvo, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("busca por id falhou: %v", err)
	}
	if salvo == nil {
		t.Fatal("lead criado não foi encontrado")
	}
	if salvo.Nome != "Ana Souza" {
		t.Errorf("nome %q, esperado %q", salvo.Nome, "Ana Souza")
	}
	if salvo.EtapaID != etapas[0] {
		t.Errorf("etapa %s, esperada %s", salvo.EtapaID, etapas[0])
	}
	if !salvo.Ativo {
		t.Error("lead deveria ter sido criado ativo")
	}
}

func TestLeadInexistenteDevolveNil(t *testing.T) {
	repo, _, _, _ := cenario(t)

	salvo, err := repo.GetByID(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("busca por id inexistente falhou: %v", err)
	}
	if salvo != nil {
		t.Errorf("esperava nenhum lead, veio %+v", salvo)
	}
}

func TestListarAtivosPaginaEExcluiInativos(t *testing.T) {
	repo, banco, _, etapas := cenario(t)
	ctx := context.Background()

	ids := []uuid.UUID{
		helpers.CriarLead(t, banco, etapas[0], "Ana Souza"),
		helpers.CriarLead(t, banco, etapas[0], "Bruno Lima"),
		helpers.CriarLead(t, banco, etapas[0], "Carla Dias"),
		helpers.CriarLead(t, banco, etapas[0], "Diego Ramos"),
	}
	helpers.DesativarLead(t, banco, ids[3])

	primeira, err := repo.ListAtivos(ctx, domain.PaginacaoFiltro{Page: 1, Size: 2})
	if err != nil {
		t.Fatalf("listagem da primeira página falhou: %v", err)
	}
	if len(primeira) != 2 {
		t.Fatalf("primeira página com %d itens, esperado 2", len(primeira))
	}

	segunda, err := repo.ListAtivos(ctx, domain.PaginacaoFiltro{Page: 2, Size: 2})
	if err != nil {
		t.Fatalf("listagem da segunda página falhou: %v", err)
	}
	if len(segunda) != 1 {
		t.Fatalf("segunda página com %d itens, esperado 1", len(segunda))
	}

	// A ordenação é por criado_em decrescente e o Diego está inativo, então a
	// primeira página são os dois ativos mais recentes (Carla e Bruno) e a
	// segunda página é o mais antigo (Ana).
	if primeira[0].Nome != "Carla Dias" || segunda[0].Nome != "Ana Souza" {
		t.Errorf("ordem inesperada: %q, %q", primeira[0].Nome, segunda[0].Nome)
	}

	for _, item := range append(primeira, segunda...) {
		if item.ID == ids[3] {
			t.Error("lead inativo apareceu na listagem de ativos")
		}
	}
}

func TestBuscarPorNomeIgnoraInativos(t *testing.T) {
	repo, banco, _, etapas := cenario(t)
	ctx := context.Background()

	ativa := helpers.CriarLead(t, banco, etapas[0], "Maria Silva")
	inativo := helpers.CriarLead(t, banco, etapas[0], "Márcio Souza")
	helpers.DesativarLead(t, banco, inativo)

	// A busca ignora maiúsculas/minúsculas e não devolve inativos.
	encontrados, err := repo.Search(ctx, "SILVA", domain.PaginacaoFiltro{Page: 1, Size: 20})
	if err != nil {
		t.Fatalf("busca falhou: %v", err)
	}
	if len(encontrados) != 1 {
		t.Fatalf("busca devolveu %d itens, esperado 1", len(encontrados))
	}
	if encontrados[0].ID != ativa {
		t.Errorf("busca devolveu %s, esperado %s", encontrados[0].ID, ativa)
	}

	// A fixture não preenche telefone; completa na ativa para exercitar o
	// ramo de busca por telefone.
	if _, err := banco.Exec(`UPDATE lead SET telefone = '21988887777' WHERE id = $1`, ativa); err != nil {
		t.Fatalf("não consegui preencher o telefone do lead: %v", err)
	}

	telefones, err := repo.Search(ctx, "988887777", domain.PaginacaoFiltro{Page: 1, Size: 20})
	if err != nil {
		t.Fatalf("busca por telefone falhou: %v", err)
	}
	if len(telefones) != 1 {
		t.Fatalf("busca por telefone devolveu %d itens, esperado 1", len(telefones))
	}
	if telefones[0].ID != ativa {
		t.Errorf("busca por telefone devolveu %s, esperado %s", telefones[0].ID, ativa)
	}
}

func TestMoverParaEtapaGravaHistorico(t *testing.T) {
	repo, banco, _, etapas := cenario(t)
	ctx := context.Background()

	leadID := helpers.CriarLead(t, banco, etapas[0], "Ana Souza")

	if err := repo.MoverParaEtapa(ctx, leadID, etapas[0], etapas[1]); err != nil {
		t.Fatalf("movimentação falhou: %v", err)
	}

	salvo, err := repo.GetByID(ctx, leadID)
	if err != nil {
		t.Fatalf("busca após movimentação falhou: %v", err)
	}
	if salvo == nil {
		t.Fatal("lead sumiu depois da movimentação")
	}
	if salvo.EtapaID != etapas[1] {
		t.Errorf("etapa do lead %s, esperada %s", salvo.EtapaID, etapas[1])
	}

	var historico struct {
		Anterior uuid.UUID `db:"etapa_anterior_id"`
		Atual    uuid.UUID `db:"etapa_atual_id"`
	}
	if err := banco.Get(
		&historico,
		`SELECT etapa_anterior_id, etapa_atual_id FROM lead_historico WHERE lead_id = $1`,
		leadID,
	); err != nil {
		t.Fatalf("histórico não gravou a movimentação: %v", err)
	}
	if historico.Anterior != etapas[0] || historico.Atual != etapas[1] {
		t.Errorf(
			"histórico registrou %s -> %s, esperado %s -> %s",
			historico.Anterior,
			historico.Atual,
			etapas[0],
			etapas[1],
		)
	}

	var registros int
	if err := banco.Get(
		&registros,
		`SELECT COUNT(*) FROM lead_historico WHERE lead_id = $1`,
		leadID,
	); err != nil {
		t.Fatalf("não consegui contar o histórico: %v", err)
	}
	if registros != 1 {
		t.Errorf("histórico com %d registros, esperado 1", registros)
	}
}

func TestMoverLeadInexistenteDevolveNaoEncontrado(t *testing.T) {
	repo, _, _, etapas := cenario(t)

	err := repo.MoverParaEtapa(context.Background(), uuid.New(), etapas[0], etapas[1])
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("erro %v, esperado %v", err, domain.ErrNotFound)
	}
}

func TestMoverLeadComEtapaAnteriorDesatualizadaDevolveConflito(t *testing.T) {
	repo, banco, _, etapas := cenario(t)
	ctx := context.Background()

	leadID, err := repo.Create(ctx, novoLead(etapas[0], "Ana Souza"))
	if err != nil {
		t.Fatalf("criação falhou: %v", err)
	}

	err = repo.MoverParaEtapa(ctx, leadID, etapas[1], etapas[0])
	if !errors.Is(err, domain.ErrConflito) {
		t.Fatalf("erro %v, esperado %v", err, domain.ErrConflito)
	}

	var registros int
	if err := banco.Get(&registros, `SELECT COUNT(*) FROM lead_historico WHERE lead_id = $1`, leadID); err != nil {
		t.Fatalf("contagem do histórico falhou: %v", err)
	}
	if registros != 0 {
		t.Errorf("histórico com %d registros, esperado nenhum", registros)
	}
}

func TestContarPorFunilETapa(t *testing.T) {
	repo, banco, funil, etapas := cenario(t)
	ctx := context.Background()

	outroFunil := helpers.CriarFunil(t, banco, "Outro funil")
	etapasOutro := helpers.CriarEtapas(t, banco, outroFunil, "Novo")

	helpers.CriarLead(t, banco, etapas[0], "Ana Souza")
	helpers.CriarLead(t, banco, etapas[1], "Bruno Lima")
	helpers.CriarLead(t, banco, etapasOutro[0], "Carla Dias")

	totalEtapa, err := repo.CountByEtapa(ctx, etapas[0])
	if err != nil {
		t.Fatalf("contagem por etapa falhou: %v", err)
	}
	if totalEtapa != 1 {
		t.Errorf("contagem por etapa %d, esperado 1", totalEtapa)
	}

	totalFunil, err := repo.CountByFunil(ctx, funil)
	if err != nil {
		t.Fatalf("contagem por funil falhou: %v", err)
	}
	if totalFunil != 2 {
		t.Errorf("contagem por funil %d, esperado 2 (o lead do outro funil não conta)", totalFunil)
	}
}
