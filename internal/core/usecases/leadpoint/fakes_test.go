package leadpoint_test

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/lead"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/leads"
	"github.com/google/uuid"
)

var (
	_ portsout.LeadRepository  = (*repositorioLeads)(nil)
	_ portsout.EtapaRepository = (*repositorioEtapas)(nil)
)

type movimentacao struct {
	leadID   uuid.UUID
	anterior uuid.UUID
	atual    uuid.UUID
}

type repositorioLeads struct {
	itens         map[uuid.UUID]*lead.Lead
	movimentacoes []movimentacao
}

func novoRepositorioLeads() *repositorioLeads {
	return &repositorioLeads{itens: map[uuid.UUID]*lead.Lead{}}
}

func (r *repositorioLeads) Create(_ context.Context, item *lead.Lead) (uuid.UUID, error) {
	copia := *item
	r.itens[item.ID] = &copia

	return item.ID, nil
}

func (r *repositorioLeads) Update(_ context.Context, item *lead.Lead) error {
	atual, ok := r.itens[item.ID]
	if !ok {
		return nil
	}

	copia := *item
	copia.EtapaID = atual.EtapaID
	r.itens[item.ID] = &copia

	return nil
}

func (r *repositorioLeads) GetByID(_ context.Context, id uuid.UUID) (*lead.Lead, error) {
	item, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *item

	return &copia, nil
}

func (r *repositorioLeads) ListByFunil(_ context.Context, _ uuid.UUID) ([]*lead.Lead, error) {
	return nil, nil
}

func (r *repositorioLeads) ListByEtapa(_ context.Context, _ uuid.UUID) ([]*lead.Lead, error) {
	return nil, nil
}

func (r *repositorioLeads) ListAtivos(_ context.Context, _ domain.PaginacaoFiltro) ([]*lead.Lead, error) {
	return nil, nil
}

func (r *repositorioLeads) Search(_ context.Context, _ string, _ domain.PaginacaoFiltro) ([]*lead.Lead, error) {
	return nil, nil
}

func (r *repositorioLeads) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.itens, id)

	return nil
}

func (r *repositorioLeads) CountByEtapa(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (r *repositorioLeads) CountByFunil(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (r *repositorioLeads) MoverParaEtapa(_ context.Context, leadID, anterior, atual uuid.UUID) error {
	item, ok := r.itens[leadID]
	if !ok || item.EtapaID != anterior {
		return domain.ErroConflito("o lead mudou de etapa durante a operação")
	}

	item.EtapaID = atual
	r.movimentacoes = append(r.movimentacoes, movimentacao{leadID: leadID, anterior: anterior, atual: atual})

	return nil
}

type repositorioEtapas struct {
	itens map[uuid.UUID]*lead.Etapa
}

func novoRepositorioEtapas() *repositorioEtapas {
	return &repositorioEtapas{itens: map[uuid.UUID]*lead.Etapa{}}
}

func (r *repositorioEtapas) Create(_ context.Context, etapa *lead.Etapa) (uuid.UUID, error) {
	copia := *etapa
	r.itens[etapa.EtapaID] = &copia

	return etapa.EtapaID, nil
}

func (r *repositorioEtapas) Update(_ context.Context, _ *lead.Etapa) error {
	return nil
}

func (r *repositorioEtapas) GetByID(_ context.Context, id uuid.UUID) (*lead.Etapa, error) {
	etapa, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *etapa

	return &copia, nil
}

func (r *repositorioEtapas) ListByFunilID(_ context.Context, funilID uuid.UUID) ([]*lead.Etapa, error) {
	var etapas []*lead.Etapa
	for _, etapa := range r.itens {
		if etapa.FunilID == funilID {
			copia := *etapa
			etapas = append(etapas, &copia)
		}
	}

	return etapas, nil
}

func (r *repositorioEtapas) ListByFunilOrdenado(_ context.Context, _ uuid.UUID) ([]*lead.Etapa, error) {
	return nil, nil
}

func (r *repositorioEtapas) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (r *repositorioEtapas) Reordenar(_ context.Context, _ uuid.UUID, etapas []*lead.Etapa) error {
	for i, etapa := range etapas {
		r.itens[etapa.EtapaID].Ordem = i + 1
	}

	return nil
}

func (r *repositorioEtapas) GetNextEtapa(_ context.Context, _ uuid.UUID) (*lead.Etapa, error) {
	return nil, nil
}

func (r *repositorioEtapas) GetPreviousEtapa(_ context.Context, _ uuid.UUID) (*lead.Etapa, error) {
	return nil, nil
}

func (r *repositorioEtapas) ExistsByFunil(_ context.Context, _ uuid.UUID) (bool, error) {
	return false, nil
}
